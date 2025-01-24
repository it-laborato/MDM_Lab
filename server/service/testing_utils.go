package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mixer/clock"
	eeservice "github.com/it-laborato/MDM_Lab/ee/server/service"
	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/contexts/license"
	"github.com/it-laborato/MDM_Lab/server/datastore/cached_mysql"
	"github.com/it-laborato/MDM_Lab/server/datastore/filesystem"
	"github.com/it-laborato/MDM_Lab/server/errorstore"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/logging"
	"github.com/it-laborato/MDM_Lab/server/mail"
	apple_mdm "github.com/it-laborato/MDM_Lab/server/mdm/apple"
	microsoft_mdm "github.com/it-laborato/MDM_Lab/server/mdm/microsoft"
	nanodep_storage "github.com/it-laborato/MDM_Lab/server/mdm/nanodep/storage"
	"github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/mdm"
	"github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/push"
	nanomdm_push "github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/push"
	scep_depot "github.com/it-laborato/MDM_Lab/server/mdm/scep/depot"
	nanodep_mock "github.com/it-laborato/MDM_Lab/server/mock/nanodep"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/service/async"
	"github.com/it-laborato/MDM_Lab/server/service/mock"
	"github.com/it-laborato/MDM_Lab/server/service/redis_key_value"
	"github.com/it-laborato/MDM_Lab/server/service/redis_lock"
	"github.com/it-laborato/MDM_Lab/server/sso"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2"
	"github.com/throttled/throttled/v2/store/memstore"
)

func newTestService(t *testing.T, ds mdmlab.Datastore, rs mdmlab.QueryResultStore, lq mdmlab.LiveQueryStore, opts ...*TestServerOpts) (mdmlab.Service, context.Context) {
	return newTestServiceWithConfig(t, ds, config.TestConfig(), rs, lq, opts...)
}

