package service

import (
	"context"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/contexts/license"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestServiceSoftwareTitlesAuth(t *testing.T) {
	ds := new(mock.Store)

	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt mdmlab.SoftwareTitleListOptions, tmf mdmlab.TeamFilter) ([]mdmlab.SoftwareTitleListResult, int, *mdmlab.PaginationMetadata, error) {
		return []mdmlab.SoftwareTitleListResult{}, 0, &mdmlab.PaginationMetadata{}, nil
	}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter mdmlab.TeamFilter) (*mdmlab.SoftwareTitle, error) {
		return &mdmlab.SoftwareTitle{}, nil
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
			premiumCtx := license.NewContext(ctx, &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium})

			// List all software titles.
			_, _, _, err := svc.ListSoftwareTitles(ctx, mdmlab.SoftwareTitleListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			// List software for a team.
			_, _, _, err = svc.ListSoftwareTitles(premiumCtx, mdmlab.SoftwareTitleListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			// List software for a team should fail no matter what
			// with a non-premium context
			if !tc.shouldFailTeamRead {
				_, _, _, err = svc.ListSoftwareTitles(ctx, mdmlab.SoftwareTitleListOptions{
					TeamID: ptr.Uint(1),
				})
				require.ErrorContains(t, err, "Requires MDMlab Premium license")
			}

			// Get a software title for a team
			_, err = svc.SoftwareTitleByID(ctx, 1, ptr.Uint(1))
			checkAuthErr(t, tc.shouldFailTeamRead, err)
		})
	}
}
