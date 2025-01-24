package service

import (
	"context"
	"strings"

	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

func (svc *Service) GetOrbitSetupExperienceStatus(ctx context.Context, orbitNodeKey string, forceRelease bool) (*mdmlab.SetupExperienceStatusPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)
	host, err := svc.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading host by orbit node key")
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	// get the status of the bootstrap package deployment
	bootstrapPkg, err := svc.ds.GetHostMDMMacOSSetup(ctx, host.ID)
	if err != nil && !mdmlab.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get bootstrap package status")
	}

	// NOTE: bootstrapPkg can be nil if there was none to install.
	var bootstrapPkgResult *mdmlab.SetupExperienceBootstrapPackageResult
	if bootstrapPkg != nil {
		bootstrapPkgResult = &mdmlab.SetupExperienceBootstrapPackageResult{
			Name:   bootstrapPkg.BootstrapPackageName,
			Status: bootstrapPkg.BootstrapPackageStatus,
		}
	}

	// get the status of the configuration profiles
	cfgProfs, err := svc.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get configuration profiles status")
	}
	var cfgProfResults []*mdmlab.SetupExperienceConfigurationProfileResult
	for _, prof := range cfgProfs {
		// NOTE: DDM profiles (declarations) are ignored because while a device is
		// awaiting to be released, it cannot process a DDM session (at least
		// that's what we noticed during testing).
		if strings.HasPrefix(prof.ProfileUUID, mdmlab.MDMAppleDeclarationUUIDPrefix) {
			continue
		}

		status := mdmlab.MDMDeliveryPending
		if prof.Status != nil {
			status = *prof.Status
		}
		cfgProfResults = append(cfgProfResults, &mdmlab.SetupExperienceConfigurationProfileResult{
			ProfileUUID: prof.ProfileUUID,
			Name:        prof.Name,
			Status:      status,
		})
	}

	// AccountConfiguration covers the (optional) command to setup SSO.
	adminTeamFilter := mdmlab.TeamFilter{
		User: &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
	}
	acctCmds, err := svc.ds.ListMDMCommands(ctx, adminTeamFilter, &mdmlab.MDMCommandListOptions{
		Filters: mdmlab.MDMCommandFilters{
			HostIdentifier: host.UUID,
			RequestType:    "AccountConfiguration",
		},
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list AccountConfiguration commands")
	}

	var acctCfgResult *mdmlab.SetupExperienceAccountConfigurationResult
	if len(acctCmds) > 0 {
		// there may be more than one if e.g. the worker job that sends them had to
		// retry, but they would all be processed anyway so we can only care about
		// the first one.
		acctCfgResult = &mdmlab.SetupExperienceAccountConfigurationResult{
			CommandUUID: acctCmds[0].CommandUUID,
			Status:      acctCmds[0].Status,
		}
	}

	// get status of software installs and script execution
	res, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
	}

	payload := &mdmlab.SetupExperienceStatusPayload{
		BootstrapPackage:      bootstrapPkgResult,
		ConfigurationProfiles: cfgProfResults,
		AccountConfiguration:  acctCfgResult,
		Software:              make([]*mdmlab.SetupExperienceStatusResult, 0),
		OrgLogoURL:            appCfg.OrgInfo.OrgLogoURLLightBackground,
	}
	for _, r := range res {
		if r.IsForScript() {
			payload.Script = r
		}

		if r.IsForSoftware() {
			payload.Software = append(payload.Software, r)
		}
	}

	if forceRelease || isDeviceReadyForRelease(payload) {
		manual, err := isDeviceReleasedManually(ctx, svc.ds, host)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "check if device is released manually")
		}
		if manual {
			return payload, nil
		}

		// otherwise the device is not released manually, proceed with automatic
		// release
		if forceRelease {
			level.Warn(svc.logger).Log("msg", "force-releasing device, DEP enrollment commands, profiles, software installs and script execution may not have all completed", "host_uuid", host.UUID)
		} else {
			level.Info(svc.logger).Log("msg", "releasing device, all DEP enrollment commands, profiles, software installs and script execution have completed", "host_uuid", host.UUID)
		}

		// Host will be marked as no longer "awaiting configuration" in the command handler
		if err := svc.mdmAppleCommander.DeviceConfigured(ctx, host.UUID, uuid.NewString()); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to enqueue DeviceConfigured command")
		}

	}

	_, err = svc.SetupExperienceNextStep(ctx, host.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting next step for host setup experience")
	}

	return payload, nil
}

func isDeviceReleasedManually(ctx context.Context, ds mdmlab.Datastore, host *mdmlab.Host) (bool, error) {
	var manualRelease bool
	if host.TeamID == nil {
		ac, err := ds.AppConfig(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get AppConfig to read enable_release_device_manually")
		}
		manualRelease = ac.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
	} else {
		tm, err := ds.Team(ctx, *host.TeamID)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "get Team to read enable_release_device_manually")
		}
		manualRelease = tm.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value
	}
	return manualRelease, nil
}

func isDeviceReadyForRelease(payload *mdmlab.SetupExperienceStatusPayload) bool {
	// default to "do release" and return false as soon as we find a reason not
	// to.

	if payload.BootstrapPackage != nil {
		if payload.BootstrapPackage.Status != mdmlab.MDMBootstrapPackageFailed &&
			payload.BootstrapPackage.Status != mdmlab.MDMBootstrapPackageInstalled {
			// bootstrap package is still pending, not ready for release
			return false
		}
	}

	if payload.AccountConfiguration != nil {
		if payload.AccountConfiguration.Status != mdmlab.MDMAppleStatusAcknowledged &&
			payload.AccountConfiguration.Status != mdmlab.MDMAppleStatusError &&
			payload.AccountConfiguration.Status != mdmlab.MDMAppleStatusCommandFormatError {
			// account configuration command is still pending, not ready for release
			return false
		}
	}

	for _, prof := range payload.ConfigurationProfiles {
		if prof.Status != mdmlab.MDMDeliveryFailed &&
			prof.Status != mdmlab.MDMDeliveryVerifying &&
			prof.Status != mdmlab.MDMDeliveryVerified {
			// profile is still pending, not ready for release
			return false
		}
	}

	for _, sw := range payload.Software {
		if sw.Status != mdmlab.SetupExperienceStatusFailure &&
			sw.Status != mdmlab.SetupExperienceStatusSuccess {
			// software is still pending, not ready for release
			return false
		}
	}

	if payload.Script != nil {
		if payload.Script.Status != mdmlab.SetupExperienceStatusFailure &&
			payload.Script.Status != mdmlab.SetupExperienceStatusSuccess {
			// script is still pending, not ready for release
			return false
		}
	}

	return true
}
