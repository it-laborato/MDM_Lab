package service

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/it-laborato/MDM_Lab/pkg/optjson"
	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	nanodep_client "github.com/it-laborato/MDM_Lab/server/mdm/nanodep/client"
	"github.com/it-laborato/MDM_Lab/server/mock"
	nanodep_mock "github.com/it-laborato/MDM_Lab/server/mock/nanodep"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfigAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// start a TLS server and use its URL as the server URL in the app config,
	// required by the CertificateChain service call.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{
			OrgInfo: mdmlab.OrgInfo{
				OrgName: "Test",
			},
			ServerSettings: mdmlab.ServerSettings{
				ServerURL: srv.URL,
			},
		}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
		return nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
		return []*mdmlab.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
		return []*mdmlab.ABMToken{}, nil
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
			true,
			false,
		},
		{
			"global observer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
			true,
			false,
		},
		{
			"global observer+",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			true,
			false,
		},
		{
			"global gitops",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			false,
			false,
		},
		{
			"team admin",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			true,
			false,
		},
		{
			"team maintainer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team observer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			true,
			false,
		},
		{
			"team observer+",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			true,
			false,
		},
		{
			"team gitops",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			true,
			true,
		},
		{
			"user without roles",
			&mdmlab.User{ID: 777},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.AppConfigObfuscated(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyAppConfig(ctx, []byte(`{}`), mdmlab.ApplySpecOptions{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.CertificateChain(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

// TestVersion tests that all users can access the version endpoint.
func TestVersion(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	testCases := []struct {
		name string
		user *mdmlab.User
	}{
		{
			"global admin",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
		},
		{
			"global maintainer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
		},
		{
			"global observer",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
		},
		{
			"global observer+",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
		},
		{
			"global gitops",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
		},
		{
			"team admin",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
		},
		{
			"team maintainer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
		},
		{
			"team observer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
		},
		{
			"team observer+",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
		},
		{
			"team gitops",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
		},
		{
			"user without roles",
			&mdmlab.User{ID: 777},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			_, err := svc.Version(ctx)
			require.NoError(t, err)
		})
	}
}

func TestEnrollSecretAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, tid *uint, secrets []*mdmlab.EnrollSecret) error {
		return nil
	}
	ds.GetEnrollSecretsFunc = func(ctx context.Context, tid *uint) ([]*mdmlab.EnrollSecret, error) {
		return nil, nil
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
			true,
		},
		{
			"team admin",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
			true,
			true,
		},
		{
			"user",
			&mdmlab.User{ID: 777},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.ApplyEnrollSecretSpec(
				ctx, &mdmlab.EnrollSecretSpec{Secrets: []*mdmlab.EnrollSecret{{Secret: "ABC"}}}, mdmlab.ApplySpecOptions{},
			)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetEnrollSecretSpec(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

func TestApplyEnrollSecretWithGlobalEnrollConfig(t *testing.T) {
	ds := new(mock.Store)

	cfg := config.TestConfig()
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil)
	ctx = test.UserContext(ctx, test.UserAdmin)

	// Dry run
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		assert.False(t, new)
		assert.Nil(t, teamID)
		return true, nil
	}
	err := svc.ApplyEnrollSecretSpec(
		ctx, &mdmlab.EnrollSecretSpec{Secrets: []*mdmlab.EnrollSecret{{Secret: "ABC"}}}, mdmlab.ApplySpecOptions{DryRun: true},
	)
	assert.True(t, ds.IsEnrollSecretAvailableFuncInvoked)
	assert.NoError(t, err)

	// Dry run fails
	ds.IsEnrollSecretAvailableFuncInvoked = false
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		assert.False(t, new)
		assert.Nil(t, teamID)
		return false, nil
	}
	err = svc.ApplyEnrollSecretSpec(
		ctx, &mdmlab.EnrollSecretSpec{Secrets: []*mdmlab.EnrollSecret{{Secret: "ABC"}}}, mdmlab.ApplySpecOptions{DryRun: true},
	)
	assert.True(t, ds.IsEnrollSecretAvailableFuncInvoked)
	assert.ErrorContains(t, err, "secret is already being used")

	// Dry run with error
	ds.IsEnrollSecretAvailableFuncInvoked = false
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return false, assert.AnError
	}
	err = svc.ApplyEnrollSecretSpec(
		ctx, &mdmlab.EnrollSecretSpec{Secrets: []*mdmlab.EnrollSecret{{Secret: "ABC"}}}, mdmlab.ApplySpecOptions{DryRun: true},
	)
	assert.True(t, ds.IsEnrollSecretAvailableFuncInvoked)
	assert.Equal(t, assert.AnError, err)

	ds.IsEnrollSecretAvailableFunc = nil
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*mdmlab.EnrollSecret) error {
		return nil
	}
	err = svc.ApplyEnrollSecretSpec(
		ctx, &mdmlab.EnrollSecretSpec{Secrets: []*mdmlab.EnrollSecret{{Secret: "ABC"}}}, mdmlab.ApplySpecOptions{},
	)
	require.True(t, ds.ApplyEnrollSecretsFuncInvoked)
	require.NoError(t, err)

	// try to change the enroll secret with the config set
	ds.ApplyEnrollSecretsFuncInvoked = false
	cfg.Packaging.GlobalEnrollSecret = "xyz"
	svc, ctx = newTestServiceWithConfig(t, ds, cfg, nil, nil)
	ctx = test.UserContext(ctx, test.UserAdmin)
	err = svc.ApplyEnrollSecretSpec(ctx, &mdmlab.EnrollSecretSpec{Secrets: []*mdmlab.EnrollSecret{{Secret: "DEF"}}}, mdmlab.ApplySpecOptions{})
	require.Error(t, err)
	require.False(t, ds.ApplyEnrollSecretsFuncInvoked)
}

