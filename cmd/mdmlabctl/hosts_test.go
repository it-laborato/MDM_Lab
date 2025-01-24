package main

import (
	"context"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostTransferFlagChecks(t *testing.T) {
	runServerWithMockedDS(t)

	runAppCheckErr(t,
		[]string{"hosts", "transfer", "--team", "team1", "--hosts", "host1", "--label", "AAA"},
		"--hosts cannot be used along side any other flag",
	)
	runAppCheckErr(t,
		[]string{"hosts", "transfer", "--team", "team1"},
		"You need to define either --hosts, or one or more of --label, --status, --search_query",
	)
}

func TestHostsTransferByHosts(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*mdmlab.Host, error) {
		require.Equal(t, "host1", identifier)
		return &mdmlab.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		require.Equal(t, "team1", name)
		return &mdmlab.Team{ID: 99, Name: "team1"}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		require.NotNil(t, teamID)
		require.Equal(t, uint(99), *teamID)
		require.Equal(t, []uint{42}, hostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates mdmlab.MDMProfilesUpdates, err error) {
		return mdmlab.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, mdmlab.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*mdmlab.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", runAppForTest(t, []string{"hosts", "transfer", "--team", "team1", "--hosts", "host1"}))
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)

	// Now, transfer out of the team.
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		assert.Nil(t, teamID)
		assert.Equal(t, []uint{42}, hostIDs)
		return nil
	}
	ds.NewActivityFuncInvoked = false
	ds.AddHostsToTeamFuncInvoked = false
	assert.Equal(t, "", runAppForTest(t, []string{"hosts", "transfer", "--team", "", "--hosts", "host1"}))
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
}

func TestHostsTransferByLabel(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*mdmlab.Host, error) {
		require.Equal(t, "host1", identifier)
		return &mdmlab.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		require.Equal(t, "team1", name)
		return &mdmlab.Team{ID: 99, Name: "team1"}, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Equal(t, []string{"label1"}, labels)
		return map[string]uint{"label1": uint(11)}, nil
	}

	ds.ListHostsInLabelFunc = func(ctx context.Context, filter mdmlab.TeamFilter, lid uint, opt mdmlab.HostListOptions) ([]*mdmlab.Host, error) {
		require.Equal(t, mdmlab.HostStatus(""), opt.StatusFilter)
		require.Equal(t, uint(11), lid)
		return []*mdmlab.Host{{ID: 32}, {ID: 12}}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		require.NotNil(t, teamID)
		require.Equal(t, uint(99), *teamID)
		require.Equal(t, []uint{32, 12}, hostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates mdmlab.MDMProfilesUpdates, err error) {
		return mdmlab.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, mdmlab.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*mdmlab.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", runAppForTest(t, []string{"hosts", "transfer", "--team", "team1", "--label", "label1"}))
	require.True(t, ds.NewActivityFuncInvoked)
	assert.True(t, ds.AddHostsToTeamFuncInvoked)

	// Now, transfer out of the team.
	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		assert.Nil(t, teamID)
		require.Equal(t, []uint{32, 12}, hostIDs)
		return nil
	}
	ds.NewActivityFuncInvoked = false
	ds.AddHostsToTeamFuncInvoked = false
	assert.Equal(t, "", runAppForTest(t, []string{"hosts", "transfer", "--team", "", "--label", "label1"}))
	assert.True(t, ds.AddHostsToTeamFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
}

func TestHostsTransferByStatus(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*mdmlab.Host, error) {
		require.Equal(t, "host1", identifier)
		return &mdmlab.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		require.Equal(t, "team1", name)
		return &mdmlab.Team{ID: 99, Name: "team1"}, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Equal(t, []string{"label1"}, labels)
		return map[string]uint{"label1": uint(11)}, nil
	}

	ds.ListHostsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, opt mdmlab.HostListOptions) ([]*mdmlab.Host, error) {
		require.Equal(t, mdmlab.StatusOnline, opt.StatusFilter)
		return []*mdmlab.Host{{ID: 32}, {ID: 12}}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		require.NotNil(t, teamID)
		require.Equal(t, uint(99), *teamID)
		require.Equal(t, []uint{32, 12}, hostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates mdmlab.MDMProfilesUpdates, err error) {
		return mdmlab.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, mdmlab.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*mdmlab.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", runAppForTest(t,
		[]string{"hosts", "transfer", "--team", "team1", "--status", "online"}))
	require.True(t, ds.NewActivityFuncInvoked)
}

func TestHostsTransferByStatusAndSearchQuery(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*mdmlab.Host, error) {
		require.Equal(t, "host1", identifier)
		return &mdmlab.Host{ID: 42}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		require.Equal(t, "team1", name)
		return &mdmlab.Team{ID: 99, Name: "team1"}, nil
	}

	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.Equal(t, []string{"label1"}, labels)
		return map[string]uint{"label1": uint(11)}, nil
	}

	ds.ListHostsFunc = func(ctx context.Context, filter mdmlab.TeamFilter, opt mdmlab.HostListOptions) ([]*mdmlab.Host, error) {
		require.Equal(t, mdmlab.StatusOnline, opt.StatusFilter)
		require.Equal(t, "somequery", opt.MatchQuery)
		return []*mdmlab.Host{{ID: 32}, {ID: 12}}, nil
	}

	ds.AddHostsToTeamFunc = func(ctx context.Context, teamID *uint, hostIDs []uint) error {
		require.NotNil(t, teamID)
		require.Equal(t, uint(99), *teamID)
		require.Equal(t, []uint{32, 12}, hostIDs)
		return nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, uuids []string,
	) (updates mdmlab.MDMProfilesUpdates, err error) {
		return mdmlab.MDMProfilesUpdates{}, nil
	}

	ds.ListMDMAppleDEPSerialsInHostIDsFunc = func(ctx context.Context, hostIDs []uint) ([]string, error) {
		return nil, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: tid, Name: "team1"}, nil
	}

	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		require.IsType(t, mdmlab.ActivityTypeTransferredHostsToTeam{}, activity)
		return nil
	}

	ds.ListHostsLiteByIDsFunc = func(ctx context.Context, ids []uint) ([]*mdmlab.Host, error) {
		return nil, nil
	}

	assert.Equal(t, "", runAppForTest(t,
		[]string{"hosts", "transfer", "--team", "team1", "--status", "online", "--search_query", "somequery"}))
	require.True(t, ds.NewActivityFuncInvoked)
}
