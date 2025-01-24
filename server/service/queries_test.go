package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/datastore/mysql"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryPayloadValidationCreate(t *testing.T) {
	ds := new(mock.Store)
	ds.NewQueryFunc = func(ctx context.Context, query *mdmlab.Query, opts ...mdmlab.OptionalArg) (*mdmlab.Query, error) {
		return query, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		act, ok := activity.(mdmlab.ActivityTypeCreatedSavedQuery)
		assert.True(t, ok)
		assert.NotEmpty(t, act.Name)
		return nil
	}
	svc, ctx := newTestService(t, ds, nil, nil)

	testCases := []struct {
		name         string
		queryPayload mdmlab.QueryPayload
		shouldErr    bool
	}{
		{
			"All valid",
			mdmlab.QueryPayload{
				Name:     ptr.String("test query"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			false,
		},
		{
			"Invalid  - empty string name",
			mdmlab.QueryPayload{
				Name:     ptr.String(""),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Empty SQL",
			mdmlab.QueryPayload{
				Name:     ptr.String("bad sql"),
				Query:    ptr.String(""),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Invalid logging",
			mdmlab.QueryPayload{
				Name:     ptr.String("bad logging"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("hopscotch"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Unsupported platform",
			mdmlab.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("charles"),
			},
			true,
		},
		{
			"Missing comma",
			mdmlab.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin windows"),
			},
			true,
		},
		{
			"Unsupported platform 'sphinx' ",
			mdmlab.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin,windows,sphinx"),
			},
			true,
		},
	}

	testAdmin := mdmlab.User{
		ID:         1,
		Teams:      []mdmlab.UserTeam{},
		GlobalRole: ptr.String(mdmlab.RoleAdmin),
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})
			query, err := svc.NewQuery(viewerCtx, tt.queryPayload)
			if tt.shouldErr {
				assert.Error(t, err)
				assert.Nil(t, query)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, query)
			}
		})
	}
}