func newTestServiceWithConfig(t *testing.T, ds mdmlab.Datastore, mdmlabConfig config.MDMlabConfig, rs mdmlab.QueryResultStore, lq mdmlab.LiveQueryStore, opts ...*TestServerOpts) (mdmlab.Service, context.Context) {
	lic := &mdmlab.LicenseInfo{Tier: mdmlab.TierFree}
	writer, err := logging.NewFilesystemLogWriter(mdmlabConfig.Filesystem.StatusLogFile, kitlog.NewNopLogger(), mdmlabConfig.Filesystem.EnableLogRotation, mdmlabConfig.Filesystem.EnableLogCompression, 500, 28, 3)
	require.NoError(t, err)

	osqlogger := &OsqueryLogger{Status: writer, Result: writer}
	logger := kitlog.NewNopLogger()

	var (
		failingPolicySet  mdmlab.FailingPolicySet        = NewMemFailingPolicySet()
		enrollHostLimiter mdmlab.EnrollHostLimiter       = nopEnrollHostLimiter{}
		depStorage        nanodep_storage.AllDEPStorage = &nanodep_mock.Storage{}
		mailer            mdmlab.MailService             = &mockMailService{SendEmailFn: func(e mdmlab.Email) error { return nil }}
		c                 clock.Clock                   = clock.C

		is                    mdmlab.InstallerStore
		mdmStorage            mdmlab.MDMAppleStore
		mdmPusher             nanomdm_push.Pusher
		ssoStore              sso.SessionStore
		profMatcher           mdmlab.ProfileMatcher
		softwareInstallStore  mdmlab.SoftwareInstallerStore
		bootstrapPackageStore mdmlab.MDMBootstrapPackageStore
		distributedLock       mdmlab.Lock
		keyValueStore         mdmlab.KeyValueStore
	)
	if len(opts) > 0 {
		if opts[0].Clock != nil {
			c = opts[0].Clock
		}
	}

	if len(opts) > 0 && opts[0].KeyValueStore != nil {
		keyValueStore = opts[0].KeyValueStore
	}

	task := async.NewTask(ds, nil, c, config.OsqueryConfig{})
	if len(opts) > 0 {
		if opts[0].Task != nil {
			task = opts[0].Task
		} else {
			opts[0].Task = task
		}
	}

	if len(opts) > 0 {
		if opts[0].Logger != nil {
			logger = opts[0].Logger
		}
		if opts[0].License != nil {
			lic = opts[0].License
		}
		if opts[0].Pool != nil {
			ssoStore = sso.NewSessionStore(opts[0].Pool)
			profMatcher = apple_mdm.NewProfileMatcher(opts[0].Pool)
			distributedLock = redis_lock.NewLock(opts[0].Pool)
			keyValueStore = redis_key_value.New(opts[0].Pool)
		}
		if opts[0].ProfileMatcher != nil {
			profMatcher = opts[0].ProfileMatcher
		}
		if opts[0].FailingPolicySet != nil {
			failingPolicySet = opts[0].FailingPolicySet
		}
		if opts[0].EnrollHostLimiter != nil {
			enrollHostLimiter = opts[0].EnrollHostLimiter
		}
		if opts[0].UseMailService {
			mailer, err = mail.NewService(config.TestConfig())
			require.NoError(t, err)
		}
		if opts[0].SoftwareInstallStore != nil {
			softwareInstallStore = opts[0].SoftwareInstallStore
		}
		if opts[0].BootstrapPackageStore != nil {
			bootstrapPackageStore = opts[0].BootstrapPackageStore
		}

		// allow to explicitly set installer store to nil
		is = opts[0].Is
		// allow to explicitly set MDM storage to nil
		mdmStorage = opts[0].MDMStorage
		if opts[0].DEPStorage != nil {
			depStorage = opts[0].DEPStorage
		}
		// allow to explicitly set mdm pusher to nil
		mdmPusher = opts[0].MDMPusher
	}

	ctx := license.NewContext(context.Background(), lic)

	cronSchedulesService := mdmlab.NewCronSchedules()

	var eh *errorstore.Handler
	if len(opts) > 0 {
		if opts[0].Pool != nil {
			eh = errorstore.NewHandler(ctx, opts[0].Pool, logger, time.Minute*10)
			ctx = ctxerr.NewContext(ctx, eh)
		}
		if opts[0].StartCronSchedules != nil {
			for _, fn := range opts[0].StartCronSchedules {
				err = cronSchedulesService.StartCronSchedule(fn(ctx, ds))
				require.NoError(t, err)
			}
		}
	}

	var wstepManager microsoft_mdm.CertManager
	if mdmlabConfig.MDM.WindowsWSTEPIdentityCert != "" && mdmlabConfig.MDM.WindowsWSTEPIdentityKey != "" {
		rawCert, err := os.ReadFile(mdmlabConfig.MDM.WindowsWSTEPIdentityCert)
		require.NoError(t, err)
		rawKey, err := os.ReadFile(mdmlabConfig.MDM.WindowsWSTEPIdentityKey)
		require.NoError(t, err)

		wstepManager, err = microsoft_mdm.NewCertManager(ds, rawCert, rawKey)
		require.NoError(t, err)
	}

	svc, err := NewService(
		ctx,
		ds,
		task,
		rs,
		logger,
		osqlogger,
		mdmlabConfig,
		mailer,
		c,
		ssoStore,
		lq,
		ds,
		is,
		failingPolicySet,
		&mdmlab.NoOpGeoIP{},
		enrollHostLimiter,
		depStorage,
		mdmStorage,
		mdmPusher,
		cronSchedulesService,
		wstepManager,
	)
	if err != nil {
		panic(err)
	}
	if lic.IsPremium() {
		if softwareInstallStore == nil {
			// default to file-based
			dir := t.TempDir()
			store, err := filesystem.NewSoftwareInstallerStore(dir)
			if err != nil {
				panic(err)
			}
			softwareInstallStore = store
		}
		svc, err = eeservice.NewService(
			svc,
			ds,
			logger,
			mdmlabConfig,
			mailer,
			c,
			depStorage,
			apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPusher),
			ssoStore,
			profMatcher,
			softwareInstallStore,
			bootstrapPackageStore,
			distributedLock,
			keyValueStore,
		)
		if err != nil {
			panic(err)
		}
	}
	return svc, ctx
}

func newTestServiceWithClock(t *testing.T, ds mdmlab.Datastore, rs mdmlab.QueryResultStore, lq mdmlab.LiveQueryStore, c clock.Clock) (mdmlab.Service, context.Context) {
	testConfig := config.TestConfig()
	return newTestServiceWithConfig(t, ds, testConfig, rs, lq, &TestServerOpts{
		Clock: c,
	})
}

func createTestUsers(t *testing.T, ds mdmlab.Datastore) map[string]mdmlab.User {
	users := make(map[string]mdmlab.User)
	// Map iteration is random so we sort and iterate using the testUsers keys.
	var keys []string
	for key := range testUsers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	userID := uint(1)
	for _, key := range keys {
		u := testUsers[key]
		role := mdmlab.RoleObserver
		if strings.Contains(u.Email, "admin") {
			role = mdmlab.RoleAdmin
		}
		user := &mdmlab.User{
			ID:         userID, // We need to set this in case ds is a mocked Datastore.
			Name:       "Test Name " + u.Email,
			Email:      u.Email,
			GlobalRole: &role,
		}
		err := user.SetPassword(u.PlaintextPassword, 10, 10)
		require.Nil(t, err)
		user, err = ds.NewUser(context.Background(), user)
		require.Nil(t, err)
		users[user.Email] = *user
		userID++
	}
	return users
}

