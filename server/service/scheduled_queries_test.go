package service

import (
	"context"
	"testing"

	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com:it-laborato/MDM_Lab/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduledQueriesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts mdmlab.ListOptions) ([]*mdmlab.ScheduledQuery, error) {
		return nil, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, sq *mdmlab.ScheduledQuery, opts ...mdmlab.OptionalArg) (*mdmlab.ScheduledQuery, error) {
		return sq, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*mdmlab.Query, error) {
		return &mdmlab.Query{}, nil
	}
	ds.ScheduledQueryFunc = func(ctx context.Context, id uint) (*mdmlab.ScheduledQuery, error) {
		return &mdmlab.ScheduledQuery{}, nil
	}
	ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *mdmlab.ScheduledQuery) (*mdmlab.ScheduledQuery, error) {
		return sq, nil
	}
	ds.DeleteScheduledQueryFunc = func(ctx context.Context, id uint) error {
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
			shouldFailRead:  true,
		},
		{
			name:            "global observer+",
			user:            &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "global gitops",
			user:            &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			shouldFailWrite: false,
			shouldFailRead:  false, // Global gitops can read packs (exception to the write only rule)
		},
		// Team users cannot read or write scheduled queries using the below service APIs.
		// Team users must use the "Team" endpoints (GetTeamScheduledQueries, TeamScheduleQuery,
		// ModifyTeamScheduledQueries and DeleteTeamScheduledQueries).
		{
			name:            "team admin",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team maintainer",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team observer",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team observer+",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
		{
			name:            "team gitops",
			user:            &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			shouldFailWrite: true,
			shouldFailRead:  true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetScheduledQueriesInPack(ctx, 1, mdmlab.ListOptions{})
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ScheduleQuery(ctx, &mdmlab.ScheduledQuery{Interval: 10})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetScheduledQuery(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyScheduledQuery(ctx, 1, mdmlab.ScheduledQueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteScheduledQuery(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestScheduleQuery(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &mdmlab.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.NewScheduledQueryFunc = func(ctx context.Context, q *mdmlab.ScheduledQuery, opts ...mdmlab.OptionalArg) (*mdmlab.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(test.UserContext(ctx, test.UserAdmin), expectedQuery)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoName(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &mdmlab.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*mdmlab.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &mdmlab.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts mdmlab.ListOptions) ([]*mdmlab.ScheduledQuery, error) {
		// No matching query
		return []*mdmlab.ScheduledQuery{
			{
				Name: "froobling",
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, q *mdmlab.ScheduledQuery, opts ...mdmlab.OptionalArg) (*mdmlab.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&mdmlab.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 10},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestScheduleQueryNoNameMultiple(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &mdmlab.ScheduledQuery{
		Name:      "foobar-1",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*mdmlab.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &mdmlab.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts mdmlab.ListOptions) ([]*mdmlab.ScheduledQuery, error) {
		// No matching query
		return []*mdmlab.ScheduledQuery{
			{
				Name:     "foobar",
				Interval: 10,
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, q *mdmlab.ScheduledQuery, opts ...mdmlab.OptionalArg) (*mdmlab.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&mdmlab.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 10},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)
}

func TestFindNextNameForQuery(t *testing.T) {
	testCases := []struct {
		name      string
		scheduled []*mdmlab.ScheduledQuery
		expected  string
	}{
		{
			name:      "foobar",
			scheduled: []*mdmlab.ScheduledQuery{},
			expected:  "foobar",
		},
		{
			name: "foobar",
			scheduled: []*mdmlab.ScheduledQuery{
				{
					Name: "foobar",
				},
			},
			expected: "foobar-1",
		},
		{
			name: "foobar",
			scheduled: []*mdmlab.ScheduledQuery{
				{
					Name: "foobar",
				},
				{
					Name: "foobar-1",
				},
			},
			expected: "foobar-1-1",
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.expected, findNextNameForQuery(tt.name, tt.scheduled))
		})
	}
}

func TestScheduleQueryInterval(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedQuery := &mdmlab.ScheduledQuery{
		Name:      "foobar",
		QueryName: "foobar",
		QueryID:   3,
		Interval:  10,
	}

	ds.QueryFunc = func(ctx context.Context, qid uint) (*mdmlab.Query, error) {
		require.Equal(t, expectedQuery.QueryID, qid)
		return &mdmlab.Query{Name: expectedQuery.QueryName}, nil
	}
	ds.ListScheduledQueriesInPackWithStatsFunc = func(ctx context.Context, id uint, opts mdmlab.ListOptions) ([]*mdmlab.ScheduledQuery, error) {
		// No matching query
		return []*mdmlab.ScheduledQuery{
			{
				Name: "froobling",
			},
		}, nil
	}
	ds.NewScheduledQueryFunc = func(ctx context.Context, q *mdmlab.ScheduledQuery, opts ...mdmlab.OptionalArg) (*mdmlab.ScheduledQuery, error) {
		assert.Equal(t, expectedQuery, q)
		return expectedQuery, nil
	}

	_, err := svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&mdmlab.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 10},
	)
	assert.NoError(t, err)
	assert.True(t, ds.NewScheduledQueryFuncInvoked)

	// no interval
	_, err = svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&mdmlab.ScheduledQuery{QueryID: expectedQuery.QueryID},
	)
	assert.Error(t, err)

	// interval zero
	_, err = svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&mdmlab.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 0},
	)
	assert.Error(t, err)

	// interval exceeds max
	_, err = svc.ScheduleQuery(
		test.UserContext(ctx, test.UserAdmin),
		&mdmlab.ScheduledQuery{QueryID: expectedQuery.QueryID, Interval: 604801},
	)
	assert.Error(t, err)
}

func TestModifyScheduledQueryInterval(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ScheduledQueryFunc = func(ctx context.Context, id uint) (*mdmlab.ScheduledQuery, error) {
		assert.Equal(t, id, uint(1))
		return &mdmlab.ScheduledQuery{ID: id, Interval: 10}, nil
	}

	testCases := []struct {
		payload    mdmlab.ScheduledQueryPayload
		shouldFail bool
	}{
		{
			payload: mdmlab.ScheduledQueryPayload{
				QueryID:  ptr.Uint(1),
				Interval: ptr.Uint(0),
			},
			shouldFail: true,
		},
		{
			payload: mdmlab.ScheduledQueryPayload{
				QueryID:  ptr.Uint(1),
				Interval: ptr.Uint(604801),
			},
			shouldFail: true,
		},
		{
			payload: mdmlab.ScheduledQueryPayload{
				QueryID: ptr.Uint(1),
			},
			shouldFail: false,
		},
		{
			payload: mdmlab.ScheduledQueryPayload{
				QueryID:  ptr.Uint(1),
				Interval: ptr.Uint(604800),
			},
			shouldFail: false,
		},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			ds.SaveScheduledQueryFunc = func(ctx context.Context, sq *mdmlab.ScheduledQuery) (*mdmlab.ScheduledQuery, error) {
				assert.Equal(t, sq.ID, uint(1))
				return &mdmlab.ScheduledQuery{ID: sq.ID, Interval: sq.Interval}, nil
			}
			_, err := svc.ModifyScheduledQuery(test.UserContext(ctx, test.UserAdmin), *tt.payload.QueryID, tt.payload)
			if tt.shouldFail {
				assert.Error(t, err)
				assert.False(t, ds.SaveScheduledQueryFuncInvoked)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
