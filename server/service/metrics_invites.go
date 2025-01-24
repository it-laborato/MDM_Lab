package service

import (
	"context"
	"fmt"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

func (mw metricsMiddleware) InviteNewUser(ctx context.Context, payload mdmlab.InvitePayload) (*mdmlab.Invite, error) {
	var (
		invite *mdmlab.Invite
		err    error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "InviteNewUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	invite, err = mw.Service.InviteNewUser(ctx, payload)
	return invite, err
}

func (mw metricsMiddleware) DeleteInvite(ctx context.Context, id uint) error {
	var (
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteInvite", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteInvite(ctx, id)
	return err
}

func (mw metricsMiddleware) ListInvites(ctx context.Context, opt mdmlab.ListOptions) ([]*mdmlab.Invite, error) {
	var (
		invites []*mdmlab.Invite
		err     error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Invites", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	invites, err = mw.Service.ListInvites(ctx, opt)
	return invites, err
}

func (mw metricsMiddleware) VerifyInvite(ctx context.Context, token string) (*mdmlab.Invite, error) {
	var (
		err    error
		invite *mdmlab.Invite
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "VerifyInvite", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	invite, err = mw.Service.VerifyInvite(ctx, token)
	return invite, err
}