var testUsers = map[string]struct {
	Email             string
	PlaintextPassword string
	GlobalRole        *string
}{
	"admin1": {
		PlaintextPassword: test.GoodPassword,
		Email:             "admin1@example.com",
		GlobalRole:        ptr.String(mdmlab.RoleAdmin),
	},
	"user1": {
		PlaintextPassword: test.GoodPassword,
		Email:             "user1@example.com",
		GlobalRole:        ptr.String(mdmlab.RoleMaintainer),
	},
	"user2": {
		PlaintextPassword: test.GoodPassword,
		Email:             "user2@example.com",
		GlobalRole:        ptr.String(mdmlab.RoleObserver),
	},
}

func createEnrollSecrets(t *testing.T, count int) []*mdmlab.EnrollSecret {
	secrets := make([]*mdmlab.EnrollSecret, count)
	for i := 0; i < count; i++ {
		secrets[i] = &mdmlab.EnrollSecret{Secret: fmt.Sprintf("testSecret%d", i)}
	}
	return secrets
}

type mockMailService struct {
	SendEmailFn func(e mdmlab.Email) error
	Invoked     bool
}

func (svc *mockMailService) SendEmail(e mdmlab.Email) error {
	svc.Invoked = true
	return svc.SendEmailFn(e)
}

func (svc *mockMailService) CanSendEmail(smtpSettings mdmlab.SMTPSettings) bool {
	return smtpSettings.SMTPConfigured
}

type TestNewScheduleFunc func(ctx context.Context, ds mdmlab.Datastore) mdmlab.NewCronScheduleFunc

type TestServerOpts struct {
	Logger                kitlog.Logger
	License               *mdmlab.LicenseInfo
	SkipCreateTestUsers   bool
	Rs                    mdmlab.QueryResultStore
	Lq                    mdmlab.LiveQueryStore
	Pool                  mdmlab.RedisPool
	FailingPolicySet      mdmlab.FailingPolicySet
	Clock                 clock.Clock
	Task                  *async.Task
	EnrollHostLimiter     mdmlab.EnrollHostLimiter
	Is                    mdmlab.InstallerStore
	MDMlabConfig           *config.MDMlabConfig
	MDMStorage            mdmlab.MDMAppleStore
	DEPStorage            nanodep_storage.AllDEPStorage
	SCEPStorage           scep_depot.Depot
	MDMPusher             nanomdm_push.Pusher
	HTTPServerConfig      *http.Server
	StartCronSchedules    []TestNewScheduleFunc
	UseMailService        bool
	APNSTopic             string
	ProfileMatcher        mdmlab.ProfileMatcher
	EnableCachedDS        bool
	NoCacheDatastore      bool
	SoftwareInstallStore  mdmlab.SoftwareInstallerStore
	BootstrapPackageStore mdmlab.MDMBootstrapPackageStore
	KeyValueStore         mdmlab.KeyValueStore
	EnableSCEPProxy       bool
	WithDEPWebview        bool
}

