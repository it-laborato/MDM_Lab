package service

import (
	"context"
	"testing"
	"time"

	eeservice "github.com/it-laborato/MDM_Lab/ee/server/service"
	authz_ctx "github.com/it-laborato/MDM_Lab/server/contexts/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallersAuth(t *testing.T) {
	ds := new(mock.Store)

	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license})

	testCases := []struct {
		name            string
		user            *mdmlab.User
		teamID          *uint
		shouldFailRead  bool
		shouldFailWrite bool
	}{
		{"no role no team", test.UserNoRoles, nil, true, true},
		{"no role team 0", test.UserNoRoles, ptr.Uint(0), true, true},
		{"no role team", test.UserNoRoles, ptr.Uint(1), true, true},
		{"global admin no team", test.UserAdmin, nil, false, false},
		{"global admin team 0", test.UserAdmin, ptr.Uint(0), false, false},
		{"global admin team", test.UserAdmin, ptr.Uint(1), false, false},
		{"global maintainer no team", test.UserMaintainer, nil, false, false},
		{"global mainteiner team 0", test.UserMaintainer, ptr.Uint(0), false, false},
		{"global maintainer team", test.UserMaintainer, ptr.Uint(1), false, false},
		{"global observer no team", test.UserObserver, nil, true, true},
		{"global observer team 0", test.UserObserver, ptr.Uint(0), true, true},
		{"global observer team", test.UserObserver, ptr.Uint(1), true, true},
		{"global observer+ no team", test.UserObserverPlus, nil, true, true},
		{"global observer+ team 0", test.UserObserverPlus, ptr.Uint(0), true, true},
		{"global observer+ team", test.UserObserverPlus, ptr.Uint(1), true, true},
		{"global gitops no team", test.UserGitOps, nil, true, false},
		{"global gitops team 0", test.UserGitOps, ptr.Uint(0), true, false},
		{"global gitops team", test.UserGitOps, ptr.Uint(1), true, false},
		{"team admin no team", test.UserTeamAdminTeam1, nil, true, true},
		{"team admin team 0", test.UserTeamAdminTeam1, ptr.Uint(0), true, true},
		{"team admin team", test.UserTeamAdminTeam1, ptr.Uint(1), false, false},
		{"team admin other team", test.UserTeamAdminTeam2, ptr.Uint(1), true, true},
		{"team maintainer no team", test.UserTeamMaintainerTeam1, nil, true, true},
		{"team maintainer team 0", test.UserTeamMaintainerTeam1, ptr.Uint(0), true, true},
		{"team maintainer team", test.UserTeamMaintainerTeam1, ptr.Uint(1), false, false},
		{"team maintainer other team", test.UserTeamMaintainerTeam2, ptr.Uint(1), true, true},
		{"team observer no team", test.UserTeamObserverTeam1, nil, true, true},
		{"team observer team 0", test.UserTeamObserverTeam1, ptr.Uint(0), true, true},
		{"team observer team", test.UserTeamObserverTeam1, ptr.Uint(1), true, true},
		{"team observer other team", test.UserTeamObserverTeam2, ptr.Uint(1), true, true},
		{"team observer+ no team", test.UserTeamObserverPlusTeam1, nil, true, true},
		{"team observer+ team 0", test.UserTeamObserverPlusTeam1, ptr.Uint(0), true, true},
		{"team observer+ team", test.UserTeamObserverPlusTeam1, ptr.Uint(1), true, true},
		{"team observer+ other team", test.UserTeamObserverPlusTeam2, ptr.Uint(1), true, true},
		{"team gitops no team", test.UserTeamGitOpsTeam1, nil, true, true},
		{"team gitops team 0", test.UserTeamGitOpsTeam1, ptr.Uint(0), true, true},
		{"team gitops team", test.UserTeamGitOpsTeam1, ptr.Uint(1), true, false},
		{"team gitops other team", test.UserTeamGitOpsTeam2, ptr.Uint(1), true, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScripts bool) (*mdmlab.SoftwareInstaller, error) {
				return &mdmlab.SoftwareInstaller{TeamID: tt.teamID}, nil
			}

			ds.DeleteSoftwareInstallerFunc = func(ctx context.Context, installerID uint) error {
				return nil
			}

			tokenExpiration := time.Now().Add(24 * time.Hour)
			token, err := test.CreateVPPTokenEncoded(tokenExpiration, "mdmlab", "ca")
			require.NoError(t, err)
			ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*mdmlab.VPPTokenDB, error) {
				return &mdmlab.VPPTokenDB{
					ID:        1,
					OrgName:   "MDMlab",
					Location:  "Earth",
					RenewDate: tokenExpiration,
					Token:     string(token),
					Teams:     nil,
				}, nil
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
				if tt.teamID != nil {
					return &mdmlab.Team{ID: *tt.teamID}, nil
				}

				return nil, nil
			}

			ds.TeamExistsFunc = func(ctx context.Context, teamID uint) (bool, error) {
				return false, nil
			}

			ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []mdmlab.MDMAssetName,
				_ sqlx.QueryerContext,
			) (map[mdmlab.MDMAssetName]mdmlab.MDMConfigAsset, error) {
				return map[mdmlab.MDMAssetName]mdmlab.MDMConfigAsset{}, nil
			}

			_, err = svc.DownloadSoftwareInstaller(ctx, false, "media", 1, tt.teamID)
			if tt.teamID == nil {
				require.Error(t, err)
			} else {
				checkAuthErr(t, tt.shouldFailRead, err)
			}

			err = svc.DeleteSoftwareInstaller(ctx, 1, tt.teamID)
			if tt.teamID == nil {
				require.Error(t, err)
			} else {
				checkAuthErr(t, tt.shouldFailWrite, err)
			}

			// Note: these calls always return an error because they're attempting to unmarshal a
			// non-existent VPP token.
			_, err = svc.GetAppStoreApps(ctx, tt.teamID)
			if tt.teamID == nil {
				require.Error(t, err)
			} else if tt.shouldFailRead {
				checkAuthErr(t, true, err)
			}

			err = svc.AddAppStoreApp(ctx, tt.teamID, mdmlab.VPPAppTeam{VPPAppID: mdmlab.VPPAppID{AdamID: "123", Platform: mdmlab.IOSPlatform}})
			if tt.teamID == nil {
				require.Error(t, err)
			} else if tt.shouldFailWrite {
				checkAuthErr(t, true, err)
			}

			// TODO: configure test with mock software installer store and add tests to check upload auth
		})
	}
}

