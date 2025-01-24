// Package service holds the implementation of the mdmlab interface and HTTP
// endpoints for the API
package service

import (
	"context"
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/mixer/clock"
	"github.com:it-laborato/MDM_Lab/server/authz"
	"github.com:it-laborato/MDM_Lab/server/config"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	apple_mdm "github.com:it-laborato/MDM_Lab/server/mdm/apple"
	microsoft_mdm "github.com:it-laborato/MDM_Lab/server/mdm/microsoft"
	nanodep_storage "github.com:it-laborato/MDM_Lab/server/mdm/nanodep/storage"
	nanomdm_push "github.com:it-laborato/MDM_Lab/server/mdm/nanomdm/push"
	nanomdm_storage "github.com:it-laborato/MDM_Lab/server/mdm/nanomdm/storage"
	"github.com:it-laborato/MDM_Lab/server/service/async"
	"github.com:it-laborato/MDM_Lab/server/sso"
	kitlog "github.com/go-kit/log"
)

var _ mdmlab.Service = (*Service)(nil)

// Service is the struct implementing mdmlab.Service. Create a new one with NewService.
type Service struct {
	ds             mdmlab.Datastore
	task           *async.Task
	carveStore     mdmlab.CarveStore
	installerStore mdmlab.InstallerStore
	resultStore    mdmlab.QueryResultStore
	liveQueryStore mdmlab.LiveQueryStore
	logger         kitlog.Logger
	config         config.MDMlabConfig
	clock          clock.Clock

	osqueryLogWriter *OsqueryLogger

	mailService     mdmlab.MailService
	ssoSessionStore sso.SessionStore

	failingPolicySet  mdmlab.FailingPolicySet
	enrollHostLimiter mdmlab.EnrollHostLimiter

	authz *authz.Authorizer

	jitterMu *sync.Mutex
	jitterH  map[time.Duration]*jitterHashTable

	geoIP mdmlab.GeoIP

	*mdmlab.EnterpriseOverrides

	depStorage        nanodep_storage.AllDEPStorage
	mdmStorage        nanomdm_storage.AllStorage
	mdmPushService    nanomdm_push.Pusher
	mdmAppleCommander *apple_mdm.MDMAppleCommander

	cronSchedulesService mdmlab.CronSchedulesService

	wstepCertManager microsoft_mdm.CertManager
}

func (svc *Service) LookupGeoIP(ctx context.Context, ip string) *mdmlab.GeoLocation {
	return svc.geoIP.Lookup(ctx, ip)
}

func (svc *Service) SetEnterpriseOverrides(overrides mdmlab.EnterpriseOverrides) {
	svc.EnterpriseOverrides = &overrides
}

// OsqueryLogger holds osqueryd's status and result loggers.
type OsqueryLogger struct {
	// Status holds the osqueryd's status logger.
	//
	// See https://osquery.readthedocs.io/en/stable/deployment/logging/#status-logs
	Status mdmlab.JSONLogger
	// Result holds the osqueryd's result logger.
	//
	// See https://osquery.readthedocs.io/en/stable/deployment/logging/#results-logs
	Result mdmlab.JSONLogger
}

// NewService creates a new service from the config struct
func NewService(
	ctx context.Context,
	ds mdmlab.Datastore,
	task *async.Task,
	resultStore mdmlab.QueryResultStore,
	logger kitlog.Logger,
	osqueryLogger *OsqueryLogger,
	config config.MDMlabConfig,
	mailService mdmlab.MailService,
	c clock.Clock,
	sso sso.SessionStore,
	lq mdmlab.LiveQueryStore,
	carveStore mdmlab.CarveStore,
	installerStore mdmlab.InstallerStore,
	failingPolicySet mdmlab.FailingPolicySet,
	geoIP mdmlab.GeoIP,
	enrollHostLimiter mdmlab.EnrollHostLimiter,
	depStorage nanodep_storage.AllDEPStorage,
	mdmStorage mdmlab.MDMAppleStore,
	mdmPushService nanomdm_push.Pusher,
	cronSchedulesService mdmlab.CronSchedulesService,
	wstepCertManager microsoft_mdm.CertManager,
) (mdmlab.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	svc := &Service{
		ds:                ds,
		task:              task,
		carveStore:        carveStore,
		installerStore:    installerStore,
		resultStore:       resultStore,
		liveQueryStore:    lq,
		logger:            logger,
		config:            config,
		clock:             c,
		osqueryLogWriter:  osqueryLogger,
		mailService:       mailService,
		ssoSessionStore:   sso,
		failingPolicySet:  failingPolicySet,
		authz:             authorizer,
		jitterH:           make(map[time.Duration]*jitterHashTable),
		jitterMu:          new(sync.Mutex),
		geoIP:             geoIP,
		enrollHostLimiter: enrollHostLimiter,
		depStorage:        depStorage,
		// TODO: remove mdmStorage and mdmPushService when
		// we remove deprecated top-level service methods
		// from the prototype.
		mdmStorage:           mdmStorage,
		mdmPushService:       mdmPushService,
		mdmAppleCommander:    apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService),
		cronSchedulesService: cronSchedulesService,
		wstepCertManager:     wstepCertManager,
	}
	return validationMiddleware{svc, ds, sso}, nil
}

func (svc *Service) SendEmail(mail mdmlab.Email) error {
	return svc.mailService.SendEmail(mail)
}

type validationMiddleware struct {
	mdmlab.Service
	ds              mdmlab.Datastore
	ssoSessionStore sso.SessionStore
}

// getAssetURL simply returns the base url used for retrieving image assets from mdmlabdm.com.
func getAssetURL() template.URL {
	return template.URL("https://mdmlabdm.com/images/permanent")
}
