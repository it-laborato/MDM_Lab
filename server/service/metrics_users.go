package service

import (
	"context"
	"fmt"
	"time"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

func (mw metricsMiddleware) CreateUserFromInvite(ctx context.Context, p mdmlab.UserPayload) (*mdmlab.User, error) {
	var (
		user *mdmlab.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "CreateUserFromInvite", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.CreateUserFromInvite(ctx, p)
	return user, err
}

func (mw metricsMiddleware) ModifyUser(ctx context.Context, userID uint, p mdmlab.UserPayload) (*mdmlab.User, error) {
	var (
		user *mdmlab.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.ModifyUser(ctx, userID, p)
	return user, err
}

func (mw metricsMiddleware) User(ctx context.Context, id uint) (*mdmlab.User, error) {
	var (
		user *mdmlab.User
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "User", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.User(ctx, id)
	return user, err
}

func (mw metricsMiddleware) ListUsers(ctx context.Context, opt mdmlab.UserListOptions) ([]*mdmlab.User, error) {

	var (
		users []*mdmlab.User
		err   error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "Users", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	users, err = mw.Service.ListUsers(ctx, opt)
	return users, err
}

func (mw metricsMiddleware) AuthenticatedUser(ctx context.Context) (*mdmlab.User, error) {
	var (
		user *mdmlab.User
		err  error
	)

	defer func(begin time.Time) {
		lvs := []string{"method", "AuthenticatedUser", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	user, err = mw.Service.AuthenticatedUser(ctx)
	return user, err
}

func (mw metricsMiddleware) ChangePassword(ctx context.Context, oldPass, newPass string) error {
	var err error

	defer func(begin time.Time) {
		lvs := []string{"method", "ChangePassword", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = mw.Service.ChangePassword(ctx, oldPass, newPass)
	return err
}

func (mw metricsMiddleware) ResetPassword(ctx context.Context, token, password string) error {
	var err error

	defer func(begin time.Time) {
		lvs := []string{"method", "ResetPassword", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = mw.Service.ResetPassword(ctx, token, password)
	return err
}

func (mw metricsMiddleware) RequestPasswordReset(ctx context.Context, email string) error {
	var err error

	defer func(begin time.Time) {
		lvs := []string{"method", "RequestPasswordReset", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	err = mw.Service.RequestPasswordReset(ctx, email)
	return err
}