func TestCertificateChain(t *testing.T) {
	server, teardown := setupCertificateChain(t)
	defer teardown()

	certFile := "testdata/server.pem"
	cert, err := tls.LoadX509KeyPair(certFile, "testdata/server.key")
	require.Nil(t, err)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server.StartTLS()

	u, err := url.Parse(server.URL)
	require.Nil(t, err)

	conn, err := connectTLS(context.Background(), u)
	require.Nil(t, err)

	have, want := len(conn.ConnectionState().PeerCertificates), len(cert.Certificate)
	require.Equal(t, have, want)

	original, _ := os.ReadFile(certFile)
	returned, err := chain(context.Background(), conn.ConnectionState(), "")
	require.Nil(t, err)
	require.Equal(t, returned, original)
}

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(dump) //nolint:errcheck
	})
}

func setupCertificateChain(t *testing.T) (server *httptest.Server, teardown func()) {
	server = httptest.NewUnstartedServer(echoHandler())
	return server, server.Close
}

func TestSSONotPresent(t *testing.T) {
	invalid := &mdmlab.InvalidArgumentError{}
	var p mdmlab.AppConfig
	validateSSOSettings(p, &mdmlab.AppConfig{}, invalid, &mdmlab.LicenseInfo{})
	assert.False(t, invalid.HasErrors())
}

func TestNeedFieldsPresent(t *testing.T) {
	invalid := &mdmlab.InvalidArgumentError{}
	config := mdmlab.AppConfig{
		SSOSettings: &mdmlab.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:    "mdmlab",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				IDPName:     "onelogin",
			},
		},
	}
	validateSSOSettings(config, &mdmlab.AppConfig{}, invalid, &mdmlab.LicenseInfo{})
	assert.False(t, invalid.HasErrors())
}

func TestShortIDPName(t *testing.T) {
	invalid := &mdmlab.InvalidArgumentError{}
	config := mdmlab.AppConfig{
		SSOSettings: &mdmlab.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:    "mdmlab",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				// A customer once found the MDMlab server erroring when they used "SSO" for their IdP name.
				IDPName: "SSO",
			},
		},
	}
	validateSSOSettings(config, &mdmlab.AppConfig{}, invalid, &mdmlab.LicenseInfo{})
	assert.False(t, invalid.HasErrors())
}

func TestMissingMetadata(t *testing.T) {
	invalid := &mdmlab.InvalidArgumentError{}
	config := mdmlab.AppConfig{
		SSOSettings: &mdmlab.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:  "mdmlab",
				IssuerURI: "http://issuer.idp.com",
				IDPName:   "onelogin",
			},
		},
	}
	validateSSOSettings(config, &mdmlab.AppConfig{}, invalid, &mdmlab.LicenseInfo{})
	require.True(t, invalid.HasErrors())
	assert.Contains(t, invalid.Error(), "metadata")
	assert.Contains(t, invalid.Error(), "either metadata or metadata_url must be defined")
}

func TestSSOValidationValidatesSchemaInMetadataURL(t *testing.T) {
	var schemas []string
	schemas = append(schemas, getURISchemas()...)
	schemas = append(schemas, "asdfaklsdfjalksdfja")

	for _, scheme := range schemas {
		actual := &mdmlab.InvalidArgumentError{}
		sut := mdmlab.AppConfig{
			SSOSettings: &mdmlab.SSOSettings{
				EnableSSO: true,
				SSOProviderSettings: mdmlab.SSOProviderSettings{
					EntityID:    "mdmlab",
					IDPName:     "onelogin",
					MetadataURL: fmt.Sprintf("%s://somehost", scheme),
				},
			},
		}

		validateSSOSettings(sut, &mdmlab.AppConfig{}, actual, &mdmlab.LicenseInfo{})

		require.Equal(t, scheme == "http" || scheme == "https", !actual.HasErrors())
		require.Equal(t, scheme == "http" || scheme == "https", !strings.Contains(actual.Error(), "metadata_url"))
		require.Equal(t, scheme == "http" || scheme == "https", !strings.Contains(actual.Error(), "must be either https or http"))
	}
}

func TestJITProvisioning(t *testing.T) {
	config := mdmlab.AppConfig{
		SSOSettings: &mdmlab.SSOSettings{
			EnableSSO:             true,
			EnableJITProvisioning: true,
			SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:    "mdmlab",
				IssuerURI:   "http://issuer.idp.com",
				IDPName:     "onelogin",
				MetadataURL: "http://isser.metadata.com",
			},
		},
	}

	t.Run("doesn't allow to enable JIT provisioning without a premium license", func(t *testing.T) {
		invalid := &mdmlab.InvalidArgumentError{}
		validateSSOSettings(config, &mdmlab.AppConfig{}, invalid, &mdmlab.LicenseInfo{})
		require.True(t, invalid.HasErrors())
		assert.Contains(t, invalid.Error(), "enable_jit_provisioning")
		assert.Contains(t, invalid.Error(), "missing or invalid license")
	})

	t.Run("allows JIT provisioning to be enabled with a premium license", func(t *testing.T) {
		invalid := &mdmlab.InvalidArgumentError{}
		validateSSOSettings(config, &mdmlab.AppConfig{}, invalid, &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium})
		require.False(t, invalid.HasErrors())
	})

	t.Run("doesn't care if JIT provisioning is set to false on free licenses", func(t *testing.T) {
		invalid := &mdmlab.InvalidArgumentError{}
		oldConfig := &mdmlab.AppConfig{
			SSOSettings: &mdmlab.SSOSettings{
				EnableJITProvisioning: false,
			},
		}
		config.SSOSettings.EnableJITProvisioning = false
		validateSSOSettings(config, oldConfig, invalid, &mdmlab.LicenseInfo{})
		require.False(t, invalid.HasErrors())
	})
}

