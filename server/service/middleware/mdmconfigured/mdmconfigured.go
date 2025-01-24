// Package mdmconfigured implements middleware functions for the supported platform-specific MDM
// solutions to ensure MDM is configured and fail fast before reaching the handler if that is not the case.
package mdmconfigured

import (
	"context"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/go-kit/kit/endpoint"
)

type Middleware struct {
	svc mdmlab.Service
}

func NewMDMConfigMiddleware(svc mdmlab.Service) *Middleware {
	return &Middleware{svc: svc}
}

func (m *Middleware) VerifyAppleOrWindowsMDM() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if err := m.svc.VerifyMDMAppleOrWindowsConfigured(ctx); err != nil {
				return nil, err
			}

			return next(ctx, req)
		}
	}
}

func (m *Middleware) VerifyAppleMDM() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if err := m.svc.VerifyMDMAppleConfigured(ctx); err != nil {
				return nil, err
			}

			return next(ctx, req)
		}
	}
}

func (m *Middleware) VerifyWindowsMDM() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if err := m.svc.VerifyMDMWindowsConfigured(ctx); err != nil {
				return nil, err
			}

			return next(ctx, req)
		}
	}
}
