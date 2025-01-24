package capabilities

import (
	"context"
	"net/http"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

type key int

const capabilitiesKey key = 0

// NewContext creates a new context with the given capabilities.
func NewContext(ctx context.Context, r *http.Request) context.Context {
	capabilities := mdmlab.CapabilityMap{}
	capabilities.PopulateFromString(r.Header.Get(mdmlab.CapabilitiesHeader))
	return context.WithValue(ctx, capabilitiesKey, capabilities)
}

// FromContext returns the capabilities in the request if present.
func FromContext(ctx context.Context) (mdmlab.CapabilityMap, bool) {
	v, ok := ctx.Value(capabilitiesKey).(mdmlab.CapabilityMap)
	return v, ok
}