func TestAppConfigSecretsObfuscated(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// start a TLS server and use its URL as the server URL in the app config,
	// required by the CertificateChain service call.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{
			SMTPSettings: &mdmlab.SMTPSettings{
				SMTPPassword: "smtppassword",
			},
			Integrations: mdmlab.Integrations{
				Jira: []*mdmlab.JiraIntegration{
					{APIToken: "jiratoken"},
				},
				Zendesk: []*mdmlab.ZendeskIntegration{
					{APIToken: "zendesktoken"},
				},
				GoogleCalendar: []*mdmlab.GoogleCalendarIntegration{
					{ApiKey: map[string]string{mdmlab.GoogleCalendarPrivateKey: "google-calendar-private-key"}},
				},
			},
		}, nil
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
			false,
		},
		{
			"global observer+",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
			false,
		},
		{
			"global gitops",
			&mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
			false,
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
			false,
		},
		{
			"team observer+",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
			false,
		},
		{
			"team gitops",
			&mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
			true,
		},
		{
			"user without roles",
			&mdmlab.User{ID: 777},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			ac, err := svc.AppConfigObfuscated(ctx)
			if tt.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, ac.SMTPSettings.SMTPPassword, mdmlab.MaskedPassword)
				require.Equal(t, ac.Integrations.Jira[0].APIToken, mdmlab.MaskedPassword)
				require.Equal(t, ac.Integrations.Zendesk[0].APIToken, mdmlab.MaskedPassword)
				// Google Calendar private key is not obfuscated
				require.Equal(t, ac.Integrations.GoogleCalendar[0].ApiKey[mdmlab.GoogleCalendarPrivateKey], "google-calendar-private-key")
			}
		})
	}
}

// TestModifyAppConfigSMTPConfigured tests that disabling SMTP
// should set the SMTPConfigured field to false.
func TestModifyAppConfigSMTPConfigured(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// SMTP is initially enabled and configured.
	dsAppConfig := &mdmlab.AppConfig{
		OrgInfo: mdmlab.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: mdmlab.ServerSettings{
			ServerURL: "https://example.org",
		},
		SMTPSettings: &mdmlab.SMTPSettings{
			SMTPEnabled:    true,
			SMTPConfigured: true,
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return dsAppConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
		*dsAppConfig = *conf
		return nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
		return []*mdmlab.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
		return []*mdmlab.ABMToken{}, nil
	}

	// Disable SMTP.
	newAppConfig := mdmlab.AppConfig{
		SMTPSettings: &mdmlab.SMTPSettings{
			SMTPEnabled:    false,
			SMTPConfigured: true,
		},
	}
	b, err := json.Marshal(newAppConfig.SMTPSettings) // marshaling appconfig sets all fields, resetting e.g. OrgName to empty
	require.NoError(t, err)
	b = []byte(`{"smtp_settings":` + string(b) + `}`)

	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	updatedAppConfig, err := svc.ModifyAppConfig(ctx, b, mdmlab.ApplySpecOptions{})
	require.NoError(t, err)

	// After disabling SMTP, the app config should be "not configured".
	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, updatedAppConfig.SMTPSettings.SMTPConfigured)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPConfigured)
}

// TestTransparencyURL tests that MDMlab Premium licensees can use custom transparency urls and MDMlab
// Free licensees are restricted to the default transparency url.
func TestTransparencyURL(t *testing.T) {
	ds := new(mock.Store)

	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}

	checkLicenseErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.ErrorContains(t, err, "missing or invalid license")
		} else {
			require.NoError(t, err)
		}
	}
	testCases := []struct {
		name             string
		licenseTier      string
		initialURL       string
		newURL           string
		expectedURL      string
		shouldFailModify bool
	}{
		{
			name:             "customURL",
			licenseTier:      "free",
			initialURL:       "",
			newURL:           "customURL",
			expectedURL:      "",
			shouldFailModify: true,
		},
		{
			name:             "customURL",
			licenseTier:      mdmlab.TierPremium,
			initialURL:       "",
			newURL:           "customURL",
			expectedURL:      "customURL",
			shouldFailModify: false,
		},
		{
			name:             "emptyURL",
			licenseTier:      "free",
			initialURL:       "",
			newURL:           "",
			expectedURL:      "",
			shouldFailModify: false,
		},
		{
			name:             "emptyURL",
			licenseTier:      mdmlab.TierPremium,
			initialURL:       "customURL",
			newURL:           "",
			expectedURL:      "",
			shouldFailModify: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: tt.licenseTier}})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

			dsAppConfig := &mdmlab.AppConfig{
				OrgInfo: mdmlab.OrgInfo{
					OrgName: "Test",
				},
				ServerSettings: mdmlab.ServerSettings{
					ServerURL: "https://example.org",
				},
				MDMlabDesktop: mdmlab.MDMlabDesktopSettings{TransparencyURL: tt.initialURL},
			}

			ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
				*dsAppConfig = *conf
				return nil
			}

			ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
				return nil
			}

			ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
				return []*mdmlab.VPPTokenDB{}, nil
			}

			ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
				return []*mdmlab.ABMToken{}, nil
			}

			ac, err := svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.initialURL, ac.MDMlabDesktop.TransparencyURL)

			raw, err := json.Marshal(mdmlab.MDMlabDesktopSettings{TransparencyURL: tt.newURL})
			require.NoError(t, err)
			raw = []byte(`{"mdmlab_desktop":` + string(raw) + `}`)
			modified, err := svc.ModifyAppConfig(ctx, raw, mdmlab.ApplySpecOptions{})
			checkLicenseErr(t, tt.shouldFailModify, err)

			if modified != nil {
				require.Equal(t, tt.expectedURL, modified.MDMlabDesktop.TransparencyURL)
				ac, err = svc.AppConfigObfuscated(ctx)
				require.NoError(t, err)
				require.Equal(t, tt.expectedURL, ac.MDMlabDesktop.TransparencyURL)
			}
		})
	}
}

