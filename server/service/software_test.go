package service

import (
	"context"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ListSoftware(t *testing.T) {
	ds := new(mock.Store)

	var calledWithTeamID *uint
	var calledWithOpt mdmlab.SoftwareListOptions
	ds.ListSoftwareFunc = func(ctx context.Context, opt mdmlab.SoftwareListOptions) ([]mdmlab.Software, *mdmlab.PaginationMetadata, error) {
		calledWithTeamID = opt.TeamID
		calledWithOpt = opt
		return []mdmlab.Software{}, &mdmlab.PaginationMetadata{}, nil
	}

	user := &mdmlab.User{
		ID:         3,
		Email:      "foo@bar.com",
		GlobalRole: ptr.String(mdmlab.RoleAdmin),
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	_, _, err := svc.ListSoftware(ctx, mdmlab.SoftwareListOptions{TeamID: ptr.Uint(42), ListOptions: mdmlab.ListOptions{PerPage: 77, Page: 4}})
	require.NoError(t, err)

	assert.True(t, ds.ListSoftwareFuncInvoked)
	assert.Equal(t, ptr.Uint(42), calledWithTeamID)
	// sort order defaults to hosts_count descending, automatically, if not explicitly provided
	assert.Equal(t, mdmlab.ListOptions{PerPage: 77, Page: 4, OrderKey: "hosts_count", OrderDirection: mdmlab.OrderDescending}, calledWithOpt.ListOptions)
	assert.True(t, calledWithOpt.WithHostCounts)

	// call again, this time with an explicit sort
	ds.ListSoftwareFuncInvoked = false
	_, _, err = svc.ListSoftware(ctx, mdmlab.SoftwareListOptions{TeamID: nil, ListOptions: mdmlab.ListOptions{PerPage: 11, Page: 2, OrderKey: "id", OrderDirection: mdmlab.OrderAscending}})
	require.NoError(t, err)

	assert.True(t, ds.ListSoftwareFuncInvoked)
	assert.Nil(t, calledWithTeamID)
	assert.Equal(t, mdmlab.ListOptions{PerPage: 11, Page: 2, OrderKey: "id", OrderDirection: mdmlab.OrderAscending}, calledWithOpt.ListOptions)
	assert.True(t, calledWithOpt.WithHostCounts)
}

func TestServiceSoftwareInventoryAuth(t *testing.T) {
	ds := new(mock.Store)

	ds.ListSoftwareFunc = func(ctx context.Context, opt mdmlab.SoftwareListOptions) ([]mdmlab.Software, *mdmlab.PaginationMetadata, error) {
		return []mdmlab.Software{}, &mdmlab.PaginationMetadata{}, nil
	}
	ds.CountSoftwareFunc = func(ctx context.Context, opt mdmlab.SoftwareListOptions) (int, error) {
		return 0, nil
	}
	ds.SoftwareByIDFunc = func(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *mdmlab.TeamFilter) (*mdmlab.Software, error) {
		return &mdmlab.Software{}, nil
	}
	ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) { return true, nil }
	svc, ctx := newTestService(t, ds, nil, nil)

	for _, tc := range []struct {
		name                 string
		user                 *mdmlab.User
		shouldFailGlobalRead bool
		shouldFailTeamRead   bool
	}{
		{
			name: "global-admin",
			user: &mdmlab.User{
				ID:         1,
				GlobalRole: ptr.String(mdmlab.RoleAdmin),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name: "global-maintainer",
			user: &mdmlab.User{
				ID:         1,
				GlobalRole: ptr.String(mdmlab.RoleMaintainer),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name: "global-observer",
			user: &mdmlab.User{
				ID:         1,
				GlobalRole: ptr.String(mdmlab.RoleObserver),
			},
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-admin-belongs-to-team",
			user: &mdmlab.User{
				ID: 1,
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 1},
					Role: mdmlab.RoleAdmin,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-maintainer-belongs-to-team",
			user: &mdmlab.User{
				ID: 1,
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 1},
					Role: mdmlab.RoleMaintainer,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-observer-belongs-to-team",
			user: &mdmlab.User{
				ID: 1,
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 1},
					Role: mdmlab.RoleObserver,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name: "team-admin-does-not-belong-to-team",
			user: &mdmlab.User{
				ID: 1,
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 2},
					Role: mdmlab.RoleAdmin,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name: "team-maintainer-does-not-belong-to-team",
			user: &mdmlab.User{
				ID: 1,
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 2},
					Role: mdmlab.RoleMaintainer,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name: "team-observer-does-not-belong-to-team",
			user: &mdmlab.User{
				ID: 1,
				Teams: []mdmlab.UserTeam{{
					Team: mdmlab.Team{ID: 2},
					Role: mdmlab.RoleObserver,
				}},
			},
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tc.user})

			// List all software.
			_, _, err := svc.ListSoftware(ctx, mdmlab.SoftwareListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// Count all software.
			_, err = svc.CountSoftware(ctx, mdmlab.SoftwareListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// List software for a team.
			_, _, err = svc.ListSoftware(ctx, mdmlab.SoftwareListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			// Count software for a team.
			_, err = svc.CountSoftware(ctx, mdmlab.SoftwareListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			_, err = svc.SoftwareByID(ctx, 1, ptr.Uint(1), false)
			checkAuthErr(t, tc.shouldFailTeamRead, err)
		})
	}
}
