package service

import (
	"fmt"

	"github.com/mixer/clock"
	"github.com/it-laborato/MDM_Lab/server/authz"
	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	apple_mdm "github.com/it-laborato/MDM_Lab/server/mdm/apple"
	"github.com/it-laborato/MDM_Lab/server/mdm/nanodep/storage"
	"github.com/it-laborato/MDM_Lab/server/sso"
	kitlog "github.com/go-kit/log"
)

// Service wraps a free Service and implements additional premium functionality on top of it.
type Service struct {
	mdmlab.Service

	ds                    mdmlab.Datastore
	logger                kitlog.Logger
	config                config.MDMlabConfig
	clock                 clock.Clock
	authz                 *authz.Authorizer
	depStorage            storage.AllDEPStorage
	mdmAppleCommander     mdmlab.MDMAppleCommandIssuer
	ssoSessionStore       sso.SessionStore
	depService            *apple_mdm.DEPService
	profileMatcher        mdmlab.ProfileMatcher
	softwareInstallStore  mdmlab.SoftwareInstallerStore
	bootstrapPackageStore mdmlab.MDMBootstrapPackageStore
	distributedLock       mdmlab.Lock
	keyValueStore         mdmlab.KeyValueStore
}

func NewService(
	svc mdmlab.Service,
	ds mdmlab.Datastore,
	logger kitlog.Logger,
	config config.MDMlabConfig,
	mailService mdmlab.MailService,
	c clock.Clock,
	depStorage storage.AllDEPStorage,
	mdmAppleCommander mdmlab.MDMAppleCommandIssuer,
	sso sso.SessionStore,
	profileMatcher mdmlab.ProfileMatcher,
	softwareInstallStore mdmlab.SoftwareInstallerStore,
	bootstrapPackageStore mdmlab.MDMBootstrapPackageStore,
	distributedLock mdmlab.Lock,
	keyValueStore mdmlab.KeyValueStore,
) (*Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	eeservice := &Service{
		Service:               svc,
		ds:                    ds,
		logger:                logger,
		config:                config,
		clock:                 c,
		authz:                 authorizer,
		depStorage:            depStorage,
		mdmAppleCommander:     mdmAppleCommander,
		ssoSessionStore:       sso,
		depService:            apple_mdm.NewDEPService(ds, depStorage, logger),
		profileMatcher:        profileMatcher,
		softwareInstallStore:  softwareInstallStore,
		bootstrapPackageStore: bootstrapPackageStore,
		distributedLock:       distributedLock,
		keyValueStore:         keyValueStore,
	}

	// Override methods that can't be easily overriden via
	// embedding.
	svc.SetEnterpriseOverrides(mdmlab.EnterpriseOverrides{
		HostFeatures:                      eeservice.HostFeatures,
		TeamByIDOrName:                    eeservice.teamByIDOrName,
		UpdateTeamMDMDiskEncryption:       eeservice.updateTeamMDMDiskEncryption,
		MDMAppleEnableFileVaultAndEscrow:  eeservice.MDMAppleEnableFileVaultAndEscrow,
		MDMAppleDisableFileVaultAndEscrow: eeservice.MDMAppleDisableFileVaultAndEscrow,
		DeleteMDMAppleSetupAssistant:      eeservice.DeleteMDMAppleSetupAssistant,
		MDMAppleSyncDEPProfiles:           eeservice.mdmAppleSyncDEPProfiles,
		DeleteMDMAppleBootstrapPackage:    eeservice.DeleteMDMAppleBootstrapPackage,
		MDMWindowsEnableOSUpdates:         eeservice.mdmWindowsEnableOSUpdates,
		MDMWindowsDisableOSUpdates:        eeservice.mdmWindowsDisableOSUpdates,
		MDMAppleEditedAppleOSUpdates:      eeservice.mdmAppleEditedAppleOSUpdates,
		SetupExperienceNextStep:           eeservice.SetupExperienceNextStep,
		GetVPPTokenIfCanInstallVPPApps:    eeservice.GetVPPTokenIfCanInstallVPPApps,
		InstallVPPAppPostValidation:       eeservice.InstallVPPAppPostValidation,
	})

	return eeservice, nil
}