// TestTransparencyURLDowngradeLicense tests scenarios where a transparency url value has previously
// been stored (for example, if a licensee downgraded without manually resetting the transparency url)
func TestTransparencyURLDowngradeLicense(t *testing.T) {
	ds := new(mock.Store)

	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: "free"}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	dsAppConfig := &mdmlab.AppConfig{
		OrgInfo: mdmlab.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: mdmlab.ServerSettings{
			ServerURL: "https://example.org",
		},
		MDMlabDesktop: mdmlab.MDMlabDesktopSettings{TransparencyURL: "https://example.com/transparency"},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return dsAppConfig, nil
	}

	ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
		*dsAppConfig = *conf
		return nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
		return []*mdmlab.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
		return []*mdmlab.ABMToken{}, nil
	}

	ac, err := svc.AppConfigObfuscated(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/transparency", ac.MDMlabDesktop.TransparencyURL)

	// setting transparency url fails
	raw, err := json.Marshal(mdmlab.MDMlabDesktopSettings{TransparencyURL: "https://f1337.com/transparency"})
	require.NoError(t, err)
	raw = []byte(`{"mdmlab_desktop":` + string(raw) + `}`)
	_, err = svc.ModifyAppConfig(ctx, raw, mdmlab.ApplySpecOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "missing or invalid license")

	// setting unrelated config value does not fail and resets transparency url to ""
	raw, err = json.Marshal(mdmlab.OrgInfo{OrgName: "f1337"})
	require.NoError(t, err)
	raw = []byte(`{"org_info":` + string(raw) + `}`)
	modified, err := svc.ModifyAppConfig(ctx, raw, mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	require.NotNil(t, modified)
	require.Equal(t, "", modified.MDMlabDesktop.TransparencyURL)
	ac, err = svc.AppConfigObfuscated(ctx)
	require.NoError(t, err)
	require.Equal(t, "f1337", ac.OrgInfo.OrgName)
	require.Equal(t, "", ac.MDMlabDesktop.TransparencyURL)
}

func TestMDMAppleConfig(t *testing.T) {
	ds := new(mock.Store)
	depStorage := new(nanodep_mock.Storage)

	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}

	depSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			_, _ = w.Write([]byte(`{"profile_uuid": "xyz"}`))
		}
	}))
	t.Cleanup(depSrv.Close)

	const licenseErr = "missing or invalid license"
	const notFoundErr = "not found"
	testCases := []struct {
		name          string
		licenseTier   string
		oldMDM        mdmlab.MDM
		newMDM        mdmlab.MDM
		expectedMDM   mdmlab.MDM
		expectedError string
		findTeam      bool
	}{
		{
			name:        "nochange",
			licenseTier: "free",
			expectedMDM: mdmlab.MDM{
				AppleBusinessManager: optjson.Slice[mdmlab.MDMAppleABMAssignmentInfo]{Set: true, Value: []mdmlab.MDMAppleABMAssignmentInfo{}},
				MacOSSetup: mdmlab.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*mdmlab.MacOSSetupSoftware]{Set: true, Value: []*mdmlab.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[mdmlab.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []mdmlab.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          mdmlab.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: mdmlab.WindowsSettings{
					CustomSettings: optjson.Slice[mdmlab.MDMProfileSpec]{Set: true, Value: []mdmlab.MDMProfileSpec{}},
				},
			},
		}, {
			name:          "newDefaultTeamNoLicense",
			licenseTier:   "free",
			newMDM:        mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedError: licenseErr,
		}, {
			name:          "notFoundNew",
			licenseTier:   "premium",
			newMDM:        mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedError: notFoundErr,
		}, {
			name:          "notFoundEdit",
			licenseTier:   "premium",
			oldMDM:        mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			newMDM:        mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "bar"},
			expectedError: notFoundErr,
		}, {
			name:        "foundNew",
			licenseTier: "premium",
			findTeam:    true,
			newMDM:      mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedMDM: mdmlab.MDM{
				AppleBusinessManager:         optjson.Slice[mdmlab.MDMAppleABMAssignmentInfo]{Set: true, Value: []mdmlab.MDMAppleABMAssignmentInfo{}},
				DeprecatedAppleBMDefaultTeam: "foobar",
				MacOSSetup: mdmlab.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*mdmlab.MacOSSetupSoftware]{Set: true, Value: []*mdmlab.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[mdmlab.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []mdmlab.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          mdmlab.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: mdmlab.WindowsSettings{
					CustomSettings: optjson.Slice[mdmlab.MDMProfileSpec]{Set: true, Value: []mdmlab.MDMProfileSpec{}},
				},
			},
		}, {
			name:        "foundEdit",
			licenseTier: "premium",
			findTeam:    true,
			oldMDM:      mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "bar"},
			newMDM:      mdmlab.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedMDM: mdmlab.MDM{
				AppleBusinessManager:         optjson.Slice[mdmlab.MDMAppleABMAssignmentInfo]{Set: true, Value: []mdmlab.MDMAppleABMAssignmentInfo{}},
				DeprecatedAppleBMDefaultTeam: "foobar",
				MacOSSetup: mdmlab.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*mdmlab.MacOSSetupSoftware]{Set: true, Value: []*mdmlab.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[mdmlab.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []mdmlab.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          mdmlab.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: mdmlab.WindowsSettings{
					CustomSettings: optjson.Slice[mdmlab.MDMProfileSpec]{Set: true, Value: []mdmlab.MDMProfileSpec{}},
				},
			},
		}, {
			name:          "ssoFree",
			licenseTier:   "free",
			findTeam:      true,
			newMDM:        mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{EntityID: "foo"}}},
			expectedError: licenseErr,
		}, {
			name:        "ssoFreeNoChanges",
			licenseTier: "free",
			findTeam:    true,
			newMDM:      mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{EntityID: "foo"}}},
			oldMDM:      mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{EntityID: "foo"}}},
			expectedMDM: mdmlab.MDM{
				AppleBusinessManager:  optjson.Slice[mdmlab.MDMAppleABMAssignmentInfo]{Set: true, Value: []mdmlab.MDMAppleABMAssignmentInfo{}},
				EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{EntityID: "foo"}},
				MacOSSetup: mdmlab.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*mdmlab.MacOSSetupSoftware]{Set: true, Value: []*mdmlab.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[mdmlab.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []mdmlab.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          mdmlab.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: mdmlab.WindowsSettings{
					CustomSettings: optjson.Slice[mdmlab.MDMProfileSpec]{Set: true, Value: []mdmlab.MDMProfileSpec{}},
				},
			},
		}, {
			name:        "ssoAllFields",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:    "mdmlab",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				IDPName:     "onelogin",
			}}},
			expectedMDM: mdmlab.MDM{
				AppleBusinessManager: optjson.Slice[mdmlab.MDMAppleABMAssignmentInfo]{Set: true, Value: []mdmlab.MDMAppleABMAssignmentInfo{}},
				EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{
					EntityID:    "mdmlab",
					IssuerURI:   "http://issuer.idp.com",
					MetadataURL: "http://isser.metadata.com",
					IDPName:     "onelogin",
				}},
				MacOSSetup: mdmlab.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*mdmlab.MacOSSetupSoftware]{Set: true, Value: []*mdmlab.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[mdmlab.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []mdmlab.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          mdmlab.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: mdmlab.WindowsSettings{
					CustomSettings: optjson.Slice[mdmlab.MDMProfileSpec]{Set: true, Value: []mdmlab.MDMProfileSpec{}},
				},
			},
		}, {
			name:        "ssoShortEntityID",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:    "f",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				IDPName:     "onelogin",
			}}},
			expectedError: "validation failed: entity_id must be 5 or more characters",
		}, {
			name:        "ssoMissingMetadata",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:  "mdmlab",
				IssuerURI: "http://issuer.idp.com",
				IDPName:   "onelogin",
			}}},
			expectedError: "either metadata or metadata_url must be defined",
		}, {
			name:        "ssoMultiMetadata",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:    "mdmlab",
				IssuerURI:   "http://issuer.idp.com",
				Metadata:    "not-empty",
				MetadataURL: "not-empty",
				IDPName:     "onelogin",
			}}},
			expectedError: "metadata both metadata and metadata_url are defined, only one is allowed",
		}, {
			name:        "ssoIdPName",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: mdmlab.MDM{EndUserAuthentication: mdmlab.MDMEndUserAuthentication{SSOProviderSettings: mdmlab.SSOProviderSettings{
				EntityID:  "mdmlab",
				IssuerURI: "http://issuer.idp.com",
				Metadata:  "not-empty",
			}}},
			expectedError: "idp_name required",
		}, {
			name:        "disableDiskEncryption",
			licenseTier: "premium",
			newMDM: mdmlab.MDM{
				EnableDiskEncryption: optjson.SetBool(false),
			},
			expectedMDM: mdmlab.MDM{
				AppleBusinessManager: optjson.Slice[mdmlab.MDMAppleABMAssignmentInfo]{Set: true, Value: []mdmlab.MDMAppleABMAssignmentInfo{}},
				EnableDiskEncryption: optjson.Bool{Set: true, Valid: true, Value: false},
				MacOSSetup: mdmlab.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*mdmlab.MacOSSetupSoftware]{Set: true, Value: []*mdmlab.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           mdmlab.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[mdmlab.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []mdmlab.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          mdmlab.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: mdmlab.WindowsSettings{
					CustomSettings: optjson.Slice[mdmlab.MDMProfileSpec]{Set: true, Value: []mdmlab.MDMProfileSpec{}},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: tt.licenseTier}, DEPStorage: depStorage})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

			dsAppConfig := &mdmlab.AppConfig{
				OrgInfo:        mdmlab.OrgInfo{OrgName: "Test"},
				ServerSettings: mdmlab.ServerSettings{ServerURL: "https://example.org"},
				MDM:            tt.oldMDM,
			}

			ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
				*dsAppConfig = *conf
				return nil
			}
			ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
				if tt.findTeam {
					return &mdmlab.Team{}, nil
				}
				return nil, sql.ErrNoRows
			}
			ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, enrollmentPayload mdmlab.MDMAppleEnrollmentProfilePayload) (*mdmlab.MDMAppleEnrollmentProfile, error) {
				return &mdmlab.MDMAppleEnrollmentProfile{}, nil
			}
			ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ mdmlab.MDMAppleEnrollmentType) (*mdmlab.MDMAppleEnrollmentProfile, error) {
				raw := json.RawMessage("{}")
				return &mdmlab.MDMAppleEnrollmentProfile{DEPProfile: &raw}, nil
			}
			ds.NewJobFunc = func(ctx context.Context, job *mdmlab.Job) (*mdmlab.Job, error) {
				return job, nil
			}
			ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
				return []*mdmlab.ABMToken{{ID: 1}}, nil
			}
			ds.SaveABMTokenFunc = func(ctx context.Context, token *mdmlab.ABMToken) error {
				return nil
			}
			depStorage.RetrieveConfigFunc = func(p0 context.Context, p1 string) (*nanodep_client.Config, error) {
				return &nanodep_client.Config{BaseURL: depSrv.URL}, nil
			}
			depStorage.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
				return &nanodep_client.OAuth1Tokens{}, nil
			}
			depStorage.StoreAssignerProfileFunc = func(ctx context.Context, name string, profileUUID string) error {
				return nil
			}
			ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
				return nil
			}
			ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
				return []*mdmlab.VPPTokenDB{}, nil
			}
			ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
				return []*mdmlab.ABMToken{{OrganizationName: t.Name()}}, nil
			}

			ac, err := svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.oldMDM, ac.MDM)

			raw, err := json.Marshal(tt.newMDM)
			require.NoError(t, err)
			raw = []byte(`{"mdm":` + string(raw) + `}`)
			modified, err := svc.ModifyAppConfig(ctx, raw, mdmlab.ApplySpecOptions{})
			if tt.expectedError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedMDM, modified.MDM)
			ac, err = svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.expectedMDM, ac.MDM)
		})
	}
}

