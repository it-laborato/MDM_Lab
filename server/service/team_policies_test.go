package service

import (
	"context"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/authz"
	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestTeamPoliciesAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.NewTeamPolicyFunc = func(ctx context.Context, teamID uint, authorID *uint, args mdmlab.PolicyPayload) (*mdmlab.Policy, error) {
		return &mdmlab.Policy{
			PolicyData: mdmlab.PolicyData{
				ID:     1,
				TeamID: ptr.Uint(1),
			},
		}, nil
	}
	ds.ListTeamPoliciesFunc = func(ctx context.Context, teamID uint, opts mdmlab.ListOptions, iopts mdmlab.ListOptions) (tpol, ipol []*mdmlab.Policy, err error) {
		return nil, nil, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*mdmlab.Policy, error) {
		return nil, nil
	}
	ds.TeamPolicyFunc = func(ctx context.Context, teamID uint, policyID uint) (*mdmlab.Policy, error) {
		return &mdmlab.Policy{}, nil
	}
	ds.PolicyFunc = func(ctx context.Context, id uint) (*mdmlab.Policy, error) {
		if id == 1 {
			return &mdmlab.Policy{
				PolicyData: mdmlab.PolicyData{
					ID:     1,
					TeamID: ptr.Uint(1),
				},
			}, nil
		}
		return nil, nil
	}
	ds.SavePolicyFunc = func(ctx context.Context, p *mdmlab.Policy, shouldDeleteAll bool, removePolicyStats bool) error {
		return nil
	}
	ds.DeleteTeamPoliciesFunc = func(ctx context.Context, teamID uint, ids []uint) ([]uint, error) {
		return nil, nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: 1}, nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*mdmlab.PolicySpec) error {
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
	ds.TeamFunc = func(ctx context.Context, tid uint) (*mdmlab.Team, error) {
		return &mdmlab.Team{ID: 1}, nil
	}
	ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*mdmlab.SoftwareInstaller, error) {
		return &mdmlab.SoftwareInstaller{}, nil
	}

	testCases := []struct {
		name            string
		user            *mdmlab.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
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
			false,
		},
		{
			"team admin, belongs to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
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
			"team admin, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleAdmin}}},
			true,
			true,
		},
		{
			"team observer, and team admin of another team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: mdmlab.RoleObserver,
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: mdmlab.RoleAdmin,
				},
			}},
			true,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 2}, Role: mdmlab.RoleObserver}}},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.NewTeamPolicy(ctx, 1, mdmlab.NewTeamPolicyPayload{
				Name:  "query1",
				Query: "select 1;",
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, _, err = svc.ListTeamPolicies(ctx, 1, mdmlab.ListOptions{}, mdmlab.ListOptions{}, false)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetTeamPolicyByIDQueries(ctx, 1, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyTeamPolicy(ctx, 1, 1, mdmlab.ModifyPolicyPayload{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.DeleteTeamPolicies(ctx, 1, []uint{1})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.ApplyPolicySpecs(ctx, []*mdmlab.PolicySpec{
				{
					Name:  "query1",
					Query: "select 1;",
					Team:  "team1",
				},
			})
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestTeamPolicyVPPAutomationRejectsNonMacOS(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}})

	appID := mdmlab.VPPAppID{AdamID: "123456", Platform: mdmlab.IOSPlatform}
	ds.TeamExistsFunc = func(ctx context.Context, id uint) (bool, error) {
		return true, nil
	}
	ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter mdmlab.TeamFilter) (*mdmlab.SoftwareTitle, error) {
		return &mdmlab.SoftwareTitle{
			AppStoreApp: &mdmlab.VPPAppStoreApp{
				VPPAppID: appID,
			},
		}, nil
	}

	_, err := svc.NewTeamPolicy(ctx, 1, mdmlab.NewTeamPolicyPayload{
		Name:            "query1",
		Query:           "select 1;",
		SoftwareTitleID: ptr.Uint(123),
	})
	require.ErrorContains(t, err, "is associated to an iOS or iPadOS VPP app")
}

func checkAuthErr(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	if shouldFail {
		require.Error(t, err)
		var forbiddenError *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenError)
	} else {
		require.NoError(t, err)
	}
}
