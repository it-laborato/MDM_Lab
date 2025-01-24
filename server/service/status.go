package service

import (
	"context"

	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

////////////////////////////////////////////////////////////////////////////////
// Status Result Store
////////////////////////////////////////////////////////////////////////////////

type statusResponse struct {
	Err error `json:"error,omitempty"`
}

func (m statusResponse) error() error { return m.Err }

func statusResultStoreEndpoint(ctx context.Context, req interface{}, svc mdmlab.Service) (errorer, error) {
	var resp statusResponse
	if err := svc.StatusResultStore(ctx); err != nil {
		resp.Err = err
	}
	return resp, nil
}

func (svc *Service) StatusResultStore(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppConfig{}, mdmlab.ActionRead); err != nil {
		return err
	}

	return svc.resultStore.HealthCheck()
}

////////////////////////////////////////////////////////////////////////////////
// Status Live Query
////////////////////////////////////////////////////////////////////////////////

func statusLiveQueryEndpoint(ctx context.Context, req interface{}, svc mdmlab.Service) (errorer, error) {
	var resp statusResponse
	if err := svc.StatusLiveQuery(ctx); err != nil {
		resp.Err = err
	}
	return resp, nil
}

func (svc *Service) StatusLiveQuery(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppConfig{}, mdmlab.ActionRead); err != nil {
		return err
	}

	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieve app config")
	}

	if cfg.ServerSettings.LiveQueryDisabled {
		return ctxerr.Wrap(ctx, mdmlab.NewPermissionError("disabled by administrator"))
	}

	return svc.StatusResultStore(ctx)
}
