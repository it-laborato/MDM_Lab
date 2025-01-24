package service

import (
	"context"
	"fmt"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

func (mw metricsMiddleware) NewAppConfig(ctx context.Context, p mdmlab.AppConfig) (*mdmlab.AppConfig, error) {
	var (
		info *mdmlab.AppConfig
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "NewOrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.NewAppConfig(ctx, p)
	return info, err
}

func (mw metricsMiddleware) AppConfigObfuscated(ctx context.Context) (*mdmlab.AppConfig, error) {
	var (
		info *mdmlab.AppConfig
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "OrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.AppConfigObfuscated(ctx)
	return info, err
}

func (mw metricsMiddleware) ModifyAppConfig(ctx context.Context, p []byte, applyOpts mdmlab.ApplySpecOptions) (*mdmlab.AppConfig, error) {
	var (
		info *mdmlab.AppConfig
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyOrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.ModifyAppConfig(ctx, p, applyOpts)
	return info, err
}
