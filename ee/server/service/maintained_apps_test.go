package service

import (
	"context"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestListMaintainedAppsAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.ListAvailableMDMlabMaintainedAppsFunc = func(ctx context.Context, teamID *uint, opt mdmlab.ListOptions) ([]mdmlab.MaintainedApp, *mdmlab.PaginationMetadata, error) {
		return []mdmlab.MaintainedApp{}, &mdmlab.PaginationMetadata{}, nil
	}
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{authz: authorizer, ds: ds}

	testCases := []struct {
		name                        string
		user                        *mdmlab.User
		shouldFailWithNoTeam        bool
		shouldFailWithMatchingTeam  bool
		shouldFailWithDifferentTeam bool
	}{
		{
			"global admin",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			false,
			false,
			false,
		},
		{
			"global observer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			true,
			true,
			true,
		},
		{
			"team admin",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			false,
			false,
			true,
		},
		{
			"team maintainer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			false,
			false,
			true,
		},
		{
			"team observer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			true,
			true,
			true,
		},
	}

	var forbiddenError *authz.Forbidden
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, _, err := svc.ListMDMlabMaintainedApps(ctx, nil, mdmlab.ListOptions{})
			if tt.shouldFailWithNoTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, _, err = svc.ListMDMlabMaintainedApps(ctx, ptr.Uint(1), mdmlab.ListOptions{})
			if tt.shouldFailWithMatchingTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, _, err = svc.ListMDMlabMaintainedApps(ctx, ptr.Uint(2), mdmlab.ListOptions{})
			if tt.shouldFailWithDifferentTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetMaintainedAppAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint) (*mdmlab.MaintainedApp, error) {
		return &mdmlab.MaintainedApp{}, nil
	}
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{authz: authorizer, ds: ds}

	testCases := []struct {
		name       string
		user       *mdmlab.User
		shouldFail bool
	}{
		{
			"global admin",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
			false,
		},
		{
			"global observer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			true,
		},
		{
			"team admin",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			false,
		},
		{
			"team maintainer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			false,
		},
		{
			"team observer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			true,
		},
	}

	var forbiddenError *authz.Forbidden
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			_, err := svc.GetMDMlabMaintainedApp(ctx, 123)

			if tt.shouldFail {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
