package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mixer/clock"
	eeservice "github.com/it-laborato/MDM_Lab/ee/server/service"
	"github.com/it-laborato/MDM_Lab/pkg/optjson"
	"github.com/it-laborato/MDM_Lab/server/config"
	authz_ctx "github.com/it-laborato/MDM_Lab/server/contexts/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/contexts/license"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/datastore/mysql"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mdm/apple/mobileconfig"
	nanodep_client "github.com/it-laborato/MDM_Lab/server/mdm/nanodep/client"
	"github.com/it-laborato/MDM_Lab/server/mock"
	nanodep_mock "github.com/it-laborato/MDM_Lab/server/mock/nanodep"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/service"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/it-laborato/MDM_Lab/server/worker"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func setupMockDatastorePremiumService(t testing.TB) (*mock.Store, *eeservice.Service, context.Context) {
	ds := new(mock.Store)
	lic := &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}
	ctx := license.NewContext(context.Background(), lic)

	logger := kitlog.NewNopLogger()
	mdmlabConfig := config.MDMlabConfig{
		MDM: config.MDMConfig{
			AppleSCEPCertBytes: eeservice.TestCert,
			AppleSCEPKeyBytes:  eeservice.TestKey,
		},
		Server: config.ServerConfig{
			PrivateKey: "foo",
		},
	}
	depStorage := &nanodep_mock.Storage{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/server/devices"):
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
		case strings.Contains(r.URL.Path, "/session"):
			_, err := w.Write([]byte(`{"auth_session_token": "yoo"}`))
			require.NoError(t, err)
		case strings.Contains(r.URL.Path, "/profile"):
			_, err := w.Write([]byte(`{"profile_uuid": "profile123"}`))
			require.NoError(t, err)
		}
	}))
	depStorage.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{
			BaseURL: ts.URL,
		}, nil
	}
	depStorage.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
		return &nanodep_client.OAuth1Tokens{}, nil
	}
	t.Cleanup(func() { ts.Close() })

	freeSvc, err := service.NewService(
		ctx,
		ds,
		nil,
		nil,
		logger,
		nil,
		mdmlabConfig,
		nil,
		clock.C,
		nil,
		nil,
		ds,
		nil,
		nil,
		&mdmlab.NoOpGeoIP{},
		nil,
		depStorage,
		nil,
		nil,
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}
	svc, err := eeservice.NewService(
		freeSvc,
		ds,
		logger,
		mdmlabConfig,
		nil,
		clock.C,
		depStorage,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if err != nil {
		panic(err)
	}
	return ds, svc, ctx
}