func TestDiskEncryptionSetting(t *testing.T) {
	ds := new(mock.Store)

	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	t.Run("enableDiskEncryptionWithNoPrivateKey", func(t *testing.T) {
		testConfig = config.TestConfig()
		testConfig.Server.PrivateKey = ""
		svc, ctx := newTestServiceWithConfig(t, ds, testConfig, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}})
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

		dsAppConfig := &mdmlab.AppConfig{
			OrgInfo:        mdmlab.OrgInfo{OrgName: "Test"},
			ServerSettings: mdmlab.ServerSettings{ServerURL: "https://example.org"},
			MDM:            mdmlab.MDM{},
		}

		ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
			return dsAppConfig, nil
		}

		ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
			*dsAppConfig = *conf
			return nil
		}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*mdmlab.Team, error) {
			return nil, sql.ErrNoRows
		}
		ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, enrollmentPayload mdmlab.MDMAppleEnrollmentProfilePayload) (*mdmlab.MDMAppleEnrollmentProfile, error) {
			return &mdmlab.MDMAppleEnrollmentProfile{}, nil
		}
		ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ mdmlab.MDMAppleEnrollmentType) (*mdmlab.MDMAppleEnrollmentProfile, error) {
			raw := json.RawMessage("{}")
			return &mdmlab.MDMAppleEnrollmentProfile{DEPProfile: &raw}, nil
		}
		ds.NewJobFunc = func(ctx context.Context, job *mdmlab.Job) (*mdmlab.Job, error) {
			return job, nil
		}

		ac, err := svc.AppConfigObfuscated(ctx)
		require.NoError(t, err)
		require.Equal(t, dsAppConfig.MDM, ac.MDM)

		raw, err := json.Marshal(mdmlab.MDM{
			EnableDiskEncryption: optjson.SetBool(true),
		})
		require.NoError(t, err)
		raw = []byte(`{"mdm":` + string(raw) + `}`)
		_, err = svc.ModifyAppConfig(ctx, raw, mdmlab.ApplySpecOptions{})
		require.Error(t, err)
		require.ErrorContains(t, err, "Missing required private key")
	})
}