func RunServerForTestsWithDS(t *testing.T, ds mdmlab.Datastore, opts ...*TestServerOpts) (map[string]mdmlab.User, *httptest.Server) {
	if len(opts) > 0 && opts[0].EnableCachedDS {
		ds = cached_mysql.New(ds)
	}
	var rs mdmlab.QueryResultStore
	if len(opts) > 0 && opts[0].Rs != nil {
		rs = opts[0].Rs
	}
	var lq mdmlab.LiveQueryStore
	if len(opts) > 0 && opts[0].Lq != nil {
		lq = opts[0].Lq
	}
	cfg := config.TestConfig()
	if len(opts) > 0 && opts[0].MDMlabConfig != nil {
		cfg = *opts[0].MDMlabConfig
	}
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, rs, lq, opts...)
	users := map[string]mdmlab.User{}
	if len(opts) == 0 || (len(opts) > 0 && !opts[0].SkipCreateTestUsers) {
		users = createTestUsers(t, ds)
	}
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	if len(opts) > 0 && opts[0].Logger != nil {
		logger = opts[0].Logger
	}
	var mdmPusher nanomdm_push.Pusher
	if len(opts) > 0 && opts[0].MDMPusher != nil {
		mdmPusher = opts[0].MDMPusher
	}
	limitStore, _ := memstore.New(0)
	rootMux := http.NewServeMux()

	if len(opts) > 0 {
		mdmStorage := opts[0].MDMStorage
		scepStorage := opts[0].SCEPStorage
		commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPusher)
		if mdmStorage != nil && scepStorage != nil {
			err := RegisterAppleMDMProtocolServices(
				rootMux,
				cfg.MDM,
				mdmStorage,
				scepStorage,
				logger,
				NewMDMAppleCheckinAndCommandService(ds, commander, logger),
				&MDMAppleDDMService{
					ds:     ds,
					logger: logger,
				},
				commander,
			)
			require.NoError(t, err)
		}
		if opts[0].EnableSCEPProxy {
			err := RegisterSCEPProxy(
				rootMux,
				ds,
				logger,
			)
			require.NoError(t, err)
			origValidateNDESSCEPURL := validateNDESSCEPURL
			origValidateNDESSCEPAdminURL := validateNDESSCEPAdminURL
			t.Cleanup(func() {
				validateNDESSCEPURL = origValidateNDESSCEPURL
				validateNDESSCEPAdminURL = origValidateNDESSCEPAdminURL
			})
			validateNDESSCEPURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration, _ log.Logger) error {
				return nil
			}
			validateNDESSCEPAdminURL = func(_ context.Context, _ mdmlab.NDESSCEPProxyIntegration) error {
				return nil
			}
		}
	}

	if len(opts) > 0 && opts[0].WithDEPWebview {
		frontendHandler := WithMDMEnrollmentMiddleware(svc, logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// do nothing and return 200
			w.WriteHeader(http.StatusOK)
		}))
		rootMux.Handle("/", frontendHandler)
	}

	apiHandler := MakeHandler(svc, cfg, logger, limitStore, WithLoginRateLimit(throttled.PerMin(1000)))
	rootMux.Handle("/api/", apiHandler)
	var errHandler *errorstore.Handler
	ctxErrHandler := ctxerr.FromContext(ctx)
	if ctxErrHandler != nil {
		errHandler = ctxErrHandler.(*errorstore.Handler)
	}
	debugHandler := MakeDebugHandler(svc, cfg, logger, errHandler, ds)
	rootMux.Handle("/debug/", debugHandler)

	server := httptest.NewUnstartedServer(rootMux)
	server.Config = cfg.Server.DefaultHTTPServer(ctx, rootMux)
	// WriteTimeout is set for security purposes.
	// If we don't set it, (bugy or malignant) clients making long running
	// requests could DDOS MDMlab.
	require.NotZero(t, server.Config.WriteTimeout)
	if len(opts) > 0 && opts[0].HTTPServerConfig != nil {
		server.Config = opts[0].HTTPServerConfig
		// make sure we use the application handler we just created
		server.Config.Handler = rootMux
	}
	server.Start()
	t.Cleanup(func() {
		server.Close()
	})
	return users, server
}

func testSESPluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Email = config.EmailConfig{EmailBackend: "ses"}
	c.SES = config.SESConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		SourceArn:        "qux",
	}
	return c
}

func testKinesisPluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Osquery.ResultLogPlugin = "kinesis"
	c.Osquery.StatusLogPlugin = "kinesis"
	c.Activity.AuditLogPlugin = "kinesis"
	c.Kinesis = config.KinesisConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		StatusStream:     "test-status-stream",
		ResultStream:     "test-result-stream",
		AuditStream:      "test-audit-stream",
	}
	return c
}

func testFirehosePluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Osquery.ResultLogPlugin = "firehose"
	c.Osquery.StatusLogPlugin = "firehose"
	c.Activity.AuditLogPlugin = "firehose"
	c.Firehose = config.FirehoseConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		StatusStream:     "test-status-firehose",
		ResultStream:     "test-result-firehose",
		AuditStream:      "test-audit-firehose",
	}
	return c
}

func testLambdaPluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Osquery.ResultLogPlugin = "lambda"
	c.Osquery.StatusLogPlugin = "lambda"
	c.Activity.AuditLogPlugin = "lambda"
	c.Lambda = config.LambdaConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		ResultFunction:   "result-func",
		StatusFunction:   "status-func",
		AuditFunction:    "audit-func",
	}
	return c
}

func testPubSubPluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Osquery.ResultLogPlugin = "pubsub"
	c.Osquery.StatusLogPlugin = "pubsub"
	c.Activity.AuditLogPlugin = "pubsub"
	c.PubSub = config.PubSubConfig{
		Project:       "test",
		StatusTopic:   "status-topic",
		ResultTopic:   "result-topic",
		AuditTopic:    "audit-topic",
		AddAttributes: false,
	}
	return c
}

func testStdoutPluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Osquery.ResultLogPlugin = "stdout"
	c.Osquery.StatusLogPlugin = "stdout"
	c.Activity.AuditLogPlugin = "stdout"
	return c
}

func testUnrecognizedPluginConfig() config.MDMlabConfig {
	c := config.TestConfig()
	c.Osquery = config.OsqueryConfig{
		ResultLogPlugin: "bar",
		StatusLogPlugin: "bar",
	}
	c.Activity.AuditLogPlugin = "bar"
	return c
}

func assertBodyContains(t *testing.T, resp *http.Response, expected string) {
	bodyBytes, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, expected)
}

func getJSON(r *http.Response, target interface{}) error {
	return json.NewDecoder(r.Body).Decode(target)
}

func assertErrorCodeAndMessage(t *testing.T, resp *http.Response, code int, message string) {
	err := &mdmlab.Error{}
	require.Nil(t, getJSON(resp, err))
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
}

type memFailingPolicySet struct {
	mMu sync.RWMutex
	m   map[uint][]mdmlab.PolicySetHost
}

var _ mdmlab.FailingPolicySet = (*memFailingPolicySet)(nil)

func NewMemFailingPolicySet() *memFailingPolicySet {
	return &memFailingPolicySet{
		m: make(map[uint][]mdmlab.PolicySetHost),
	}
}

// AddFailingPoliciesForHost adds the given host to the policy sets.
func (m *memFailingPolicySet) AddHost(policyID uint, host mdmlab.PolicySetHost) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	m.m[policyID] = append(m.m[policyID], host)
	return nil
}

// ListHosts returns the list of hosts present in the policy set.
func (m *memFailingPolicySet) ListHosts(policyID uint) ([]mdmlab.PolicySetHost, error) {
	m.mMu.RLock()
	defer m.mMu.RUnlock()

	hosts := make([]mdmlab.PolicySetHost, len(m.m[policyID]))
	copy(hosts, m.m[policyID])
	return hosts, nil
}

// RemoveHosts removes the hosts from the policy set.
func (m *memFailingPolicySet) RemoveHosts(policyID uint, hosts []mdmlab.PolicySetHost) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	if _, ok := m.m[policyID]; !ok {
		return nil
	}
	hostsSet := make(map[uint]struct{})
	for _, host := range hosts {
		hostsSet[host.ID] = struct{}{}
	}
	n := 0
	for _, host := range m.m[policyID] {
		if _, ok := hostsSet[host.ID]; !ok {
			m.m[policyID][n] = host
			n++
		}
	}
	m.m[policyID] = m.m[policyID][:n]
	return nil
}

// RemoveSet removes a policy set.
func (m *memFailingPolicySet) RemoveSet(policyID uint) error {
	m.mMu.Lock()
	defer m.mMu.Unlock()

	delete(m.m, policyID)
	return nil
}

// ListSets lists all the policy sets.
func (m *memFailingPolicySet) ListSets() ([]uint, error) {
	m.mMu.RLock()
	defer m.mMu.RUnlock()

	var policyIDs []uint
	for policyID := range m.m {
		policyIDs = append(policyIDs, policyID)
	}
	return policyIDs, nil
}

type nopEnrollHostLimiter struct{}

func (nopEnrollHostLimiter) CanEnrollNewHost(ctx context.Context) (bool, error) {
	return true, nil
}

func (nopEnrollHostLimiter) SyncEnrolledHostIDs(ctx context.Context) error {
	return nil
}

func newMockAPNSPushProviderFactory() (*mock.APNSPushProviderFactory, *mock.APNSPushProvider) {
	provider := &mock.APNSPushProvider{}
	provider.PushFunc = mockSuccessfulPush
	factory := &mock.APNSPushProviderFactory{}
	factory.NewPushProviderFunc = func(*tls.Certificate) (push.PushProvider, error) {
		return provider, nil
	}

	return factory, provider
}

func mockSuccessfulPush(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
	res := make(map[string]*push.Response, len(pushes))
	for _, p := range pushes {
		res[p.Token.String()] = &push.Response{
			Id:  uuid.New().String(),
			Err: nil,
		}
	}
	return res, nil
}

