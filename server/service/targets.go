package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

////////////////////////////////////////////////////////////////////////////////
// Search Targets
////////////////////////////////////////////////////////////////////////////////

type searchTargetsRequest struct {
	// MatchQuery is the query SQL
	MatchQuery string `json:"query"`
	// QueryID is the ID of a saved query to run (used to determine if this is a
	// query that observers can run).
	QueryID *uint `json:"query_id"`
	// Selected is the list of IDs that are already selected on the caller side
	// (e.g. the UI), so those are IDs that will be omitted from the returned
	// payload.
	Selected mdmlab.HostTargets `json:"selected"`
}

type labelSearchResult struct {
	*mdmlab.Label
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

type teamSearchResult struct {
	*mdmlab.Team
	DisplayText string `json:"display_text"`
	Count       int    `json:"count"`
}

func (t teamSearchResult) MarshalJSON() ([]byte, error) {
	x := struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		mdmlab.TeamConfig
		UserCount   int                   `json:"user_count"`
		Users       []mdmlab.TeamUser      `json:"users,omitempty"`
		HostCount   int                   `json:"host_count"`
		Hosts       []mdmlab.HostResponse  `json:"hosts,omitempty"`
		Secrets     []*mdmlab.EnrollSecret `json:"secrets,omitempty"`
		DisplayText string                `json:"display_text"`
		Count       int                   `json:"count"`
	}{
		ID:          t.ID,
		CreatedAt:   t.CreatedAt,
		Name:        t.Name,
		Description: t.Description,
		TeamConfig:  t.Config,
		UserCount:   t.UserCount,
		Users:       t.Users,
		HostCount:   t.HostCount,
		Hosts:       mdmlab.HostResponsesForHostsCheap(t.Hosts),
		Secrets:     t.Secrets,
		DisplayText: t.DisplayText,
		Count:       t.Count,
	}

	return json.Marshal(x)
}

func (t *teamSearchResult) UnmarshalJSON(b []byte) error {
	var x struct {
		ID          uint      `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		mdmlab.TeamConfig
		UserCount   int                   `json:"user_count"`
		Users       []mdmlab.TeamUser      `json:"users,omitempty"`
		HostCount   int                   `json:"host_count"`
		Hosts       []mdmlab.Host          `json:"hosts,omitempty"`
		Secrets     []*mdmlab.EnrollSecret `json:"secrets,omitempty"`
		DisplayText string                `json:"display_text"`
		Count       int                   `json:"count"`
	}

	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}

	*t = teamSearchResult{
		Team: &mdmlab.Team{
			ID:          x.ID,
			CreatedAt:   x.CreatedAt,
			Name:        x.Name,
			Description: x.Description,
			Config:      x.TeamConfig,
			UserCount:   x.UserCount,
			Users:       x.Users,
			HostCount:   x.HostCount,
			Hosts:       x.Hosts,
			Secrets:     x.Secrets,
		},
		DisplayText: x.DisplayText,
		Count:       x.Count,
	}

	return nil
}

type targetsData struct {
	Hosts  []*mdmlab.HostResponse `json:"hosts"`
	Labels []labelSearchResult   `json:"labels"`
	Teams  []teamSearchResult    `json:"teams"`
}

type searchTargetsResponse struct {
	Targets                *targetsData `json:"targets,omitempty"`
	TargetsCount           uint         `json:"targets_count"`
	TargetsOnline          uint         `json:"targets_online"`
	TargetsOffline         uint         `json:"targets_offline"`
	TargetsMissingInAction uint         `json:"targets_missing_in_action"`
	Err                    error        `json:"error,omitempty"`
}

func (r searchTargetsResponse) error() error { return r.Err }

func searchTargetsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*searchTargetsRequest)

	results, err := svc.SearchTargets(ctx, req.MatchQuery, req.QueryID, req.Selected)
	if err != nil {
		return searchTargetsResponse{Err: err}, nil
	}

	targets := &targetsData{
		Hosts:  []*mdmlab.HostResponse{},
		Labels: []labelSearchResult{},
		Teams:  []teamSearchResult{},
	}

	for _, host := range results.Hosts {
		targets.Hosts = append(targets.Hosts, mdmlab.HostResponseForHostCheap(host))
	}

	for _, label := range results.Labels {
		targets.Labels = append(targets.Labels,
			labelSearchResult{
				Label:       label,
				DisplayText: label.Name,
				Count:       label.HostCount,
			},
		)
	}

	for _, team := range results.Teams {
		targets.Teams = append(targets.Teams,
			teamSearchResult{
				Team:        team,
				DisplayText: team.Name,
				Count:       team.HostCount,
			},
		)
	}

	metrics, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
	if err != nil {
		return searchTargetsResponse{Err: err}, nil
	}

	return searchTargetsResponse{
		Targets:                targets,
		TargetsCount:           metrics.TotalHosts,
		TargetsOnline:          metrics.OnlineHosts,
		TargetsOffline:         metrics.OfflineHosts,
		TargetsMissingInAction: metrics.MissingInActionHosts,
	}, nil
}

func (svc *Service) SearchTargets(ctx context.Context, matchQuery string, queryID *uint, targets mdmlab.HostTargets) (*mdmlab.TargetSearchResults, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Target{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := mdmlab.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	results := &mdmlab.TargetSearchResults{}

	hosts, err := svc.ds.SearchHosts(ctx, filter, matchQuery, targets.HostIDs...)
	if err != nil {
		return nil, err
	}

	results.Hosts = append(results.Hosts, hosts...)

	labels, err := svc.ds.SearchLabels(ctx, filter, matchQuery, targets.LabelIDs...)
	if err != nil {
		return nil, err
	}
	results.Labels = labels

	teams, err := svc.ds.SearchTeams(ctx, filter, matchQuery, targets.TeamIDs...)
	if err != nil {
		return nil, err
	}
	results.Teams = teams

	return results, nil
}

func (svc *Service) CountHostsInTargets(ctx context.Context, queryID *uint, targets mdmlab.HostTargets) (*mdmlab.TargetMetrics, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Target{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		query, err := svc.ds.Query(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = query.ObserverCanRun
	}

	filter := mdmlab.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	metrics, err := svc.ds.CountHostsInTargets(ctx, filter, targets, svc.clock.Now())
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}

type countTargetsRequest struct {
	Selected mdmlab.HostTargets `json:"selected"`
	QueryID  *uint             `json:"query_id"`
}

type countTargetsResponse struct {
	TargetsCount   uint  `json:"targets_count"`
	TargetsOnline  uint  `json:"targets_online"`
	TargetsOffline uint  `json:"targets_offline"`
	Err            error `json:"error,omitempty"`
}

func (r countTargetsResponse) error() error { return r.Err }

func countTargetsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*countTargetsRequest)

	counts, err := svc.CountHostsInTargets(ctx, req.QueryID, req.Selected)
	if err != nil {
		return searchTargetsResponse{Err: err}, nil
	}

	return countTargetsResponse{
		TargetsCount:   counts.TotalHosts,
		TargetsOnline:  counts.OnlineHosts,
		TargetsOffline: counts.OfflineHosts,
	}, nil
}
