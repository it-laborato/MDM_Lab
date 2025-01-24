package service

import (
	"context"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
)

func TestGlobalScheduleAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	//
	// All global schedule query methods use queries datastore methods.
	//

	ds.QueryFunc = func(ctx context.Context, id uint) (*mdmlab.Query, error) {
		return &mdmlab.Query{
			Name:  "foobar",
			Query: "SELECT 1;",
		}, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *mdmlab.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opt mdmlab.ListQueryOptions) ([]*mdmlab.Query, int, *mdmlab.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *mdmlab.Query, opts ...mdmlab.OptionalArg) (*mdmlab.Query, error) {
		return &mdmlab.Query{}, nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *mdmlab.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			name:            "global admin",
			user:            &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			shouldFailWrite: false,
			shouldFailRead:  false,
		},
		{
			name:            "global maintainer",
			user:            &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			shouldFailWrite: false,
			shouldFailRead:  false,
		},
		{
			name:            "global observer",
			user:            &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
		{
			name:            "team admin",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
		{
			name:            "team maintainer",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
		{
			name:            "team observer",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			shouldFailWrite: true,
			shouldFailRead:  false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetGlobalScheduledQueries(ctx, mdmlab.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GlobalScheduleQuery(ctx, &mdmlab.ScheduledQuery{
				Name:      "query",
				QueryName: "query",
				Interval:  10,
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ModifyGlobalScheduledQueries(ctx, 1, mdmlab.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteGlobalScheduledQueries(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