func TestValidateSoftwareInstallerLabels(t *testing.T) {
	ds := new(mock.Store)

	license := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license})

	t.Run("validate no update", func(t *testing.T) {
		t.Run("no auth context", func(t *testing.T) {
			_, err := eeservice.ValidateSoftwareLabels(context.Background(), svc, nil, nil)
			require.ErrorContains(t, err, "Authentication required")
		})

		authCtx := authz_ctx.AuthorizationContext{}
		ctx = authz_ctx.NewContext(ctx, &authCtx)

		t.Run("no auth checked", func(t *testing.T) {
			_, err := eeservice.ValidateSoftwareLabels(ctx, svc, nil, nil)
			require.ErrorContains(t, err, "Authentication required")
		})

		// validator requires that an authz check has been performed upstream so we'll set it now for
		// the rest of the tests
		authCtx.SetChecked()

		mockLabels := map[string]uint{
			"foo": 1,
			"bar": 2,
			"baz": 3,
		}

		ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) (map[string]uint, error) {
			res := make(map[string]uint)
			if names == nil {
				return res, nil
			}
			for _, name := range names {
				if id, ok := mockLabels[name]; ok {
					res[name] = id
				}
			}
			return res, nil
		}

		testCases := []struct {
			name              string
			payloadIncludeAny []string
			payloadExcludeAny []string
			expectLabels      map[string]mdmlab.LabelIdent
			expectScope       mdmlab.LabelScope
			expectError       string
		}{
			{
				"no labels",
				nil,
				nil,
				nil,
				"",
				"",
			},
			{
				"include labels",
				[]string{"foo", "bar"},
				nil,
				map[string]mdmlab.LabelIdent{
					"foo": {LabelID: 1, LabelName: "foo"},
					"bar": {LabelID: 2, LabelName: "bar"},
				},
				mdmlab.LabelScopeIncludeAny,
				"",
			},
			{
				"exclude labels",
				nil,
				[]string{"bar", "baz"},
				map[string]mdmlab.LabelIdent{
					"bar": {LabelID: 2, LabelName: "bar"},
					"baz": {LabelID: 3, LabelName: "baz"},
				},
				mdmlab.LabelScopeExcludeAny,
				"",
			},
			{
				"include and exclude labels",
				[]string{"foo"},
				[]string{"bar"},
				nil,
				"",
				`Only one of "labels_include_any" or "labels_exclude_any" can be included.`,
			},
			{
				"non-existent label",
				[]string{"foo", "qux"},
				nil,
				nil,
				"",
				"some or all the labels provided don't exist",
			},
			{
				"duplicate label",
				[]string{"foo", "foo"},
				nil,
				map[string]mdmlab.LabelIdent{
					"foo": {LabelID: 1, LabelName: "foo"},
				},
				mdmlab.LabelScopeIncludeAny,
				"",
			},
			{
				"empty slice",
				nil,
				[]string{},
				nil,
				"",
				"",
			},
			{
				"empty string",
				nil,
				[]string{""},
				nil,
				"",
				"some or all the labels provided don't exist",
			},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				got, err := eeservice.ValidateSoftwareLabels(ctx, svc, tt.payloadIncludeAny, tt.payloadExcludeAny)
				if tt.expectError != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.expectError)
				} else {
					require.NoError(t, err)
					require.NotNil(t, got)
					require.Equal(t, tt.expectScope, got.LabelScope)
					require.Equal(t, tt.expectLabels, got.ByName)
				}
			})
		}
	})

	t.Run("validate update", func(t *testing.T) {
		t.Run("no auth context", func(t *testing.T) {
			_, _, err := eeservice.ValidateSoftwareLabelsForUpdate(context.Background(), svc, nil, nil, nil)
			require.ErrorContains(t, err, "Authentication required")
		})

		authCtx := authz_ctx.AuthorizationContext{}
		ctx = authz_ctx.NewContext(ctx, &authCtx)

		t.Run("no auth checked", func(t *testing.T) {
			_, _, err := eeservice.ValidateSoftwareLabelsForUpdate(ctx, svc, nil, nil, nil)
			require.ErrorContains(t, err, "Authentication required")
		})

		// validator requires that an authz check has been performed upstream so we'll set it now for
		// the rest of the tests
		authCtx.SetChecked()

		mockLabels := map[string]uint{
			"foo": 1,
			"bar": 2,
			"baz": 3,
		}

		ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) (map[string]uint, error) {
			res := make(map[string]uint)
			if names == nil {
				return res, nil
			}
			for _, name := range names {
				if id, ok := mockLabels[name]; ok {
					res[name] = id
				}
			}
			return res, nil
		}

		testCases := []struct {
			name              string
			existingInstaller *mdmlab.SoftwareInstaller
			payloadIncludeAny []string
			payloadExcludeAny []string
			shouldUpdate      bool
			expectLabels      map[string]mdmlab.LabelIdent
			expectScope       mdmlab.LabelScope
			expectError       string
		}{
			{
				"no installer",
				nil,
				nil,
				[]string{"foo"},
				false,
				nil,
				"",
				"existing installer must be provided",
			},
			{
				"no labels",
				&mdmlab.SoftwareInstaller{},
				nil,
				nil,
				false,
				nil,
				"",
				"",
			},
			{
				"add label",
				&mdmlab.SoftwareInstaller{
					LabelsIncludeAny: []mdmlab.SoftwareScopeLabel{{LabelID: 1, LabelName: "foo"}},
					LabelsExcludeAny: []mdmlab.SoftwareScopeLabel{},
				},
				[]string{"foo", "bar"},
				nil,
				true,
				map[string]mdmlab.LabelIdent{
					"foo": {LabelID: 1, LabelName: "foo"},
					"bar": {LabelID: 2, LabelName: "bar"},
				},
				mdmlab.LabelScopeIncludeAny,
				"",
			},
			{
				"change scope",
				&mdmlab.SoftwareInstaller{
					LabelsIncludeAny: []mdmlab.SoftwareScopeLabel{{LabelID: 1, LabelName: "foo"}},
					LabelsExcludeAny: []mdmlab.SoftwareScopeLabel{},
				},
				nil,
				[]string{"foo"},
				true,
				map[string]mdmlab.LabelIdent{
					"foo": {LabelID: 1, LabelName: "foo"},
				},
				mdmlab.LabelScopeExcludeAny,
				"",
			},
			{
				"remove label",
				&mdmlab.SoftwareInstaller{
					LabelsIncludeAny: []mdmlab.SoftwareScopeLabel{{LabelID: 1, LabelName: "foo"}},
					LabelsExcludeAny: []mdmlab.SoftwareScopeLabel{},
				},
				[]string{},
				nil,
				true,
				nil,
				"",
				"",
			},
			{
				"no change",
				&mdmlab.SoftwareInstaller{
					LabelsIncludeAny: []mdmlab.SoftwareScopeLabel{{LabelID: 1, LabelName: "foo"}},
					LabelsExcludeAny: []mdmlab.SoftwareScopeLabel{},
				},
				[]string{"foo"},
				nil,
				false,
				nil,
				"",
				"",
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				shouldUpate, got, err := eeservice.ValidateSoftwareLabelsForUpdate(ctx, svc, tt.existingInstaller, tt.payloadIncludeAny, tt.payloadExcludeAny)
				if tt.expectError != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.expectError)
				} else {
					require.NoError(t, err)
					if tt.shouldUpdate {
						require.True(t, shouldUpate)
						require.NotNil(t, got)
						require.Equal(t, tt.expectScope, got.LabelScope)
						require.Equal(t, tt.expectLabels, got.ByName)
					} else {
						require.False(t, shouldUpate)
						require.Nil(t, got)
					}
				}
			})
		}
	})
}
