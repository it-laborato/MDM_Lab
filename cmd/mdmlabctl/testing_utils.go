package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/datastore/cached_mysql"
	"github.com:it-laborato/MDM_Lab/server/datastore/mysql"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/service"
	"github.com:it-laborato/MDM_Lab/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli/v2"
)

type withDS struct {
	suite *suite.Suite
	ds    *mysql.Datastore
}

func (ts *withDS) SetupSuite(dbName string) {
	t := ts.suite.T()
	ts.ds = mysql.CreateNamedMySQLDS(t, dbName)
	test.AddAllHostsLabel(t, ts.ds)

	// Set up the required fields on AppConfig
	appConf, err := ts.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConf.OrgInfo.OrgName = "MDMlabTest"
	appConf.ServerSettings.ServerURL = "https://example.org"
	err = ts.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)
}

func (ts *withDS) TearDownSuite() {
	_ = ts.ds.Close()
}

type withServer struct {
	withDS

	server *httptest.Server
	users  map[string]mdmlab.User
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (ts *withServer) getTestToken(email string, password string) string {
	params := loginRequest{
		Email:    email,
		Password: password,
	}
	j, err := json.Marshal(&params)
	require.NoError(ts.suite.T(), err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(ts.server.URL+"/api/latest/mdmlab/login", "application/json", requestBody)
	require.NoError(ts.suite.T(), err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(ts.suite.T(), http.StatusOK, resp.StatusCode)

	jsn := struct {
		User  *mdmlab.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(ts.suite.T(), err)
	require.Len(ts.suite.T(), jsn.Err, 0)

	return jsn.Token
}

// runServerWithMockedDS runs the mdmlab server with several mocked DS methods.
//
// NOTE: Assumes the current session is always from the admin user (see ds.SessionByKeyFunc below).
func runServerWithMockedDS(t *testing.T, opts ...*service.TestServerOpts) (*httptest.Server, *mock.Store) {
	ds := new(mock.Store)
	var users []*mdmlab.User
	var admin *mdmlab.User
	ds.NewUserFunc = func(ctx context.Context, user *mdmlab.User) (*mdmlab.User, error) {
		if user.GlobalRole != nil && *user.GlobalRole == mdmlab.RoleAdmin {
			admin = user
		}
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*mdmlab.Session, error) {
		return &mdmlab.Session{
			CreateTimestamp: mdmlab.CreateTimestamp{CreatedAt: time.Now()},
			ID:              1,
			AccessedAt:      time.Now(),
			UserID:          admin.ID,
			Key:             key,
		}, nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *mdmlab.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*mdmlab.User, error) {
		return admin, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt mdmlab.UserListOptions) ([]*mdmlab.User, error) {
		return users, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*mdmlab.AppConfig, error) {
		return &mdmlab.AppConfig{}, nil
	}
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes()
	require.NoError(t, err)
	certPEM, keyPEM, tokenBytes, err := mysql.GenerateTestABMAssets(t)
	require.NoError(t, err)
	ds.GetAllMDMConfigAssetsHashesFunc = func(ctx context.Context, assetNames []mdmlab.MDMAssetName) (map[mdmlab.MDMAssetName]string, error) {
		return map[mdmlab.MDMAssetName]string{
			mdmlab.MDMAssetABMCert:            "abmcert",
			mdmlab.MDMAssetABMKey:             "abmkey",
			mdmlab.MDMAssetABMTokenDeprecated: "abmtoken",
			mdmlab.MDMAssetAPNSCert:           "apnscert",
			mdmlab.MDMAssetAPNSKey:            "apnskey",
			mdmlab.MDMAssetCACert:             "scepcert",
			mdmlab.MDMAssetCAKey:              "scepkey",
		}, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []mdmlab.MDMAssetName, _ sqlx.QueryerContext) (map[mdmlab.MDMAssetName]mdmlab.MDMConfigAsset, error) {
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

	ds.ApplyYaraRulesFunc = func(context.Context, []mdmlab.YaraRule) error {
		return nil
	}
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}

	var cachedDS mdmlab.Datastore
	if len(opts) > 0 && opts[0].NoCacheDatastore {
		cachedDS = ds
	} else {
		cachedDS = cached_mysql.New(ds)
	}
	_, server := service.RunServerForTestsWithDS(t, cachedDS, opts...)
	os.Setenv("FLEET_SERVER_ADDRESS", server.URL)

	return server, ds
}

func runAppForTest(t *testing.T, args []string) string {
	w, err := runAppNoChecks(args)
	require.NoError(t, err)
	return w.String()
}

func runAppCheckErr(t *testing.T, args []string, errorMsg string) string {
	w, err := runAppNoChecks(args)
	require.Error(t, err)
	require.Equal(t, errorMsg, err.Error())
	return w.String()
}

func runAppNoChecks(args []string) (*bytes.Buffer, error) {
	// first arg must be the binary name. Allow tests to omit it.
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, os.Stderr, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func runWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, errWriter, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}

func serveMDMBootstrapPackage(t *testing.T, pkgPath, pkgName string) (*httptest.Server, int) {
	pkgBytes, err := os.ReadFile(pkgPath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(pkgBytes)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, pkgName))
		if n, err := w.Write(pkgBytes); err != nil {
			require.NoError(t, err)
			require.Equal(t, len(pkgBytes), n)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, len(pkgBytes)
}
