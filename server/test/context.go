package test

import (
	"context"

	authz_ctx "github.com:it-laborato/MDM_Lab/server/contexts/authz"
	hostctx "github.com:it-laborato/MDM_Lab/server/contexts/host"
	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

// UserContext returns a new context with the provided user as the viewer.
func UserContext(ctx context.Context, user *mdmlab.User) context.Context {
	return viewer.NewContext(ctx, viewer.Viewer{User: user})
}

// HostContext returns a new context with the provided host as the
// device-authenticated host.
func HostContext(ctx context.Context, host *mdmlab.Host) context.Context {
	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, authzCtx)
	ctx = hostctx.NewContext(ctx, host)
	if ac, ok := authz_ctx.FromContext(ctx); ok {
		ac.SetAuthnMethod(authz_ctx.AuthnDeviceToken)
	}
	return ctx
}
