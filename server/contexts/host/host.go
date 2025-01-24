// Package host enables setting and reading
// the current host from context
package host

import (
	"context"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

type key int

const hostKey key = 0

// NewContext returns a new context carrying the current osquery host.
func NewContext(ctx context.Context, host *mdmlab.Host) context.Context {
	return context.WithValue(ctx, hostKey, host)
}

// FromContext extracts the osquery host from context if present.
func FromContext(ctx context.Context) (*mdmlab.Host, bool) {
	host, ok := ctx.Value(hostKey).(*mdmlab.Host)
	return host, ok
}
