package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/contexts/logging"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignRequest struct {
	QuerySQL string            `json:"query"`
	QueryID  *uint             `json:"query_id"`
	Selected mdmlab.HostTargets `json:"selected"`
}

type createDistributedQueryCampaignResponse struct {
	Campaign *mdmlab.DistributedQueryCampaign `json:"campaign,omitempty"`
	Err      error                           `json:"error,omitempty"`
}

func (r createDistributedQueryCampaignResponse) error() error { return r.Err }

func createDistributedQueryCampaignEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*createDistributedQueryCampaignRequest)
	campaign, err := svc.NewDistributedQueryCampaign(ctx, req.QuerySQL, req.QueryID, req.Selected)
	if err != nil {
		return createDistributedQueryCampaignResponse{Err: err}, nil
	}
	return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
}

func (svc *Service) NewDistributedQueryCampaign(ctx context.Context, queryString string, queryID *uint, targets mdmlab.HostTargets) (*mdmlab.DistributedQueryCampaign, error) {
	if err := svc.StatusLiveQuery(ctx); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}

	if queryID == nil && strings.TrimSpace(queryString) == "" {
		return nil, mdmlab.NewInvalidArgumentError("query", "one of query or query_id must be specified")
	}

	var query *mdmlab.Query
	var err error
	if queryID != nil {
		query, err = svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		queryString = query.Query
	} else {
		if err := svc.authz.Authorize(ctx, &mdmlab.Query{}, mdmlab.ActionRunNew); err != nil {
			return nil, err
		}
		query = &mdmlab.Query{
			Name:     fmt.Sprintf("distributed_%s_%d", vc.Email(), time.Now().UnixNano()),
			Query:    queryString,
			Saved:    false,
			AuthorID: ptr.Uint(vc.UserID()),
			// We must set a valid value for this field, even if unused by live queries.
			Logging: mdmlab.LoggingSnapshot,
		}
		if err := query.Verify(); err != nil {
			return nil, ctxerr.Wrap(ctx, &mdmlab.BadRequestError{
				Message: fmt.Sprintf("query payload verification: %s", err),
			})
		}
		query, err = svc.ds.NewQuery(ctx, query)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "new query")
		}
	}

	tq := &mdmlab.TargetedQuery{Query: query, HostTargets: targets}
	if err := svc.authz.Authorize(ctx, tq, mdmlab.ActionRun); err != nil {
		return nil, err
	}

	filter := mdmlab.TeamFilter{User: vc.User, IncludeObserver: query.ObserverCanRun}

	campaign, err := svc.ds.NewDistributedQueryCampaign(ctx, &mdmlab.DistributedQueryCampaign{
		QueryID: query.ID,
		Status:  mdmlab.QueryWaiting,
		UserID:  vc.UserID(),
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new campaign")
	}

	defer func() {
		var numHosts uint
		if campaign != nil {
			numHosts = campaign.Metrics.TotalHosts
		}
		logging.WithExtras(ctx, "sql", queryString, "query_id", queryID, "numHosts", numHosts)
	}()

	// Add host targets
	for _, hid := range targets.HostIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &mdmlab.DistributedQueryCampaignTarget{
			Type:                       mdmlab.TargetHost,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   hid,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding host target")
		}
	}

	// Add label targets
	for _, lid := range targets.LabelIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &mdmlab.DistributedQueryCampaignTarget{
			Type:                       mdmlab.TargetLabel,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   lid,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding label target")
		}
	}

	// Add team targets
	for _, tid := range targets.TeamIDs {
		_, err = svc.ds.NewDistributedQueryCampaignTarget(ctx, &mdmlab.DistributedQueryCampaignTarget{
			Type:                       mdmlab.TargetTeam,
			DistributedQueryCampaignID: campaign.ID,
			TargetID:                   tid,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "adding team target")
		}
	}

	hostIDs, err := svc.ds.HostIDsInTargets(ctx, filter, targets)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get target IDs")
	}

	if len(hostIDs) == 0 {
		return nil, &mdmlab.BadRequestError{
			Message: "no hosts targeted",
		}
	}

	// Metrics are used for total hosts targeted for the activity feed.
	campaign.Metrics, err = svc.ds.CountHostsInTargets(ctx, filter, targets, time.Now())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "counting hosts")
	}

	err = svc.liveQueryStore.RunQuery(fmt.Sprint(campaign.ID), queryString, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "run query")
	}

	return campaign, nil
}

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign By Names
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignByIdentifierRequest struct {
	QuerySQL string                                       `json:"query"`
	QueryID  *uint                                        `json:"query_id"`
	Selected distributedQueryCampaignTargetsByIdentifiers `json:"selected"`
}

type distributedQueryCampaignTargetsByIdentifiers struct {
	Labels []string `json:"labels"`
	// list of hostnames, UUIDs, and/or hardware serials
	Hosts []string `json:"hosts"`
}

func createDistributedQueryCampaignByIdentifierEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*createDistributedQueryCampaignByIdentifierRequest)
	campaign, err := svc.NewDistributedQueryCampaignByIdentifiers(ctx, req.QuerySQL, req.QueryID, req.Selected.Hosts, req.Selected.Labels)
	if err != nil {
		return createDistributedQueryCampaignResponse{Err: err}, nil
	}
	return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
}

func (svc *Service) NewDistributedQueryCampaignByIdentifiers(ctx context.Context, queryString string, queryID *uint, hostIdentifiers []string, labels []string) (*mdmlab.DistributedQueryCampaign, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}
	filter := mdmlab.TeamFilter{User: vc.User, IncludeObserver: true}

	hostIDs, err := svc.ds.HostIDsByIdentifier(ctx, filter, hostIdentifiers)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding host IDs")
	}

	if err := svc.authz.Authorize(ctx, &mdmlab.Label{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}
	labelMap, err := svc.ds.LabelIDsByName(ctx, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding label IDs")
	}

	// DetectMissingLabels will return the list of labels that are not found in the database
	// These labels are considered invalid
	invalidLabels := mdmlab.DetectMissingLabels(labelMap, labels)
	if len(invalidLabels) > 0 {
		return nil, ctxerr.Wrap(ctx, &mdmlab.BadRequestError{
			Message: fmt.Sprintf("%s %s.", mdmlab.InvalidLabelSpecifiedErrMsg, strings.Join(invalidLabels, ", ")),
		}, "invalid labels")
	}

	var labelIDs []uint
	for _, labelID := range labelMap {
		labelIDs = append(labelIDs, labelID)
	}

	targets := mdmlab.HostTargets{HostIDs: hostIDs, LabelIDs: labelIDs}
	return svc.NewDistributedQueryCampaign(ctx, queryString, queryID, targets)
}
