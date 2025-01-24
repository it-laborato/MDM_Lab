package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com:it-laborato/MDM_Lab/server"
	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	hostctx "github.com:it-laborato/MDM_Lab/server/contexts/host"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/go-kit/log/level"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *mdmlab.Host) ([]*mdmlab.HostPolicy, error) {
	return svc.ds.ListPoliciesForHost(ctx, host)
}

// TriggerMigrateMDMDevice triggers the webhook associated with the MDM
// migration to MDMlab configuration. It is located in the ee package instead of
// the server/webhooks one because it is a MDMlab Premium only feature and for
// licensing reasons this needs to live under this package.
func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *mdmlab.Host) error {
	level.Debug(svc.logger).Log("msg", "trigger migration webhook", "host_id", host.ID,
		"refetch_critical_queries_until", host.RefetchCriticalQueriesUntil)

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.MDM.EnabledAndConfigured {
		return mdmlab.ErrMDMNotConfigured
	}

	if host.RefetchCriticalQueriesUntil != nil && host.RefetchCriticalQueriesUntil.After(svc.clock.Now()) {
		// the webhook has already been triggered successfully recently (within the
		// refetch critical queries delay), so return as if it did send it successfully
		// but do not re-send.
		level.Debug(svc.logger).Log("msg", "waiting for critical queries refetch, skip sending webhook",
			"host_id", host.ID)
		return nil
	}

	connected, err := svc.ds.IsHostConnectedToMDMlabMDM(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host is connected to MDMlab")
	}

	var bre mdmlab.BadRequestError
	switch {
	case !ac.MDM.MacOSMigration.Enable:
		bre.InternalErr = ctxerr.New(ctx, "macOS migration not enabled")
	case ac.MDM.MacOSMigration.WebhookURL == "":
		bre.InternalErr = ctxerr.New(ctx, "macOS migration webhook URL not configured")
	}

	mdmInfo, err := svc.ds.GetHostMDM(ctx, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching host mdm info")
	}

	manualMigrationEligible, err := mdmlab.IsEligibleForManualMigration(host, mdmInfo, connected)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking manual migration eligibility")
	}

	if !mdmlab.IsEligibleForDEPMigration(host, mdmInfo, connected) && !manualMigrationEligible {
		bre.InternalErr = ctxerr.New(ctx, "host not eligible for macOS migration")
	}

	if bre.InternalErr != nil {
		return &bre
	}

	p := mdmlab.MigrateMDMDeviceWebhookPayload{}
	p.Timestamp = time.Now().UTC()
	p.Host.ID = host.ID
	p.Host.UUID = host.UUID
	p.Host.HardwareSerial = host.HardwareSerial

	if err := server.PostJSONWithTimeout(ctx, ac.MDM.MacOSMigration.WebhookURL, p); err != nil {
		return ctxerr.Wrap(ctx, err, "posting macOS migration webhook")
	}

	// if the webhook was successfully triggered, we update the host to
	// constantly run the query to check if it has been unenrolled from its
	// existing third-party MDM.
	refetchUntil := svc.clock.Now().Add(mdmlab.RefetchMDMUnenrollCriticalQueryDuration)
	host.RefetchCriticalQueriesUntil = &refetchUntil
	if err := svc.ds.UpdateHostRefetchCriticalQueriesUntil(ctx, host.ID, &refetchUntil); err != nil {
		return ctxerr.Wrap(ctx, err, "save host with refetch critical queries timestamp")
	}

	return nil
}

