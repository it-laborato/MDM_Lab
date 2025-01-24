package service

import (
	"context"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestListVulnerabilities(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}})

	ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt mdmlab.VulnListOptions) ([]mdmlab.VulnerabilityWithMetadata, *mdmlab.PaginationMetadata, error) {
		return []mdmlab.VulnerabilityWithMetadata{
			{
				CVE: mdmlab.CVE{
					CVE:         "CVE-2019-1234",
					Description: ptr.StringPtr("A vulnerability"),
				},
				CreatedAt:  time.Now(),
				HostsCount: 10,
			},
		}, nil, nil
	}

	t.Run("no list options", func(t *testing.T) {
		_, _, err := svc.ListVulnerabilities(ctx, mdmlab.VulnListOptions{})
		require.NoError(t, err)
	})

	t.Run("can only sort by supported columns", func(t *testing.T) {
		// invalid order key
		opts := mdmlab.VulnListOptions{ListOptions: mdmlab.ListOptions{
			OrderKey: "invalid",
		}, ValidSortColumns: freeValidVulnSortColumns}

		_, _, err := svc.ListVulnerabilities(ctx, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid order key")

		// valid order key
		opts.ListOptions.OrderKey = "cve"
		_, _, err = svc.ListVulnerabilities(ctx, opts)
		require.NoError(t, err)
	})
}

func TestVulnerabilitesAuth(t *testing.T) {
	ds := new(mock.Store)

	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt mdmlab.VulnListOptions) ([]mdmlab.VulnerabilityWithMetadata, *mdmlab.PaginationMetadata, error) {
		return []mdmlab.VulnerabilityWithMetadata{}, &mdmlab.PaginationMetadata{}, nil
	}

	ds.VulnerabilityFunc = func(cxt context.Context, cve string, teamID *uint, includeCVEScores bool) (*mdmlab.VulnerabilityWithMetadata, error) {
		return &mdmlab.VulnerabilityWithMetadata{}, nil
	}

	ds.CountVulnerabilitiesFunc = func(cxt context.Context, opt mdmlab.VulnListOptions) (uint, error) {
		return 0, nil
	}

	ds.TeamExistsFunc = func(cxt context.Context, teamID uint) (bool, error) {
		return true, nil
	}

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
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: tc.user})
			_, _, err := svc.ListVulnerabilities(ctx, mdmlab.VulnListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			_, _, err = svc.ListVulnerabilities(ctx, mdmlab.VulnListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			_, err = svc.CountVulnerabilities(ctx, mdmlab.VulnListOptions{})
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			_, err = svc.CountVulnerabilities(ctx, mdmlab.VulnListOptions{
				TeamID: ptr.Uint(1),
			})
			checkAuthErr(t, tc.shouldFailTeamRead, err)

			_, _, err = svc.Vulnerability(ctx, "CVE-2019-1234", nil, false)
			checkAuthErr(t, tc.shouldFailGlobalRead, err)

			_, _, err = svc.Vulnerability(ctx, "CVE-2019-1234", ptr.Uint(1), false)
			checkAuthErr(t, tc.shouldFailTeamRead, err)
		})
	}
}
