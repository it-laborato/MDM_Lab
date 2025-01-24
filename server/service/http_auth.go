package service

import (
	"context"
	"net/http"

	"github.com:it-laborato/MDM_Lab/server/contexts/logging"
	"github.com:it-laborato/MDM_Lab/server/contexts/token"
	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"

	kithttp "github.com/go-kit/kit/transport/http"
)

// setRequestsContexts updates the request with necessary context values for a request
func setRequestsContexts(svc mdmlab.Service) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		bearer := token.FromHTTPRequest(r)
		ctx = token.NewContext(ctx, bearer)
		if bearer != "" {
			v, err := authViewer(ctx, string(bearer), svc)
			if err == nil {
				ctx = viewer.NewContext(ctx, *v)
			}
		}

		ctx = logging.NewContext(ctx, &logging.LoggingContext{})
		ctx = logging.WithStartTime(ctx)
		return ctx
	}
}