// similar for modify
func TestQueryPayloadValidationModify(t *testing.T) {
	ds := new(mock.Store)
	ds.QueryFunc = func(ctx context.Context, id uint) (*mdmlab.Query, error) {
		return &mdmlab.Query{
			ID:             id,
			Name:           "mock saved query",
			Description:    "some desc",
			Query:          "select 1;",
			Platform:       "",
			Saved:          true,
			ObserverCanRun: false,
		}, nil
	}
	ds.SaveQueryFunc = func(ctx context.Context, query *mdmlab.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		assert.NotEmpty(t, query)
		return nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		act, ok := activity.(mdmlab.ActivityTypeEditedSavedQuery)
		assert.True(t, ok)
		assert.NotEmpty(t, act.Name)
		return nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)

	testCases := []struct {
		name         string
		queryPayload mdmlab.QueryPayload
		shouldErr    bool
	}{
		{
			"All valid",
			mdmlab.QueryPayload{
				Name:     ptr.String("updated test query"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			false,
		},
		{
			"Invalid  - empty string name",
			mdmlab.QueryPayload{
				Name:     ptr.String(""),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Empty SQL",
			mdmlab.QueryPayload{
				Name:     ptr.String("bad sql"),
				Query:    ptr.String(""),
				Logging:  ptr.String("snapshot"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Invalid logging",
			mdmlab.QueryPayload{
				Name:     ptr.String("bad logging"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("hopscotch"),
				Platform: ptr.String(""),
			},
			true,
		},
		{
			"Unsupported platform",
			mdmlab.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("charles"),
			},
			true,
		},
		{
			"Missing comma delimeter in platform string",
			mdmlab.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin windows"),
			},
			true,
		},
		{
			"Unsupported platform 2",
			mdmlab.QueryPayload{
				Name:     ptr.String("invalid platform"),
				Query:    ptr.String("select 1"),
				Logging:  ptr.String("differential"),
				Platform: ptr.String("darwin,windows,sphinx"),
			},
			true,
		},
	}

	testAdmin := mdmlab.User{
		ID:         1,
		Teams:      []mdmlab.UserTeam{},
		GlobalRole: ptr.String(mdmlab.RoleAdmin),
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &testAdmin})
			_, err := svc.ModifyQuery(viewerCtx, 1, tt.queryPayload)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQueryAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	team := mdmlab.Team{
		ID:   1,
		Name: "Foobar",
	}

	team2 := mdmlab.Team{
		ID:   2,
		Name: "Barfoo",
	}

	teamAdmin := &mdmlab.User{
		ID: 42,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team.ID},
				Role: mdmlab.RoleAdmin,
			},
		},
	}
	teamMaintainer := &mdmlab.User{
		ID: 43,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team.ID},
				Role: mdmlab.RoleMaintainer,
			},
		},
	}
	teamObserver := &mdmlab.User{
		ID: 44,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team.ID},
				Role: mdmlab.RoleObserver,
			},
		},
	}
	teamObserverPlus := &mdmlab.User{
		ID: 45,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team.ID},
				Role: mdmlab.RoleObserverPlus,
			},
		},
	}
	teamGitOps := &mdmlab.User{
		ID: 46,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team.ID},
				Role: mdmlab.RoleGitOps,
			},
		},
	}
	globalQuery := mdmlab.Query{
		ID:     99,
		Name:   "global query",
		TeamID: nil,
	}
	teamQuery := mdmlab.Query{
		ID:     88,
		Name:   "team query",
		TeamID: ptr.Uint(team.ID),
	}
	team2Query := mdmlab.Query{
		ID:     77,
		Name:   "team2 query",
		TeamID: ptr.Uint(team2.ID),
	}
	queriesMap := map[uint]mdmlab.Query{
		globalQuery.ID: globalQuery,
		teamQuery.ID:   teamQuery,
		team2Query.ID:  team2Query,
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*mdmlab.Team, error) {
		if tid == team.ID {
			return &team, nil
		} else if tid == team2.ID {
			return &team2, nil
		}
		return nil, newNotFoundError()
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		if name == team.Name {
			return &team, nil
		} else if name == team2.Name {
			return &team2, nil
		}
		return nil, newNotFoundError()
	}
	ds.NewQueryFunc = func(ctx context.Context, query *mdmlab.Query, opts ...mdmlab.OptionalArg) (*mdmlab.Query, error) {
		return query, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*mdmlab.Query, error) {
		if teamID == nil && name == "global query" { //nolint:gocritic // ignore ifElseChain
			return &globalQuery, nil
		} else if teamID != nil && *teamID == team.ID && name == "team query" {
			return &teamQuery, nil
		} else if teamID != nil && *teamID == team2.ID && name == "team2 query" {
			return &team2Query, nil
		}
		return nil, newNotFoundError()
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*mdmlab.Query, error) {
		if id == 99 { //nolint:gocritic // ignore ifElseChain
			return &globalQuery, nil
		} else if id == 88 {
			return &teamQuery, nil
		} else if id == 77 {
			return &team2Query, nil
		}
		return nil, newNotFoundError()
	}

	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}

	ds.SaveQueryFunc = func(ctx context.Context, query *mdmlab.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
		return nil
	}
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.DeleteQueriesFunc = func(ctx context.Context, ids []uint) (uint, error) {
		return 0, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts mdmlab.ListQueryOptions) ([]*mdmlab.Query, int, *mdmlab.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.ApplyQueriesFunc = func(ctx context.Context, authID uint, queries []*mdmlab.Query, queriesToDiscardResults map[uint]struct{}) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *mdmlab.User
		qid             uint
		shouldFailWrite bool
		shouldFailRead  bool
		shouldFailNew   bool
	}{
		{
			"global admin and global query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			globalQuery.ID,
			false,
			false,
			false,
		},
		{
			"global admin and team query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"global maintainer and global query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			globalQuery.ID,
			false,
			false,
			false,
		},
		{
			"global maintainer and team query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"global observer and global query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"global observer and team query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"global observer+ and global query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"global observer+ and team query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"global gitops and global query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			globalQuery.ID,
			false,
			false,
			false,
		},
		{
			"global gitops and team query",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"team admin and global query",
			teamAdmin,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team admin and team query",
			teamAdmin,
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"team admin and team2 query",
			teamAdmin,
			team2Query.ID,
			true,
			true,
			true,
		},
		{
			"team maintainer and global query",
			teamMaintainer,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team maintainer and team query",
			teamMaintainer,
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"team maintainer and team2 query",
			teamMaintainer,
			team2Query.ID,
			true,
			true,
			true,
		},
		{
			"team observer and global query",
			teamObserver,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer and team query",
			teamObserver,
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer and team2 query",
			teamObserver,
			team2Query.ID,
			true,
			true,
			true,
		},
		{
			"team observer+ and global query",
			teamObserverPlus,
			globalQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer+ and team query",
			teamObserverPlus,
			teamQuery.ID,
			true,
			false,
			true,
		},
		{
			"team observer+ and team2 query",
			teamObserverPlus,
			team2Query.ID,
			true,
			true,
			true,
		},
		{
			"team gitops and global query",
			teamGitOps,
			globalQuery.ID,
			true,
			true,
			true,
		},
		{
			"team gitops and team query",
			teamGitOps,
			teamQuery.ID,
			false,
			false,
			false,
		},
		{
			"team gitops and team2 query",
			teamGitOps,
			team2Query.ID,
			true,
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			query := queriesMap[tt.qid]

			_, err := svc.NewQuery(ctx, mdmlab.QueryPayload{
				Name:   ptr.String("name"),
				Query:  ptr.String("select 1"),
				TeamID: query.TeamID,
			})
			checkAuthErr(t, tt.shouldFailNew, err)

			_, err = svc.ModifyQuery(ctx, tt.qid, mdmlab.QueryPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteQuery(ctx, query.TeamID, query.Name)
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteQueryByID(ctx, tt.qid)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteQueries(ctx, []uint{tt.qid})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetQuery(ctx, tt.qid)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.QueryReportIsClipped(ctx, tt.qid, mdmlab.DefaultMaxQueryReportRows)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, _, _, err = svc.ListQueries(ctx, mdmlab.ListOptions{}, query.TeamID, nil, false, nil)
			checkAuthErr(t, tt.shouldFailRead, err)

			teamName := ""
			if query.TeamID != nil && *query.TeamID == team.ID {
				teamName = team.Name
			} else if query.TeamID != nil && *query.TeamID == team2.ID {
				teamName = team2.Name
			}

			err = svc.ApplyQuerySpecs(ctx, []*mdmlab.QuerySpec{{
				Name:     query.Name,
				Query:    "SELECT 1",
				TeamName: teamName,
			}})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetQuerySpecs(ctx, query.TeamID)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetQuerySpec(ctx, query.TeamID, query.Name)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

func TestQueryReportIsClipped(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &mdmlab.User{
		ID:         1,
		GlobalRole: ptr.String(mdmlab.RoleAdmin),
	}})

	ds.QueryFunc = func(ctx context.Context, queryID uint) (*mdmlab.Query, error) {
		return &mdmlab.Query{}, nil
	}
	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return 0, nil
	}

	isClipped, err := svc.QueryReportIsClipped(viewerCtx, 1, mdmlab.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.False(t, isClipped)

	ds.ResultCountForQueryFunc = func(ctx context.Context, queryID uint) (int, error) {
		return mdmlab.DefaultMaxQueryReportRows, nil
	}

	isClipped, err = svc.QueryReportIsClipped(viewerCtx, 1, mdmlab.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	require.True(t, isClipped)
}

func TestQueryReportReturnsNilIfDiscardDataIsTrue(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	viewerCtx := viewer.NewContext(ctx, viewer.Viewer{User: &mdmlab.User{
		ID:         1,
		GlobalRole: ptr.String(mdmlab.RoleAdmin),
	}})

	ds.QueryFunc = func(ctx context.Context, queryID uint) (*mdmlab.Query, error) {
		return &mdmlab.Query{
			DiscardData: true,
		}, nil
	}
	ds.QueryResultRowsFunc = func(ctx context.Context, queryID uint, opts mdmlab.TeamFilter) ([]*mdmlab.ScheduledQueryResultRow, error) {
		return []*mdmlab.ScheduledQueryResultRow{
			{
				QueryID:     1,
				HostID:      1,
				Data:        ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
				LastFetched: time.Now(),
			},
		}, nil
	}

	results, reportClipped, err := svc.GetQueryReportResults(viewerCtx, 1, nil)
	require.NoError(t, err)
	require.Nil(t, results)
	require.False(t, reportClipped)
}

func TestInheritedQueryReportTeamPermissions(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc, ctx := newTestService(t, ds, nil, nil)

	team1, err := ds.NewTeam(ctx, &mdmlab.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &mdmlab.Team{
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)

	hostTeam2, err := ds.NewHost(ctx, &mdmlab.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		ComputerName:    "Foo Local",
		Hostname:        "foo.local",
		OsqueryHostID:   ptr.String("1"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-61",
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &team2.ID, []uint{hostTeam2.ID})
	require.NoError(t, err)

	hostTeam1, err := ds.NewHost(ctx, &mdmlab.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("42"),
		UUID:            "42",
		ComputerName:    "bar Local",
		Hostname:        "bar.local",
		OsqueryHostID:   ptr.String("42"),
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-62",
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(ctx, &team1.ID, []uint{hostTeam1.ID})
	require.NoError(t, err)

	globalQuery, err := ds.NewQuery(ctx, &mdmlab.Query{
		ID:      77,
		Name:    "team2 query",
		TeamID:  nil,
		Query:   "select * from usb_devices;",
		Logging: mdmlab.LoggingSnapshot,
	})
	require.NoError(t, err)
	// Insert initial Result Rows
	mockTime := time.Now().UTC().Truncate(time.Second)
	host2Row := []*mdmlab.ScheduledQueryResultRow{
		{
			QueryID:     globalQuery.ID,
			HostID:      hostTeam2.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Keyboard", "vendor": "Apple Inc."}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(ctx, host2Row, mdmlab.DefaultMaxQueryReportRows)
	require.NoError(t, err)
	host1Row := []*mdmlab.ScheduledQueryResultRow{
		{
			QueryID:     globalQuery.ID,
			HostID:      hostTeam1.ID,
			LastFetched: mockTime,
			Data:        ptr.RawMessage([]byte(`{"model": "USB Mouse", "vendor": "Apple Inc."}`)),
		},
	}
	err = ds.OverwriteQueryResultRows(ctx, host1Row, mdmlab.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	team2Admin := &mdmlab.User{
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team2.ID},
				Role: mdmlab.RoleAdmin,
			},
		},
	}

	queryReportResults, _, err := svc.GetQueryReportResults(viewer.NewContext(ctx, viewer.Viewer{User: team2Admin}), globalQuery.ID, &team2.ID)
	require.NoError(t, err)
	require.Len(t, queryReportResults, 1)

	// team admins requesting query results filtered to not-their-team should get no rows back

	teamAdmin := &mdmlab.User{
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team1.ID},
				Role: mdmlab.RoleAdmin,
			},
		},
	}
	teamMaintainer := &mdmlab.User{
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team1.ID},
				Role: mdmlab.RoleMaintainer,
			},
		},
	}
	teamObserver := &mdmlab.User{
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team1.ID},
				Role: mdmlab.RoleObserver,
			},
		},
	}
	teamObserverPlus := &mdmlab.User{
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: team1.ID},
				Role: mdmlab.RoleObserverPlus,
			},
		},
	}

	testCases := []struct {
		name string
		user *mdmlab.User
	}{
		{
			name: "team admin",
			user: teamAdmin,
		},
		{
			name: "team maintainer",
			user: teamMaintainer,
		},
		{
			name: "team observer",
			user: teamObserver,
		},
		{
			name: "team observer+",
			user: teamObserverPlus,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			queryReportResults, _, err := svc.GetQueryReportResults(viewer.NewContext(ctx, viewer.Viewer{User: tt.user}), globalQuery.ID, &team2.ID)
			require.NoError(t, err)
			require.Len(t, queryReportResults, 0)
		})
	}
}

func TestComparePlatforms(t *testing.T) {
	for _, tc := range []struct {
		name     string
		p1       string
		p2       string
		expected bool
	}{
		{
			name:     "equal single value",
			p1:       "linux",
			p2:       "linux",
			expected: true,
		},
		{
			name:     "different single value",
			p1:       "macos",
			p2:       "linux",
			expected: false,
		},
		{
			name:     "equal multiple values",
			p1:       "linux,windows",
			p2:       "linux,windows",
			expected: true,
		},
		{
			name:     "equal multiple values out of order",
			p1:       "linux,windows",
			p2:       "windows,linux",
			expected: true,
		},
		{
			name:     "different multiple values",
			p1:       "linux,windows",
			p2:       "linux,windows,darwin",
			expected: false,
		},
		{
			name:     "no values set",
			p1:       "",
			p2:       "",
			expected: true,
		},
		{
			name:     "no values set",
			p1:       "",
			p2:       "linux",
			expected: false,
		},
		{
			name:     "single and multiple values",
			p1:       "linux",
			p2:       "windows,linux",
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := comparePlatforms(tc.p1, tc.p2)
			require.Equal(t, tc.expected, actual)
		})
	}
}