func (svc *Service) GetMDMlabDesktopSummary(ctx context.Context) (mdmlab.DesktopSummary, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	var sum mdmlab.DesktopSummary

	host, ok := hostctx.FromContext(ctx)

	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return sum, err
	}

	hasSelfService, err := svc.ds.HasSelfServiceSoftwareInstallers(ctx, host.Platform, host.TeamID)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving self service software installers")
	}
	sum.SelfService = &hasSelfService

	r, err := svc.ds.FailingPoliciesCount(ctx, host)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving failing policies")
	}
	sum.FailingPolicies = &r

	appCfg, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	if appCfg.MDM.EnabledAndConfigured && appCfg.MDM.MacOSMigration.Enable {
		connected, err := svc.ds.IsHostConnectedToMDMlabMDM(ctx, host)
		if err != nil {
			return sum, ctxerr.Wrap(ctx, err, "checking if host is connected to MDMlab")
		}

		mdmInfo, err := svc.ds.GetHostMDM(ctx, host.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return sum, ctxerr.Wrap(ctx, err, "could not retrieve mdm info")
		}

		needsDEPEnrollment := mdmInfo != nil && !mdmInfo.Enrolled && host.IsDEPAssignedToMDMlab()

		if needsDEPEnrollment {
			sum.Notifications.RenewEnrollmentProfile = true
		}

		manualMigrationEligible, err := mdmlab.IsEligibleForManualMigration(host, mdmInfo, connected)
		if err != nil {
			return sum, ctxerr.Wrap(ctx, err, "checking manual migration eligibility")
		}

		if mdmlab.IsEligibleForDEPMigration(host, mdmInfo, connected) || manualMigrationEligible {
			sum.Notifications.NeedsMDMMigration = true
		}

	}

	// organization information
	sum.Config.OrgInfo.OrgName = appCfg.OrgInfo.OrgName
	sum.Config.OrgInfo.OrgLogoURL = appCfg.OrgInfo.OrgLogoURL
	sum.Config.OrgInfo.OrgLogoURLLightBackground = appCfg.OrgInfo.OrgLogoURLLightBackground
	sum.Config.OrgInfo.ContactURL = appCfg.OrgInfo.ContactURL

	// mdm information
	sum.Config.MDM.MacOSMigration.Mode = appCfg.MDM.MacOSMigration.Mode

	return sum, nil
}

func (svc *Service) TriggerLinuxDiskEncryptionEscrow(ctx context.Context, host *mdmlab.Host) error {
	if svc.ds.IsHostPendingEscrow(ctx, host.ID) {
		return nil
	}

	if err := svc.validateReadyForLinuxEscrow(ctx, host); err != nil {
		_ = svc.ds.ReportEscrowError(ctx, host.ID, err.Error())
		return err
	}

	return svc.ds.QueueEscrow(ctx, host.ID)
}

func (svc *Service) validateReadyForLinuxEscrow(ctx context.Context, host *mdmlab.Host) error {
	if !host.IsLUKSSupported() {
		return &mdmlab.BadRequestError{Message: "MDMlab does not yet support creating LUKS disk encryption keys on this platform."}
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	if host.TeamID == nil {
		if !ac.MDM.EnableDiskEncryption.Value {
			return &mdmlab.BadRequestError{Message: "Disk encryption is not enabled for hosts not assigned to a team."}
		}
	} else {
		tc, err := svc.ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			return err
		}
		if !tc.EnableDiskEncryption {
			return &mdmlab.BadRequestError{Message: "Disk encryption is not enabled for this host's team."}
		}
	}

	if host.DiskEncryptionEnabled == nil || !*host.DiskEncryptionEnabled {
		return &mdmlab.BadRequestError{Message: "Host's disk is not encrypted. Please encrypt your disk first."}
	}

	// We have to pull Orbit info because the auth context doesn't fill in host.OrbitVersion
	orbitInfo, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
	if err != nil {
		return err
	}

	if orbitInfo == nil || !mdmlab.IsAtLeastVersion(orbitInfo.Version, mdmlab.MinOrbitLUKSVersion) {
		return &mdmlab.BadRequestError{Message: "Your version of mdmlabd does not support creating disk encryption keys on Linux. Please upgrade mdmlabd, then click Refetch, then try again."}
	}

	return svc.ds.AssertHasNoEncryptionKeyStored(ctx, host.ID)
}
