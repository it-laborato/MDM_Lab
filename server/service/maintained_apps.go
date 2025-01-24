package service

import (
	"context"
	"errors"
	"time"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mdm/maintainedapps"
)

type addMDMlabMaintainedAppRequest struct {
	TeamID            *uint    `json:"team_id"`
	AppID             uint     `json:"mdmlab_maintained_app_id"`
	InstallScript     string   `json:"install_script"`
	PreInstallQuery   string   `json:"pre_install_query"`
	PostInstallScript string   `json:"post_install_script"`
	SelfService       bool     `json:"self_service"`
	UninstallScript   string   `json:"uninstall_script"`
	LabelsIncludeAny  []string `json:"labels_include_any"`
	LabelsExcludeAny  []string `json:"labels_exclude_any"`
}

type addMDMlabMaintainedAppResponse struct {
	SoftwareTitleID uint  `json:"software_title_id,omitempty"`
	Err             error `json:"error,omitempty"`
}

func (r addMDMlabMaintainedAppResponse) error() error { return r.Err }

func addMDMlabMaintainedAppEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*addMDMlabMaintainedAppRequest)
	ctx, cancel := context.WithTimeout(ctx, maintainedapps.InstallerTimeout)
	defer cancel()
	titleId, err := svc.AddMDMlabMaintainedApp(
		ctx,
		req.TeamID,
		req.AppID,
		req.InstallScript,
		req.PreInstallQuery,
		req.PostInstallScript,
		req.UninstallScript,
		req.SelfService,
		req.LabelsIncludeAny,
		req.LabelsExcludeAny,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = mdmlab.NewGatewayTimeoutError("Couldn't upload. Request timeout. Please make sure your server and load balancer timeout is long enough.", err)
		}

		return &addMDMlabMaintainedAppResponse{Err: err}, nil
	}
	return &addMDMlabMaintainedAppResponse{SoftwareTitleID: titleId}, nil
}

func (svc *Service) AddMDMlabMaintainedApp(ctx context.Context, teamID *uint, appID uint, installScript, preInstallQuery, postInstallScript, uninstallScript string, selfService bool, labelsIncludeAny, labelsExcludeAny []string) (uint, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return 0, mdmlab.ErrMissingLicense
}

type editMDMlabMaintainedAppRequest struct {
	TeamID            *uint    `json:"team_id"`
	AppID             uint     `json:"mdmlab_maintained_app_id"`
	InstallScript     string   `json:"install_script"`
	PreInstallQuery   string   `json:"pre_install_query"`
	PostInstallScript string   `json:"post_install_script"`
	SelfService       bool     `json:"self_service"`
	UninstallScript   string   `json:"uninstall_script"`
	LabelsIncludeAny  []string `json:"labels_include_any"`
	LabelsExcludeAny  []string `json:"labels_exclude_any"`
}

func editMDMlabMaintainedAppEndpoint(ctx context.Context, request any, svc mdmlab.Service) (errorer, error) {
	// TODO: implement this

	return nil, errors.New("not implemented")
}

type listMDMlabMaintainedAppsRequest struct {
	mdmlab.ListOptions
	TeamID *uint `query:"team_id,optional"`
}

type listMDMlabMaintainedAppsResponse struct {
	Count               int                       `json:"count"`
	AppsUpdatedAt       *time.Time                `json:"apps_updated_at"`
	MDMlabMaintainedApps []mdmlab.MaintainedApp     `json:"mdmlab_maintained_apps"`
	Meta                *mdmlab.PaginationMetadata `json:"meta"`
	Err                 error                     `json:"error,omitempty"`
}

func (r listMDMlabMaintainedAppsResponse) error() error { return r.Err }

func listMDMlabMaintainedAppsEndpoint(ctx context.Context, request any, svc mdmlab.Service) (errorer, error) {
	req := request.(*listMDMlabMaintainedAppsRequest)

	req.IncludeMetadata = true

	apps, meta, err := svc.ListMDMlabMaintainedApps(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return listMDMlabMaintainedAppsResponse{Err: err}, nil
	}

	var latest time.Time
	for _, app := range apps {
		if app.UpdatedAt != nil && !app.UpdatedAt.IsZero() && app.UpdatedAt.After(latest) {
			latest = *app.UpdatedAt
		}
	}

	listResp := listMDMlabMaintainedAppsResponse{
		MDMlabMaintainedApps: apps,
		Count:               int(meta.TotalResults), //nolint:gosec // dismiss G115
		Meta:                meta,
	}
	if !latest.IsZero() {
		listResp.AppsUpdatedAt = &latest
	}

	return listResp, nil
}

func (svc *Service) ListMDMlabMaintainedApps(ctx context.Context, teamID *uint, opts mdmlab.ListOptions) ([]mdmlab.MaintainedApp, *mdmlab.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, mdmlab.ErrMissingLicense
}

type getMDMlabMaintainedAppRequest struct {
	AppID uint `url:"app_id"`
}

type getMDMlabMaintainedAppResponse struct {
	MDMlabMaintainedApp *mdmlab.MaintainedApp `json:"mdmlab_maintained_app"`
	Err                error                `json:"error,omitempty"`
}

func (r getMDMlabMaintainedAppResponse) error() error { return r.Err }

func getMDMlabMaintainedApp(ctx context.Context, request any, svc mdmlab.Service) (errorer, error) {
	req := request.(*getMDMlabMaintainedAppRequest)

	app, err := svc.GetMDMlabMaintainedApp(ctx, req.AppID)
	if err != nil {
		return getMDMlabMaintainedAppResponse{Err: err}, nil
	}

	return getMDMlabMaintainedAppResponse{MDMlabMaintainedApp: app}, nil
}

func (svc *Service) GetMDMlabMaintainedApp(ctx context.Context, appID uint) (*mdmlab.MaintainedApp, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}