func TestModifyAppConfigSMTPSSOAgentOptions(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// SMTP and SSO are initially set.
	agentOptions := json.RawMessage(`
{
  "config": {
      "options": {
        "distributed_interval": 10
      }
  },
  "overrides": {
    "platforms": {
      "darwin": {
        "options": {
          "distributed_interval": 5
        }
      }
    }
  }
}`)
	dsAppConfig := &mdmlab.AppConfig{
		OrgInfo: mdmlab.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: mdmlab.ServerSettings{
			ServerURL: "https://example.org",
		},
		SMTPSettings: &mdmlab.SMTPSettings{
			SMTPEnabled:       true,
			SMTPConfigured:    true,
			SMTPSenderAddress: "foobar@example.com",
		},
		SSOSettings: &mdmlab.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: mdmlab.SSOProviderSettings{
				MetadataURL: "foobar.example.com/metadata",
			},
		},
		AgentOptions: &agentOptions,
	}

	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return dsAppConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
		*dsAppConfig = *conf
		return nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
		return []*mdmlab.VPPTokenDB{}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
		return []*mdmlab.ABMToken{}, nil
	}

	// Not sending smtp_settings, sso_settings or agent_settings will do nothing.
	b := []byte(`{}`)
	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	updatedAppConfig, err := svc.ModifyAppConfig(ctx, b, mdmlab.ApplySpecOptions{})
	require.NoError(t, err)

	require.True(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.True(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.True(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.True(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, agentOptions, *updatedAppConfig.AgentOptions)
	require.Equal(t, agentOptions, *dsAppConfig.AgentOptions)

	// Not sending sso_settings or agent settings will not change them, and
	// sending SMTP settings will change them.
	b = []byte(`{"smtp_settings": {"enable_smtp": false}}`)
	updatedAppConfig, err = svc.ModifyAppConfig(ctx, b, mdmlab.ApplySpecOptions{})
	require.NoError(t, err)

	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.True(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.True(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, agentOptions, *updatedAppConfig.AgentOptions)
	require.Equal(t, agentOptions, *dsAppConfig.AgentOptions)

	// Not sending smtp_settings or agent settings will not change them, and
	// sending SSO settings will change them.
	b = []byte(`{"sso_settings": {"enable_sso": false}}`)
	updatedAppConfig, err = svc.ModifyAppConfig(ctx, b, mdmlab.ApplySpecOptions{})
	require.NoError(t, err)

	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.False(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, agentOptions, *updatedAppConfig.AgentOptions)
	require.Equal(t, agentOptions, *dsAppConfig.AgentOptions)

	// Not sending smtp_settings or sso_settings will not change them, and
	// sending agent options will change them.
	newAgentOptions := json.RawMessage(`{
  "config": {
      "options": {
        "distributed_interval": 100
      }
  },
  "overrides": {
    "platforms": {
      "darwin": {
        "options": {
          "distributed_interval": 2
        }
      }
    }
  }
}`)
	b = []byte(`{"agent_options": ` + string(newAgentOptions) + `}`)
	updatedAppConfig, err = svc.ModifyAppConfig(ctx, b, mdmlab.ApplySpecOptions{})
	require.NoError(t, err)

	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.False(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, newAgentOptions, *dsAppConfig.AgentOptions)
	require.Equal(t, newAgentOptions, *dsAppConfig.AgentOptions)
}

// TestModifyEnableAnalytics tests that a premium customer cannot set ServerSettings.EnableAnalytics to be false.
// Free customers should be able to set the value to false, however.
func TestModifyEnableAnalytics(t *testing.T) {
	ds := new(mock.Store)

	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}

	testCases := []struct {
		name             string
		expectedEnabled  bool
		newEnabled       bool
		initialEnabled   bool
		licenseTier      string
		initialURL       string
		newURL           string
		expectedURL      string
		shouldFailModify bool
	}{
		{
			name:            "mdmlab free",
			expectedEnabled: false,
			initialEnabled:  true,
			newEnabled:      false,
			licenseTier:     mdmlab.TierFree,
		},
		{
			name:            "mdmlab premium",
			expectedEnabled: true,
			initialEnabled:  true,
			newEnabled:      false,
			licenseTier:     mdmlab.TierPremium,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: tt.licenseTier}})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

			dsAppConfig := &mdmlab.AppConfig{
				OrgInfo: mdmlab.OrgInfo{
					OrgName: "Test",
				},
				ServerSettings: mdmlab.ServerSettings{
					EnableAnalytics: true,
					ServerURL:       "https://localhost:8080",
				},
			}

			ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
				*dsAppConfig = *conf
				return nil
			}

			ds.SaveABMTokenFunc = func(ctx context.Context, tok *mdmlab.ABMToken) error {
				return nil
			}

			ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
				return []*mdmlab.VPPTokenDB{}, nil
			}

			ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
				return []*mdmlab.ABMToken{}, nil
			}

			ac, err := svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.initialEnabled, ac.ServerSettings.EnableAnalytics)

			raw, err := json.Marshal(mdmlab.ServerSettings{EnableAnalytics: tt.newEnabled, ServerURL: "https://localhost:8080"})
			require.NoError(t, err)
			raw = []byte(`{"server_settings":` + string(raw) + `}`)
			modified, err := svc.ModifyAppConfig(ctx, raw, mdmlab.ApplySpecOptions{})
			require.NoError(t, err)

			if modified != nil {
				require.Equal(t, tt.expectedEnabled, modified.ServerSettings.EnableAnalytics)
				ac, err = svc.AppConfigObfuscated(ctx)
				require.NoError(t, err)
				require.Equal(t, tt.expectedEnabled, ac.ServerSettings.EnableAnalytics)
			}
		})
	}
}

