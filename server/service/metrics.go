package service

import (
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/go-kit/kit/metrics"
)

type metricsMiddleware struct {
	mdmlab.Service
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

// NewMetricsService service takes an existing service and wraps it
// with instrumentation middleware.
func NewMetricsService(
	svc mdmlab.Service,
	requestCount metrics.Counter,
	requestLatency metrics.Histogram,
) mdmlab.Service {
	return metricsMiddleware{
		Service:        svc,
		requestCount:   requestCount,
		requestLatency: requestLatency,
	}
}