func TestGetOrCreatePreassignTeam(t *testing.T) {
	ds, svc, ctx := setupMockDatastorePremiumService(t)

	ssoSettings := mdmlab.SSOProviderSettings{
		EntityID:    "foo",
		MetadataURL: "https://example.com/metadata.xml",
		IssuerURI:   "https://example.com",
	}
	appConfig := &mdmlab.AppConfig{MDM: mdmlab.MDM{
		EnabledAndConfigured:  true,
		EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: ssoSettings},
		MacOSSetup: mdmlab.MacOSSetup{
			BootstrapPackage:            optjson.SetString("https://example.com/bootstrap.pkg"),
			EnableEndUserAuthentication: true,
			EnableReleaseDeviceManually: optjson.SetBool(true),
		},
	}}
	preassignGroups := []string{"one", "three"}

	// initialize team store with team one and two already created, one matches
	// preassign group [0], two does not match any preassign group
	team1 := &mdmlab.Team{
		ID:   1,
		Name: preassignGroups[0],
	}
	team2 := &mdmlab.Team{
		ID:   2,
		Name: "two",
		Config: mdmlab.TeamConfig{
			MDM: mdmlab.TeamMDM{
				MacOSSetup: mdmlab.MacOSSetup{MacOSSetupAssistant: optjson.SetString("foo/bar")},
			},
		},
	}
	teamStore := map[uint]*mdmlab.Team{1: team1, 2: team2}

	resetInvoked := func() {
		ds.TeamByNameFuncInvoked = false
		ds.NewTeamFuncInvoked = false
		ds.SaveTeamFuncInvoked = false
		ds.NewMDMAppleConfigProfileFuncInvoked = false
		ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked = false
		ds.AppConfigFuncInvoked = false
		ds.NewJobFuncInvoked = false
		ds.GetMDMAppleSetupAssistantFuncInvoked = false
		ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked = false
		ds.LabelIDsByNameFuncInvoked = false
		ds.SetOrUpdateMDMAppleDeclarationFuncInvoked = false
		ds.BulkSetPendingMDMHostProfilesFuncInvoked = false
		ds.GetAllMDMConfigAssetsByNameFuncInvoked = false
	}
	setupDS := func(t *testing.T) {
		resetInvoked()

		ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
			return appConfig, nil
		}
		ds.NewActivityFunc = func(ctx context.Context, u *mdmlab.User, a mdmlab.ActivityDetails, details []byte, createdAt time.Time) error {
			return nil
		}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
			for _, team := range teamStore {
				if team.Name == name {
					return team, nil
				}
			}
			return nil, ctxerr.Wrap(ctx, &eeservice.NotFoundError{})
		}
		ds.TeamFunc = func(ctx context.Context, id uint) (*mdmlab.Team, error) {
			tm, ok := teamStore[id]
			if !ok {
				return nil, errors.New("team id not found")
			}
			if id != tm.ID {
				// sanity chec
				return nil, errors.New("team id mismatch")
			}
			return tm, nil
		}
		ds.NewTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			return nil, errors.New("not implemented")
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			return nil, errors.New("not implemented")
		}
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, profile mdmlab.MDMAppleConfigProfile) (*mdmlab.MDMAppleConfigProfile, error) {
			return nil, errors.New("not implemented")
		}
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
			return errors.New("not implemented")
		}
		ds.CopyDefaultMDMAppleBootstrapPackageFunc = func(ctx context.Context, ac *mdmlab.AppConfig, toTeamID uint) error {
			return errors.New("not implemented")
		}
		ds.NewJobFunc = func(ctx context.Context, job *mdmlab.Job) (*mdmlab.Job, error) {
			return nil, errors.New("not implemented")
		}
		ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*mdmlab.MDMAppleSetupAssistant, error) {
			return nil, errors.New("not implemented")
		}
		ds.LabelIDsByNameFunc = func(ctx context.Context, names []string) (map[string]uint, error) {
			require.Len(t, names, 1)
			require.ElementsMatch(t, names, []string{mdmlab.BuiltinLabelMacOS14Plus})
			return map[string]uint{names[0]: 1}, nil
		}
		ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *mdmlab.MDMAppleDeclaration) (*mdmlab.MDMAppleDeclaration, error) {
			declaration.DeclarationUUID = uuid.NewString()
			return declaration, nil
		}
		ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs, teamIDs []uint, profileUUIDs, hostUUIDs []string,
		) (updates mdmlab.MDMProfilesUpdates, err error) {
			return mdmlab.MDMProfilesUpdates{}, nil
		}
		apnsCert, apnsKey, err := mysql.GenerateTestCertBytes()
		require.NoError(t, err)
		certPEM, keyPEM, tokenBytes, err := mysql.GenerateTestABMAssets(t)
		require.NoError(t, err)
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []mdmlab.MDMAssetName,
			_ sqlx.QueryerContext) (map[mdmlab.MDMAssetName]mdmlab.MDMConfigAsset, error) {
			return map[mdmlab.MDMAssetName]mdmlab.MDMConfigAsset{
				mdmlab.MDMAssetABMCert:            {Name: mdmlab.MDMAssetABMCert, Value: certPEM},
				mdmlab.MDMAssetABMKey:             {Name: mdmlab.MDMAssetABMKey, Value: keyPEM},
				mdmlab.MDMAssetABMTokenDeprecated: {Name: mdmlab.MDMAssetABMTokenDeprecated, Value: tokenBytes},
				mdmlab.MDMAssetAPNSCert:           {Name: mdmlab.MDMAssetAPNSCert, Value: apnsCert},
				mdmlab.MDMAssetAPNSKey:            {Name: mdmlab.MDMAssetAPNSKey, Value: apnsKey},
				mdmlab.MDMAssetCACert:             {Name: mdmlab.MDMAssetCACert, Value: certPEM},
				mdmlab.MDMAssetCAKey:              {Name: mdmlab.MDMAssetCAKey, Value: keyPEM},
			}, nil
		}

		ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ mdmlab.MDMAppleEnrollmentType) (*mdmlab.MDMAppleEnrollmentProfile, error) {
			return &mdmlab.MDMAppleEnrollmentProfile{Token: "foobar"}, nil
		}
		ds.GetABMTokenOrgNamesAssociatedWithTeamFunc = func(ctx context.Context, teamID *uint) ([]string, error) {
			return []string{"foobar"}, nil
		}
		ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
			return []*mdmlab.ABMToken{{ID: 1}}, nil
		}
		ds.CountABMTokensWithTermsExpiredFunc = func(ctx context.Context) (int, error) {
			return 0, nil
		}
	}

	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, authzCtx)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: test.UserAdmin})
	actx, _ := authz_ctx.FromContext(ctx)
	actx.SetChecked()

	t.Run("get preassign team", func(t *testing.T) {
		setupDS(t)

		// preasign group corresponds to existing team so simply get it
		team, err := svc.GetOrCreatePreassignTeam(ctx, preassignGroups[0:1])
		require.NoError(t, err)
		require.Equal(t, uint(1), team.ID)
		require.Equal(t, preassignGroups[0], team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("create preassign team", func(t *testing.T) {
		setupDS(t)

		lastTeamID := uint(0)
		ds.NewTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1) //nolint:gosec // dismiss G115
			_, ok := teamStore[id]
			require.False(t, ok) // sanity check
			team.ID = id
			teamStore[id] = team
			lastTeamID = id
			return team, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)     // sanity check
			require.Equal(t, tm.Name, team.Name) // sanity check
			// NOTE: BootstrapPackage gets set by CopyDefaultMDMAppleBootstrapPackage
			// require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)             // set to default
			require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant, team.Config.MDM.MacOSSetup.MacOSSetupAssistant)                             // set to default
			require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // set to default
			teamStore[tm.ID] = team
			return team, nil
		}
		ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, profile mdmlab.MDMAppleConfigProfile) (*mdmlab.MDMAppleConfigProfile, error) {
			require.Equal(t, lastTeamID, *profile.TeamID)
			require.Equal(t, mobileconfig.MDMlabFileVaultPayloadIdentifier, profile.Identifier)
			return &profile, nil
		}
		ds.DeleteMDMAppleConfigProfileByTeamAndIdentifierFunc = func(ctx context.Context, teamID *uint, profileIdentifier string) error {
			require.Equal(t, lastTeamID, *teamID)
			require.Equal(t, mobileconfig.MDMlabFileVaultPayloadIdentifier, profileIdentifier)
			return nil
		}
		ds.CopyDefaultMDMAppleBootstrapPackageFunc = func(ctx context.Context, ac *mdmlab.AppConfig, toTeamID uint) error {
			require.Equal(t, lastTeamID, toTeamID)
			require.NotNil(t, ac)
			require.Equal(t, "https://example.com/bootstrap.pkg", ac.MDM.MacOSSetup.BootstrapPackage.Value)
			teamStore[toTeamID].Config.MDM.MacOSSetup.BootstrapPackage = optjson.SetString(ac.MDM.MacOSSetup.BootstrapPackage.Value)
			return nil
		}
		var jobTask string
		ds.NewJobFunc = func(ctx context.Context, job *mdmlab.Job) (*mdmlab.Job, error) {
			// first task is UpdateProfile, next is ProfileChanged (when setup assistant is set)
			if jobTask == "" {
				jobTask = string(worker.MacosSetupAssistantUpdateProfile)
			} else {
				jobTask = string(worker.MacosSetupAssistantProfileChanged)
			}
			wantArgs, err := json.Marshal(map[string]interface{}{
				"task":    jobTask,
				"team_id": lastTeamID,
			})
			require.NoError(t, err)
			wantJob := &mdmlab.Job{
				Name:  "macos_setup_assistant",
				Args:  (*json.RawMessage)(&wantArgs),
				State: mdmlab.JobStateQueued,
			}
			require.Equal(t, wantJob.Name, job.Name)
			require.Equal(t, string(*wantJob.Args), string(*job.Args))
			require.Equal(t, wantJob.State, job.State)
			return job, nil
		}
		setupAsstByTeam := make(map[uint]*mdmlab.MDMAppleSetupAssistant)
		globalSetupAsst := &mdmlab.MDMAppleSetupAssistant{
			ID:      15,
			TeamID:  nil,
			Name:    "test asst",
			Profile: json.RawMessage(`{"foo": "bar"}`),
		}
		setupAsstByTeam[0] = globalSetupAsst
		ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*mdmlab.MDMAppleSetupAssistant, error) {
			var tmID uint
			if teamID != nil {
				tmID = *teamID
			}
			asst := setupAsstByTeam[tmID]
			if asst == nil {
				return nil, eeservice.NotFoundError{}
			}
			return asst, nil
		}
		ds.SetOrUpdateMDMAppleSetupAssistantFunc = func(ctx context.Context, asst *mdmlab.MDMAppleSetupAssistant) (*mdmlab.MDMAppleSetupAssistant, error) {
			require.Equal(t, globalSetupAsst.Name, asst.Name)
			require.JSONEq(t, string(globalSetupAsst.Profile), string(asst.Profile))
			require.NotNil(t, asst.TeamID)
			require.EqualValues(t, lastTeamID, *asst.TeamID)
			setupAsstByTeam[*asst.TeamID] = asst
			return asst, nil
		}

		// new team ("one - three") is created with bootstrap package and end user auth based on app config
		team, err := svc.GetOrCreatePreassignTeam(ctx, preassignGroups)
		require.NoError(t, err)
		require.Equal(t, uint(3), team.ID)
		require.Equal(t, eeservice.TeamNameFromPreassignGroups(preassignGroups), team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.True(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.GetMDMAppleSetupAssistantFuncInvoked)
		require.True(t, ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked)
		require.NotEmpty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant.Value, team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
		require.True(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.True(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.True(t, ds.NewJobFuncInvoked)
		resetInvoked()

		jobTask = ""

		// when called again, simply get the previously created team
		team, err = svc.GetOrCreatePreassignTeam(ctx, preassignGroups)
		require.NoError(t, err)
		require.Equal(t, uint(3), team.ID)
		require.Equal(t, eeservice.TeamNameFromPreassignGroups(preassignGroups), team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		require.False(t, ds.GetMDMAppleSetupAssistantFuncInvoked)
		require.False(t, ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked)
		require.NotEmpty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.MacOSSetupAssistant.Value, team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
		require.True(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.True(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		resetInvoked()

		jobTask = ""

		// when a custom setup assistant is not set for "no team", we don't create
		// a custom setup assistant
		setupAsstByTeam[0] = nil

		preassignGrousWithFoo := preassignGroups
		preassignGrousWithFoo = append(preassignGrousWithFoo, "foo")
		team, err = svc.GetOrCreatePreassignTeam(ctx, preassignGrousWithFoo)
		require.NoError(t, err)
		require.Equal(t, uint(4), team.ID)
		require.Equal(t, eeservice.TeamNameFromPreassignGroups(preassignGrousWithFoo), team.Name)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.True(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.GetMDMAppleSetupAssistantFuncInvoked)
		require.False(t, ds.SetOrUpdateMDMAppleSetupAssistantFuncInvoked)
		require.NotEmpty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.BootstrapPackage.Value, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
		require.True(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableEndUserAuthentication, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.True(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		require.Equal(t, appConfig.MDM.MacOSSetup.EnableReleaseDeviceManually.Value, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
		resetInvoked()
	})

	t.Run("modify team via apply team spec", func(t *testing.T) {
		setupDS(t)

		ds.SaveTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)                                               // sanity check
			require.Equal(t, tm.Name, team.Name)                                           // sanity check
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not modified
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not modified
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not modified
			return teamStore[tm.ID], nil
		}

		// apply team spec does not apply defaults
		spec := &mdmlab.TeamSpec{
			Name: team2.Name,
		}
		_, err := svc.ApplyTeamSpecs(ctx, []*mdmlab.TeamSpec{spec}, mdmlab.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("new team", func(t *testing.T) {
		setupDS(t)

		ds.NewTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1) //nolint:gosec // dismiss G115
			_, ok := teamStore[id]
			require.False(t, ok) // sanity check
			require.Equal(t, "new team", team.Name)
			require.Equal(t, "new description", team.Description)
			require.False(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not set
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not set
			team.ID = id
			teamStore[id] = team
			return team, nil
		}

		// new team does not apply defaults
		_, err := svc.NewTeam(ctx, mdmlab.TeamPayload{
			Name:        ptr.String("new team"),
			Description: ptr.String("new description"),
		})
		require.NoError(t, err)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.False(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})

	t.Run("apply team spec", func(t *testing.T) {
		setupDS(t)

		ds.NewTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			for _, tm := range teamStore {
				if tm.Name == team.Name {
					return nil, errors.New("team name already exists")
				}
			}
			id := uint(len(teamStore) + 1) //nolint:gosec // dismiss G115
			_, ok := teamStore[id]
			require.False(t, ok)                                                           // sanity check
			require.Equal(t, "new team spec", team.Name)                                   // set
			require.Equal(t, "12.0", team.Config.MDM.MacOSUpdates.MinimumVersion.Value)    // set
			require.Equal(t, "2024-01-01", team.Config.MDM.MacOSUpdates.Deadline.Value)    // set
			require.False(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not set
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not set
			team.ID = id
			teamStore[id] = team
			return team, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *mdmlab.Team) (*mdmlab.Team, error) {
			tm, ok := teamStore[team.ID]
			if !ok {
				return nil, errors.New("invalid team id")
			}
			require.Equal(t, tm.ID, team.ID)                                               // sanity check
			require.Equal(t, tm.Name, team.Name)                                           // sanity check
			require.Equal(t, "12.0", team.Config.MDM.MacOSUpdates.MinimumVersion.Value)    // unchanged
			require.Equal(t, "2025-01-01", team.Config.MDM.MacOSUpdates.Deadline.Value)    // modified
			require.Empty(t, team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)       // not set
			require.Empty(t, team.Config.MDM.MacOSSetup.BootstrapPackage.Value)            // not set
			require.False(t, team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value) // not set

			return teamStore[tm.ID], nil
		}

		spec := &mdmlab.TeamSpec{
			Name: "new team spec",
			MDM: mdmlab.TeamSpecMDM{
				MacOSUpdates: mdmlab.AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("12.0"),
					Deadline:       optjson.SetString("2024-01-01"),
				},
			},
		}

		// apply team spec creates new team without defaults
		_, err := svc.ApplyTeamSpecs(ctx, []*mdmlab.TeamSpec{spec}, mdmlab.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.NewTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.SaveTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()

		// apply team spec edits existing team without applying defaults
		spec.MDM.MacOSUpdates.Deadline = optjson.SetString("2025-01-01")
		_, err = svc.ApplyTeamSpecs(ctx, []*mdmlab.TeamSpec{spec}, mdmlab.ApplyTeamSpecOptions{})
		require.NoError(t, err)
		require.True(t, ds.SaveTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamByNameFuncInvoked)
		require.False(t, ds.NewTeamFuncInvoked)
		require.False(t, ds.NewMDMAppleConfigProfileFuncInvoked)
		require.False(t, ds.CopyDefaultMDMAppleBootstrapPackageFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		resetInvoked()
	})
}
