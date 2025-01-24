package service

import (
	"context"
	"fmt"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

func (mw metricsMiddleware) ModifyLabel(ctx context.Context, id uint, p mdmlab.ModifyLabelPayload) (*mdmlab.Label, []uint, error) {
	var (
		lic  *mdmlab.Label
		hids []uint
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyLabel", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	lic, hids, err = mw.Service.ModifyLabel(ctx, id, p)
	return lic, hids, err
}