func TestModifyAppConfigForNDESSCEPProxy(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierFree}})
	scepURL := "https://example.com/mscep/mscep.dll"
	adminURL := "https://example.com/mscep_admin/"
	username := "user"
	password := "password"

	appConfig := &mdmlab.AppConfig{
		OrgInfo: mdmlab.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: mdmlab.ServerSettings{
			ServerURL: "https://localhost:8080",
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		if appConfig.Integrations.NDESSCEPProxy.Valid {
			appConfig.Integrations.NDESSCEPProxy.Value.Password = mdmlab.MaskedPassword
		}
		return appConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *mdmlab.AppConfig) error {
		appConfig = conf
		return nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*mdmlab.ABMToken, error) {
		return []*mdmlab.ABMToken{{ID: 1}}, nil
	}
	ds.SaveABMTokenFunc = func(ctx context.Context, token *mdmlab.ABMToken) error {
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
		return []*mdmlab.VPPTokenDB{}, nil
	}

	jsonPayloadBase := `
{
	"integrations": {
		"ndes_scep_proxy": {
			"url": "%s",
			"admin_url": "%s",
			"username": "%s",
			"password": "%s"
		}
	}
}
`
	jsonPayload := fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, password)
	admin := &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	// SCEP proxy not configured for free users
	_, err := svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	assert.ErrorContains(t, err, ErrMissingLicense.Error())
	assert.ErrorContains(t, err, "integrations.ndes_scep_proxy")

	origValidateNDESSCEPURL := validateNDESSCEPURL
	origValidateNDESSCEPAdminURL := validateNDESSCEPAdminURL
	t.Cleanup(func() {
		validateNDESSCEPURL = origValidateNDESSCEPURL
		validateNDESSCEPAdminURL = origValidateNDESSCEPAdminURL
	})
	validateNDESSCEPURLCalled := false
	validateNDESSCEPURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration, _ log.Logger) error {
		validateNDESSCEPURLCalled = true
		return nil
	}
	validateNDESSCEPAdminURLCalled := false
	validateNDESSCEPAdminURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration) error {
		validateNDESSCEPAdminURLCalled = true
		return nil
	}

	mdmlabConfig := config.TestConfig()
	svc, ctx = newTestServiceWithConfig(t, ds, mdmlabConfig, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	ds.NewActivityFunc = func(ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte,
		createdAt time.Time,
	) error {
		assert.IsType(t, mdmlab.ActivityAddedNDESSCEPProxy{}, activity)
		return nil
	}
	ac, err := svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy := func() {
		require.NotNil(t, ac.Integrations.NDESSCEPProxy)
		assert.Equal(t, scepURL, ac.Integrations.NDESSCEPProxy.Value.URL)
		assert.Equal(t, adminURL, ac.Integrations.NDESSCEPProxy.Value.AdminURL)
		assert.Equal(t, username, ac.Integrations.NDESSCEPProxy.Value.Username)
		assert.Equal(t, mdmlab.MaskedPassword, ac.Integrations.NDESSCEPProxy.Value.Password)
	}
	checkSCEPProxy()
	assert.True(t, validateNDESSCEPURLCalled)
	assert.True(t, validateNDESSCEPAdminURLCalled)
	assert.True(t, ds.SaveAppConfigFuncInvoked)
	ds.SaveAppConfigFuncInvoked = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation not done if there is no change
	appConfig = ac
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	jsonPayload = fmt.Sprintf(jsonPayloadBase, " "+scepURL, adminURL+" ", " "+username+" ", mdmlab.MaskedPassword)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	require.NoError(t, err, jsonPayload)
	checkSCEPProxy()
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation not done if there is no change, part 2
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	ac, err = svc.ModifyAppConfig(ctx, []byte(`{"integrations":{}}`), mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation done for SCEP URL. Password is blank, which is not considered a change.
	scepURL = "https://new.com/mscep/mscep.dll"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, "")
	ds.NewActivityFunc = func(ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte,
		createdAt time.Time,
	) error {
		assert.IsType(t, mdmlab.ActivityEditedNDESSCEPProxy{}, activity)
		return nil
	}
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.True(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	appConfig = ac
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation done for SCEP admin URL
	adminURL = "https://new.com/mscep_admin/"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, mdmlab.MaskedPassword)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.False(t, validateNDESSCEPURLCalled)
	assert.True(t, validateNDESSCEPAdminURLCalled)
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation fails
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	validateNDESSCEPURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration, _ log.Logger) error {
		validateNDESSCEPURLCalled = true
		return errors.New("**invalid** 1")
	}
	validateNDESSCEPAdminURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration) error {
		validateNDESSCEPAdminURLCalled = true
		return errors.New("**invalid** 2")
	}
	scepURL = "https://new2.com/mscep/mscep.dll"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, password)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	assert.ErrorContains(t, err, "**invalid**")
	assert.True(t, validateNDESSCEPURLCalled)
	assert.True(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Reset validation
	validateNDESSCEPURLCalled = false
	validateNDESSCEPURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration, _ log.Logger) error {
		validateNDESSCEPURLCalled = true
		return nil
	}
	validateNDESSCEPAdminURLCalled = false
	validateNDESSCEPAdminURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration) error {
		validateNDESSCEPAdminURLCalled = true
		return nil
	}

	// Config cleared with explicit null
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	payload := `
{
	"integrations": {
		"ndes_scep_proxy": null
	}
}
`
	// First, dry run.
	appConfig.Integrations.NDESSCEPProxy.Valid = true
	ac, err = svc.ModifyAppConfig(ctx, []byte(payload), mdmlab.ApplySpecOptions{DryRun: true})
	require.NoError(t, err)
	assert.False(t, ac.Integrations.NDESSCEPProxy.Valid)
	// Also check what was saved.
	assert.False(t, appConfig.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.HardDeleteMDMConfigAssetFuncInvoked, "DB write should not happen in dry run")
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Second, real run.
	appConfig.Integrations.NDESSCEPProxy.Valid = true
	ds.NewActivityFunc = func(ctx context.Context, user *mdmlab.User, activity mdmlab.ActivityDetails, details []byte,
		createdAt time.Time,
	) error {
		assert.IsType(t, mdmlab.ActivityDeletedNDESSCEPProxy{}, activity)
		return nil
	}
	ds.HardDeleteMDMConfigAssetFunc = func(ctx context.Context, assetName mdmlab.MDMAssetName) error {
		return nil
	}
	ac, err = svc.ModifyAppConfig(ctx, []byte(payload), mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	assert.False(t, ac.Integrations.NDESSCEPProxy.Valid)
	// Also check what was saved.
	assert.False(t, appConfig.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.True(t, ds.HardDeleteMDMConfigAssetFuncInvoked)
	ds.HardDeleteMDMConfigAssetFuncInvoked = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Deleting again should be a no-op
	appConfig.Integrations.NDESSCEPProxy.Valid = false
	ac, err = svc.ModifyAppConfig(ctx, []byte(payload), mdmlab.ApplySpecOptions{})
	require.NoError(t, err)
	assert.False(t, ac.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, appConfig.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.HardDeleteMDMConfigAssetFuncInvoked)
	ds.HardDeleteMDMConfigAssetFuncInvoked = false
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Cannot configure NDES without private key
	mdmlabConfig.Server.PrivateKey = ""
	svc, ctx = newTestServiceWithConfig(t, ds, mdmlabConfig, nil, nil, &TestServerOpts{License: &mdmlab.LicenseInfo{Tier: mdmlab.TierPremium}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	_, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), mdmlab.ApplySpecOptions{})
	assert.ErrorContains(t, err, "private key")
}
