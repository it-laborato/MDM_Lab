package service

import (
	"context"
	"testing"

	"github.com:it-laborato/MDM_Lab/server/authz"
	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreProcessUninstallScript(t *testing.T) {
	t.Parallel()
	input := `
blah$PACKAGE_IDS
pkgids=$PACKAGE_ID
they are $PACKAGE_ID, right $MY_SECRET?
quotes for "$PACKAGE_ID"
blah${PACKAGE_ID}withConcat
quotes and braces for "${PACKAGE_ID}"
${PACKAGE_ID}`

	payload := mdmlab.UploadSoftwareInstallerPayload{
		Extension:       "exe",
		UninstallScript: input,
		PackageIDs:      []string{"com.foo"},
	}

	preProcessUninstallScript(&payload)
	expected := `
blah$PACKAGE_IDS
pkgids="com.foo"
they are "com.foo", right $MY_SECRET?
quotes for "com.foo"
blah"com.foo"withConcat
quotes and braces for "com.foo"
"com.foo"`
	assert.Equal(t, expected, payload.UninstallScript)

	payload = mdmlab.UploadSoftwareInstallerPayload{
		Extension:       "pkg",
		UninstallScript: input,
		PackageIDs:      []string{"com.foo", "com.bar"},
	}
	preProcessUninstallScript(&payload)
	expected = `
blah$PACKAGE_IDS
pkgids=(
  "com.foo"
  "com.bar"
)
they are (
  "com.foo"
  "com.bar"
), right $MY_SECRET?
quotes for (
  "com.foo"
  "com.bar"
)
blah(
  "com.foo"
  "com.bar"
)withConcat
quotes and braces for (
  "com.foo"
  "com.bar"
)
(
  "com.foo"
  "com.bar"
)`
	assert.Equal(t, expected, payload.UninstallScript)
}

func TestInstallUninstallAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{
			ServerSettings: mdmlab.ServerSettings{ScriptsDisabled: true}, // global scripts being disabled shouldn't impact (un)installs
		}, nil
	}
	ds.HostFunc = func(ctx context.Context, id uint) (*mdmlab.Host, error) {
		return &mdmlab.Host{
			OrbitNodeKey: ptr.String("orbit_key"),
			Platform:     "darwin",
			TeamID:       ptr.Uint(1),
		}, nil
	}
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint,
		withScriptContents bool,
	) (*mdmlab.SoftwareInstaller, error) {
		return &mdmlab.SoftwareInstaller{
			Name:     "installer.pkg",
			Platform: "darwin",
			TeamID:   ptr.Uint(1),
		}, nil
	}
	ds.GetHostLastInstallDataFunc = func(ctx context.Context, hostID uint, installerID uint) (*mdmlab.HostLastInstallData, error) {
		return nil, nil
	}
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, selfService bool, policyID *uint) (string,
		error,
	) {
		return "request_id", nil
	}
	ds.GetAnyScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("script"), nil
	}
	ds.NewInternalScriptExecutionRequestFunc = func(ctx context.Context, request *mdmlab.HostScriptRequestPayload) (*mdmlab.HostScriptResult,
		error) {
		return &mdmlab.HostScriptResult{
			ExecutionID: "execution_id",
		}, nil
	}
	ds.InsertSoftwareUninstallRequestFunc = func(ctx context.Context, executionID string, hostID uint, softwareInstallerID uint) error {
		return nil
	}

	ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID, hostID uint) (bool, error) {
		return true, nil
	}

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
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			checkAuthErr(t, tt.shouldFail, svc.InstallSoftwareTitle(ctx, 1, 10))
			checkAuthErr(t, tt.shouldFail, svc.UninstallSoftwareTitle(ctx, 1, 10))
		})
	}
}

// TestUninstallSoftwareTitle is mostly tested in enterprise integration test. This test hits a few edge cases.
func TestUninstallSoftwareTitle(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	host := &mdmlab.Host{
		OrbitNodeKey: ptr.String("orbit_key"),
		Platform:     "darwin",
		TeamID:       ptr.Uint(1),
	}

	ds.HostFunc = func(ctx context.Context, id uint) (*mdmlab.Host, error) {
		return host, nil
	}

	// Global scripts disabled (doesn't matter)
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{
			ServerSettings: mdmlab.ServerSettings{
				ScriptsDisabled: true,
			},
		}, nil
	}

	// Host scripts disabled
	host.ScriptsEnabled = ptr.Bool(false)
	require.ErrorContains(t, svc.UninstallSoftwareTitle(context.Background(), 1, 10), mdmlab.RunScriptsOrbitDisabledErrMsg)
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

func newTestService(t *testing.T, ds mdmlab.Datastore) *Service {
	t.Helper()
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{
		authz: authorizer,
		ds:    ds,
	}
	return svc
}