func mdmConfigurationRequiredEndpoints() []struct {
	method, path        string
	deviceAuthenticated bool
	premiumOnly         bool
} {
	return []struct {
		method, path        string
		deviceAuthenticated bool
		premiumOnly         bool
	}{
		{"POST", "/api/latest/mdmlab/mdm/apple/enqueue", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/commandresults", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/installers/1", false, false},
		{"DELETE", "/api/latest/mdmlab/mdm/apple/installers/1", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/installers", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/devices", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/profiles", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/profiles/1", false, false},
		{"DELETE", "/api/latest/mdmlab/mdm/apple/profiles/1", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/profiles/summary", false, false},
		{"PATCH", "/api/latest/mdmlab/mdm/hosts/1/unenroll", false, false},
		{"DELETE", "/api/latest/mdmlab/hosts/1/mdm", false, false},
		{"GET", "/api/latest/mdmlab/mdm/hosts/1/profiles", false, true},
		{"GET", "/api/latest/mdmlab/hosts/1/configuration_profiles", false, true},
		{"POST", "/api/latest/mdmlab/mdm/hosts/1/lock", false, false},
		{"POST", "/api/latest/mdmlab/mdm/hosts/1/wipe", false, false},
		{"PATCH", "/api/latest/mdmlab/mdm/apple/settings", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple", false, false},
		{"GET", "/api/latest/mdmlab/apns", false, false},
		{"GET", apple_mdm.EnrollPath + "?token=test", false, false},
		{"GET", apple_mdm.InstallerPath + "?token=test", false, false},
		{"GET", "/api/latest/mdmlab/mdm/setup/eula/0982A979-B1C9-4BDF-B584-5A37D32A1172", false, true},
		{"GET", "/api/latest/mdmlab/setup_experience/eula/0982A979-B1C9-4BDF-B584-5A37D32A1172", false, true},
		{"DELETE", "/api/latest/mdmlab/mdm/setup/eula/token", false, true},
		{"DELETE", "/api/latest/mdmlab/setup_experience/eula/token", false, true},
		{"GET", "/api/latest/mdmlab/mdm/setup/eula/metadata", false, true},
		{"GET", "/api/latest/mdmlab/setup_experience/eula/metadata", false, true},
		{"GET", "/api/latest/mdmlab/mdm/apple/setup/eula/0982A979-B1C9-4BDF-B584-5A37D32A1172", false, false},
		{"DELETE", "/api/latest/mdmlab/mdm/apple/setup/eula/token", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/setup/eula/metadata", false, false},
		{"GET", "/api/latest/mdmlab/mdm/apple/enrollment_profile", false, false},
		{"GET", "/api/latest/mdmlab/enrollment_profiles/automatic", false, false},
		{"POST", "/api/latest/mdmlab/mdm/apple/enrollment_profile", false, false},
		{"POST", "/api/latest/mdmlab/enrollment_profiles/automatic", false, false},
		{"DELETE", "/api/latest/mdmlab/mdm/apple/enrollment_profile", false, false},
		{"DELETE", "/api/latest/mdmlab/enrollment_profiles/automatic", false, false},
		{"POST", "/api/latest/mdmlab/device/%s/migrate_mdm", true, true},
		{"POST", "/api/latest/mdmlab/mdm/apple/profiles/preassign", false, true},
		{"POST", "/api/latest/mdmlab/mdm/apple/profiles/match", false, true},
		{"POST", "/api/latest/mdmlab/mdm/commands/run", false, false},
		{"POST", "/api/latest/mdmlab/commands/run", false, false},
		{"GET", "/api/latest/mdmlab/mdm/commandresults", false, false},
		{"GET", "/api/latest/mdmlab/commands/results", false, false},
		{"GET", "/api/latest/mdmlab/mdm/commands", false, false},
		{"GET", "/api/latest/mdmlab/commands", false, false},
		{"POST", "/api/mdmlab/orbit/disk_encryption_key", false, false},
		{"GET", "/api/latest/mdmlab/mdm/profiles/1", false, false},
		{"GET", "/api/latest/mdmlab/configuration_profiles/1", false, false},
		{"DELETE", "/api/latest/mdmlab/mdm/profiles/1", false, false},
		{"DELETE", "/api/latest/mdmlab/configuration_profiles/1", false, false},
		// TODO: those endpoints accept multipart/form data that gets
		// parsed before the MDM check, we need to refactor this
		// function to return more information to the caller, or find a
		// better way to test these endpoints.
		// {"POST", "/api/latest/mdmlab/mdm/profiles", false, false},
		// {"POST", "/api/latest/mdmlab/configuration_profiles", false, false},
		// {"POST", "/api/latest/mdmlab/mdm/setup/eula"},
		// {"POST", "/api/latest/mdmlab/setup_experience/eula"},
		// {"POST", "/api/latest/mdmlab/mdm/bootstrap", false, true},
		// {"POST", "/api/latest/mdmlab/bootstrap", false, true},
		{"GET", "/api/latest/mdmlab/mdm/profiles", false, false},
		{"GET", "/api/latest/mdmlab/configuration_profiles", false, false},
		{"GET", "/api/latest/mdmlab/mdm/manual_enrollment_profile", false, true},
		{"GET", "/api/latest/mdmlab/enrollment_profiles/manual", false, true},
		{"GET", "/api/latest/mdmlab/mdm/bootstrap/1/metadata", false, true},
		{"GET", "/api/latest/mdmlab/bootstrap/1/metadata", false, true},
		{"DELETE", "/api/latest/mdmlab/mdm/bootstrap/1", false, true},
		{"DELETE", "/api/latest/mdmlab/bootstrap/1", false, true},
		{"GET", "/api/latest/mdmlab/mdm/bootstrap?token=1", false, true},
		{"GET", "/api/latest/mdmlab/bootstrap?token=1", false, true},
		{"GET", "/api/latest/mdmlab/mdm/bootstrap/summary", false, true},
		{"GET", "/api/latest/mdmlab/mdm/apple/bootstrap/summary", false, true},
		{"GET", "/api/latest/mdmlab/bootstrap/summary", false, true},
		{"PATCH", "/api/latest/mdmlab/mdm/apple/setup", false, true},
		{"PATCH", "/api/latest/mdmlab/setup_experience", false, true},
		{"POST", "/api/mdmlab/orbit/setup_experience/status", false, true},
	}
}

// getURLSchemas returns a list of all valid URI schemas
func getURISchemas() []string {
	return []string{
		"aaa",
		"aaas",
		"about",
		"acap",
		"acct",
		"acd",
		"acr",
		"adiumxtra",
		"adt",
		"afp",
		"afs",
		"aim",
		"amss",
		"android",
		"appdata",
		"apt",
		"ar",
		"ark",
		"at",
		"attachment",
		"aw",
		"barion",
		"bb",
		"beshare",
		"bitcoin",
		"bitcoincash",
		"blob",
		"bolo",
		"browserext",
		"cabal",
		"calculator",
		"callto",
		"cap",
		"cast",
		"casts",
		"chrome",
		"chrome-extension",
		"cid",
		"coap",
		"coap+tcp",
		"coap+ws",
		"coaps",
		"coaps+tcp",
		"coaps+ws",
		"com-eventbrite-attendee",
		"content",
		"content-type",
		"crid",
		"cstr",
		"cvs",
		"dab",
		"dat",
		"data",
		"dav",
		"dhttp",
		"diaspora",
		"dict",
		"did",
		"dis",
		"dlna-playcontainer",
		"dlna-playsingle",
		"dns",
		"dntp",
		"doi",
		"dpp",
		"drm",
		"drop",
		"dtmi",
		"dtn",
		"dvb",
		"dvx",
		"dweb",
		"ed2k",
		"eid",
		"elsi",
		"embedded",
		"ens",
		"ethereum",
		"example",
		"facetime",
		"fax",
		"feed",
		"feedready",
		"fido",
		"file",
		"filesystem",
		"finger",
		"first-run-pen-experience",
		"fish",
		"fm",
		"ftp",
		"fuchsia-pkg",
		"geo",
		"gg",
		"git",
		"gitoid",
		"gizmoproject",
		"go",
		"gopher",
		"graph",
		"grd",
		"gtalk",
		"h323",
		"ham",
		"hcap",
		"hcp",
		"http",
		"https",
		"hxxp",
		"hxxps",
		"hydrazone",
		"hyper",
		"iax",
		"icap",
		"icon",
		"im",
		"imap",
		"info",
		"iotdisco",
		"ipfs",
		"ipn",
		"ipns",
		"ipp",
		"ipps",
		"irc",
		"irc6",
		"ircs",
		"iris",
		"iris.beep",
		"iris.lwz",
		"iris.xpc",
		"iris.xpcs",
		"isostore",
		"itms",
		"jabber",
		"jar",
		"jms",
		"keyparc",
		"lastfm",
		"lbry",
		"ldap",
		"ldaps",
		"leaptofrogans",
		"lorawan",
		"lpa",
		"lvlt",
		"magnet",
		"mailserver",
		"mailto",
		"maps",
		"market",
		"matrix",
		"message",
		"microsoft.windows.camera",
		"microsoft.windows.camera.multipicker",
		"microsoft.windows.camera.picker",
		"mid",
		"mms",
		"modem",
		"mongodb",
		"moz",
		"ms-access",
		"ms-appinstaller",
		"ms-browser-extension",
		"ms-calculator",
		"ms-drive-to",
		"ms-enrollment",
		"ms-excel",
		"ms-eyecontrolspeech",
		"ms-gamebarservices",
		"ms-gamingoverlay",
		"ms-getoffice",
		"ms-help",
		"ms-infopath",
		"ms-inputapp",
		"ms-launchremotedesktop",
		"ms-lockscreencomponent-config",
		"ms-media-stream-id",
		"ms-meetnow",
		"ms-mixedrealitycapture",
		"ms-mobileplans",
		"ms-newsandinterests",
		"ms-officeapp",
		"ms-people",
		"ms-project",
		"ms-powerpoint",
		"ms-publisher",
		"ms-remotedesktop",
		"ms-remotedesktop-launch",
		"ms-restoretabcompanion",
		"ms-screenclip",
		"ms-screensketch",
		"ms-search",
		"ms-search-repair",
		"ms-secondary-screen-controller",
		"ms-secondary-screen-setup",
		"ms-settings",
		"ms-settings-airplanemode",
		"ms-settings-bluetooth",
		"ms-settings-camera",
		"ms-settings-cellular",
		"ms-settings-cloudstorage",
		"ms-settings-connectabledevices",
		"ms-settings-displays-topology",
		"ms-settings-emailandaccounts",
		"ms-settings-language",
		"ms-settings-location",
		"ms-settings-lock",
		"ms-settings-nfctransactions",
		"ms-settings-notifications",
		"ms-settings-power",
		"ms-settings-privacy",
		"ms-settings-proximity",
		"ms-settings-screenrotation",
		"ms-settings-wifi",
		"ms-settings-workplace",
		"ms-spd",
		"ms-stickers",
		"ms-sttoverlay",
		"ms-transit-to",
		"ms-useractivityset",
		"ms-virtualtouchpad",
		"ms-visio",
		"ms-walk-to",
		"ms-whiteboard",
		"ms-whiteboard-cmd",
		"ms-word",
		"msnim",
		"msrp",
		"msrps",
		"mss",
		"mt",
		"mtqp",
		"mumble",
		"mupdate",
		"mvn",
		"news",
		"nfs",
		"ni",
		"nih",
		"nntp",
		"notes",
		"num",
		"ocf",
		"oid",
		"onenote",
		"onenote-cmd",
		"opaquelocktoken",
		"openpgp4fpr",
		"otpauth",
		"p1",
		"pack",
		"palm",
		"paparazzi",
		"payment",
		"payto",
		"pkcs11",
		"platform",
		"pop",
		"pres",
		"prospero",
		"proxy",
		"pwid",
		"psyc",
		"pttp",
		"qb",
		"query",
		"quic-transport",
		"redis",
		"rediss",
		"reload",
		"res",
		"resource",
		"rmi",
		"rsync",
		"rtmfp",
		"rtmp",
		"rtsp",
		"rtsps",
		"rtspu",
		"sarif",
		"secondlife",
		"secret-token",
		"service",
		"session",
		"sftp",
		"sgn",
		"shc",
		"shttp",
		"sieve",
		"simpleledger",
		"simplex",
		"sip",
		"sips",
		"skype",
		"smb",
		"smp",
		"sms",
		"smtp",
		"snews",
		"snmp",
		"soap.beep",
		"soap.beeps",
		"soldat",
		"spiffe",
		"spotify",
		"ssb",
		"ssh",
		"starknet",
		"steam",
		"stun",
		"stuns",
		"submit",
		"svn",
		"swh",
		"swid",
		"swidpath",
		"tag",
		"taler",
		"teamspeak",
		"tel",
		"teliaeid",
		"telnet",
		"tftp",
		"things",
		"thismessage",
		"tip",
		"tn3270",
		"tool",
		"turn",
		"turns",
		"tv",
		"udp",
		"unreal",
		"upt",
		"urn",
		"ut2004",
		"uuid-in-package",
		"v-event",
		"vemmi",
		"ventrilo",
		"ves",
		"videotex",
		"vnc",
		"view-source",
		"vscode",
		"vscode-insiders",
		"vsls",
		"w3",
		"wais",
		"web3",
		"wcr",
		"webcal",
		"web+ap",
		"wifi",
		"wpid",
		"ws",
		"wss",
		"wtai",
		"wyciwyg",
		"xcon",
		"xcon-userid",
		"xfire",
		"xmlrpc.beep",
		"xmlrpc.beeps",
		"xmpp",
		"xri",
		"ymsgr",
		"z39.50",
		"z39.50r",
		"z39.50s",
	}
}
