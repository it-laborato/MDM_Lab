package service

import (
	"context"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
)

func TestTeamScheduleAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListQueriesFunc = func(ctx context.Context, opt mdmlab.ListQueryOptions) ([]*mdmlab.Query, int, *mdmlab.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*mdmlab.Query, error) {
		if id == 99 { // for testing modify and delete of a schedule
			return &mdmlab.Query{
				Name:   "foobar",
				Query:  "SELECT 1;",
				TeamID: ptr.Uint(1),
			}, nil
		}
		return &mdmlab.Query{ // for testing creation of a schedule
			Name:  "foobar",
			Query: "SELECT 1;",
			// TeamID is set to nil because a query must be global to be able to be
			// scheduled on a team by the deprecated APIs.
			TeamID: nil,
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
			"global admin",
			&mdmlab.User{
				GlobalRole: ptr.String(mdmlab.RoleAdmin),
			},
			false,
			false,
		},
		{
			"global maintainer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			true,
			false, // global observer can view all queries and scheduled queries.
		},
		{
			"global observer+",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			true,
			false, // global observer+ can view all queries and scheduled queries.
		},
		{
			"global gitops",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			false,
			false,
		},
		{
			"team admin, belongs to team",
			&mdmlab.User{
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 1},
					Role: mdmlab.RoleAdmin,
				}},
			},
			false,
			false,
		},
		{
			"team maintainer, belongs to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			false,
			false,
		},
		{
			"team observer, belongs to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			true,
			false,
		},
		{
			"team observer+, belongs to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			true,
			false,
		},
		{
			"team gitops, belongs to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			false,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team admin, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
			true,
			true,
		},
		{
			"team observer+, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserverPlus}}},
			true,
			true,
		},
		{
			"team gitops, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleGitOps}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetTeamScheduledQueries(ctx, 1, mdmlab.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.TeamScheduleQuery(ctx, 1, &mdmlab.ScheduledQuery{Interval: 10})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.ModifyTeamScheduledQueries(ctx, 1, 99, mdmlab.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteTeamScheduledQueries(ctx, 1, 99)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}
