package service

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/it-laborato/MDM_Lab/server/contexts/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	hostctx "github.com/it-laborato/MDM_Lab/server/contexts/host"
	"github.com/it-laborato/MDM_Lab/server/contexts/logging"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	apple_mdm "github.com/it-laborato/MDM_Lab/server/mdm/apple"
	mdmcrypto "github.com/it-laborato/MDM_Lab/server/mdm/crypto"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/go-kit/log/level"
)

/////////////////////////////////////////////////////////////////////////////////
// Ping device endpoint
/////////////////////////////////////////////////////////////////////////////////

type devicePingRequest struct{}

type deviceAuthPingRequest struct {
	Token string `url:"token"`
}

func (r *deviceAuthPingRequest) deviceAuthToken() string {
	return r.Token
}

type devicePingResponse struct{}

func (r devicePingResponse) error() error { return nil }

func (r devicePingResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, mdmlab.GetServerDeviceCapabilities())
}

// NOTE: we're intentionally not reading the capabilities header in this
// endpoint as is unauthenticated and we don't want to trust whatever comes in
// there.
func devicePingEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	svc.DisableAuthForPing(ctx)
	return devicePingResponse{}, nil
}

func (svc *Service) DisableAuthForPing(ctx context.Context) {
	// skipauth: this endpoint is intentionally public to allow devices to ping
	// the server and among other things, get the mdmlab.Capabilities header to
	// determine which capabilities are enabled in the server.
	svc.authz.SkipAuthorization(ctx)
}

/////////////////////////////////////////////////////////////////////////////////
// MDMlab Desktop endpoints
/////////////////////////////////////////////////////////////////////////////////

type mdmlabDesktopResponse struct {
	Err error `json:"error,omitempty"`
	mdmlab.DesktopSummary
}

func (r mdmlabDesktopResponse) error() error { return r.Err }

type getMDMlabDesktopRequest struct {
	Token string `url:"token"`
}

func (r *getMDMlabDesktopRequest) deviceAuthToken() string {
	return r.Token
}

// getMDMlabDesktopEndpoint is meant to be the only API endpoint used by MDMlab Desktop. This
// endpoint should not include any kind of identifying information about the host.
func getMDMlabDesktopEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	sum, err := svc.GetMDMlabDesktopSummary(ctx)
	if err != nil {
		return mdmlabDesktopResponse{Err: err}, nil
	}
	return mdmlabDesktopResponse{DesktopSummary: sum}, nil
}

func (svc *Service) GetMDMlabDesktopSummary(ctx context.Context) (mdmlab.DesktopSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return mdmlab.DesktopSummary{}, mdmlab.ErrMissingLicense
}

/////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Host
/////////////////////////////////////////////////////////////////////////////////

type getDeviceHostRequest struct {
	Token           string `url:"token"`
	ExcludeSoftware bool   `query:"exclude_software,optional"`
}

func (r *getDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceHostResponse struct {
	Host                      *HostDetailResponse      `json:"host"`
	SelfService               bool                     `json:"self_service"`
	OrgLogoURL                string                   `json:"org_logo_url"`
	OrgLogoURLLightBackground string                   `json:"org_logo_url_light_background"`
	OrgContactURL             string                   `json:"org_contact_url"`
	Err                       error                    `json:"error,omitempty"`
	License                   mdmlab.LicenseInfo        `json:"license"`
	GlobalConfig              mdmlab.DeviceGlobalConfig `json:"global_config"`
}

func (r getDeviceHostResponse) error() error { return r.Err }

func getDeviceHostEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getDeviceHostRequest)
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceHostResponse{Err: err}, nil
	}

	// must still load the full host details, as it returns more information
	opts := mdmlab.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  false,
		ExcludeSoftware:  req.ExcludeSoftware,
	}
	hostDetails, err := svc.GetHost(ctx, host.ID, opts)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	resp, err := hostDetailResponseForHost(ctx, svc, hostDetails)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	// the org logo URL config is required by the frontend to render the page;
	// we need to be careful with what we return from AppConfig in the response
	// as this is a weakly authenticated endpoint (with the device auth token).
	ac, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	license, err := svc.License(ctx)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	resp.DEPAssignedToMDMlab = ptr.Bool(false)
	if ac.MDM.EnabledAndConfigured && license.IsPremium() {
		hdep, err := svc.GetHostDEPAssignment(ctx, host)
		if err != nil && !mdmlab.IsNotFound(err) {
			return getDeviceHostResponse{Err: err}, nil
		}
		resp.DEPAssignedToMDMlab = ptr.Bool(hdep.IsDEPAssignedToMDMlab())
	}

	softwareInventoryEnabled := ac.Features.EnableSoftwareInventory
	if resp.TeamID != nil {
		// load the team to get the device's team's software inventory config.
		tm, err := svc.GetTeam(ctx, *resp.TeamID)
		if err != nil && !mdmlab.IsNotFound(err) {
			return getDeviceHostResponse{Err: err}, nil
		}
		if tm != nil {
			softwareInventoryEnabled = tm.Config.Features.EnableSoftwareInventory // TODO: We should look for opportunities to fix the confusing name of the `global_config` object in the API response. Also, how can we better clarify/document the expected order of precedence for team and global feature flags?
		}
	}

	hasSelfService := false
	if softwareInventoryEnabled {
		hasSelfService, err = svc.HasSelfServiceSoftwareInstallers(ctx, host)
		if err != nil {
			return getDeviceHostResponse{Err: err}, nil
		}
	}

	deviceGlobalConfig := mdmlab.DeviceGlobalConfig{
		MDM: mdmlab.DeviceGlobalMDMConfig{
			// TODO(mna): It currently only returns the Apple enabled and configured,
			// regardless of the platform of the device. See
			// https://github.com/mdmlabdm/mdmlab/pull/19304#discussion_r1618792410.
			EnabledAndConfigured: ac.MDM.EnabledAndConfigured,
		},
		Features: mdmlab.DeviceFeatures{
			EnableSoftwareInventory: softwareInventoryEnabled,
		},
	}

	return getDeviceHostResponse{
		Host:          resp,
		OrgLogoURL:    ac.OrgInfo.OrgLogoURL,
		OrgContactURL: ac.OrgInfo.ContactURL,
		License:       *license,
		GlobalConfig:  deviceGlobalConfig,
		SelfService:   hasSelfService,
	}, nil
}

func (svc *Service) GetHostDEPAssignment(ctx context.Context, host *mdmlab.Host) (*mdmlab.HostDEPAssignment, error) {
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken)
	if !alreadyAuthd {
		if err := svc.authz.Authorize(ctx, host, mdmlab.ActionRead); err != nil {
			return nil, err
		}
	}
	return svc.ds.GetHostDEPAssignment(ctx, host.ID)
}

// AuthenticateDevice returns the host identified by the device authentication
// token, along with a boolean indicating if debug logging is enabled for that
// host.
func (svc *Service) AuthenticateDevice(ctx context.Context, authToken string) (*mdmlab.Host, bool, error) {
	const deviceAuthTokenTTL = time.Hour
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if authToken == "" {
		return nil, false, ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("authentication error: missing device authentication token"))
	}

	host, err := svc.ds.LoadHostByDeviceAuthToken(ctx, authToken, deviceAuthTokenTTL)
	switch {
	case err == nil:
		// OK
	case mdmlab.IsNotFound(err):
		return nil, false, ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("authentication error: invalid device authentication token"))
	default:
		return nil, false, ctxerr.Wrap(ctx, err, "authenticate device")
	}

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

/////////////////////////////////////////////////////////////////////////////////
// Refetch Current Device's Host
/////////////////////////////////////////////////////////////////////////////////

type refetchDeviceHostRequest struct {
	Token string `url:"token"`
}

func (r *refetchDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

func refetchDeviceHostEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return refetchHostResponse{Err: err}, nil
	}

	err := svc.RefetchHost(ctx, host.ID)
	if err != nil {
		return refetchHostResponse{Err: err}, nil
	}
	return refetchHostResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Current Device's Host Device Mappings
////////////////////////////////////////////////////////////////////////////////

type listDeviceHostDeviceMappingRequest struct {
	Token string `url:"token"`
}

func (r *listDeviceHostDeviceMappingRequest) deviceAuthToken() string {
	return r.Token
}

func listDeviceHostDeviceMappingEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return listHostDeviceMappingResponse{Err: err}, nil
	}

	dms, err := svc.ListHostDeviceMapping(ctx, host.ID)
	if err != nil {
		return listHostDeviceMappingResponse{Err: err}, nil
	}
	return listHostDeviceMappingResponse{HostID: host.ID, DeviceMapping: dms}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Macadmins
////////////////////////////////////////////////////////////////////////////////

type getDeviceMacadminsDataRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceMacadminsDataRequest) deviceAuthToken() string {
	return r.Token
}

func getDeviceMacadminsDataEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return getMacadminsDataResponse{Err: err}, nil
	}

	data, err := svc.MacadminsData(ctx, host.ID)
	if err != nil {
		return getMacadminsDataResponse{Err: err}, nil
	}
	return getMacadminsDataResponse{Macadmins: data}, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Current Device's Policies
////////////////////////////////////////////////////////////////////////////////

type listDevicePoliciesRequest struct {
	Token string `url:"token"`
}

func (r *listDevicePoliciesRequest) deviceAuthToken() string {
	return r.Token
}

type listDevicePoliciesResponse struct {
	Err      error               `json:"error,omitempty"`
	Policies []*mdmlab.HostPolicy `json:"policies"`
}

func (r listDevicePoliciesResponse) error() error { return r.Err }

func listDevicePoliciesEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return listDevicePoliciesResponse{Err: err}, nil
	}

	data, err := svc.ListDevicePolicies(ctx, host)
	if err != nil {
		return listDevicePoliciesResponse{Err: err}, nil
	}

	return listDevicePoliciesResponse{Policies: data}, nil
}

func (svc *Service) ListDevicePolicies(ctx context.Context, host *mdmlab.Host) ([]*mdmlab.HostPolicy, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Transparency URL Redirect
////////////////////////////////////////////////////////////////////////////////

type transparencyURLRequest struct {
	Token string `url:"token"`
}

func (r *transparencyURLRequest) deviceAuthToken() string {
	return r.Token
}

type transparencyURLResponse struct {
	RedirectURL string `json:"-"` // used to control the redirect, see hijackRender method
	Err         error  `json:"error,omitempty"`
}

func (r transparencyURLResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (r transparencyURLResponse) error() error { return r.Err }

func transparencyURL(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	config, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return transparencyURLResponse{Err: err}, nil
	}

	license, err := svc.License(ctx)
	if err != nil {
		return transparencyURLResponse{Err: err}, nil
	}

	transparencyURL := mdmlab.DefaultTransparencyURL
	// MDMlab Premium license is required for custom transparency url
	if license.IsPremium() && config.MDMlabDesktop.TransparencyURL != "" {
		transparencyURL = config.MDMlabDesktop.TransparencyURL
	}

	return transparencyURLResponse{RedirectURL: transparencyURL}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Receive errors from the client
////////////////////////////////////////////////////////////////////////////////

type mdmlabdErrorRequest struct {
	Token string `url:"token"`
	mdmlab.MDMlabdError
}

func (f *mdmlabdErrorRequest) deviceAuthToken() string {
	return f.Token
}

// Since we're directly storing what we get in Redis, limit the request size to
// 5MB, this combined with the rate limit of this endpoint should be enough to
// prevent a malicious actor.
const maxMDMlabdErrorReportSize int64 = 5 * 1024 * 1024

func (f *mdmlabdErrorRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	limitedReader := io.LimitReader(r, maxMDMlabdErrorReportSize+1)
	decoder := json.NewDecoder(limitedReader)

	for {
		if err := decoder.Decode(&f.MDMlabdError); err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			return &mdmlab.BadRequestError{Message: "payload exceeds maximum accepted size"}
		} else if err != nil {
			return &mdmlab.BadRequestError{Message: "invalid payload"}
		}
	}

	return nil
}

type mdmlabdErrorResponse struct{}

func (r mdmlabdErrorResponse) error() error { return nil }

func mdmlabdError(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*mdmlabdErrorRequest)
	err := svc.LogMDMlabdError(ctx, req.MDMlabdError)
	if err != nil {
		return nil, err
	}
	return mdmlabdErrorResponse{}, nil
}

func (svc *Service) LogMDMlabdError(ctx context.Context, mdmlabdError mdmlab.MDMlabdError) error {
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
		return ctxerr.Wrap(ctx, mdmlab.NewPermissionError("forbidden: only device-authenticated hosts can access this endpoint"))
	}

	err := ctxerr.WrapWithData(ctx, mdmlabdError, "receive mdmlabd error", mdmlabdError.ToMap())
	level.Warn(svc.logger).Log(
		"msg",
		"mdmlabd error",
		"error",
		err,
	)
	// Send to Redis/telemetry (if enabled)
	ctxerr.Handle(ctx, err)

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Current Device's MDM Apple Enrollment Profile
////////////////////////////////////////////////////////////////////////////////

type getDeviceMDMManualEnrollProfileRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceMDMManualEnrollProfileRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceMDMManualEnrollProfileResponse struct {
	// Profile field is used in hijackRender for the response.
	Profile []byte

	Err error `json:"error,omitempty"`
}

func (r getDeviceMDMManualEnrollProfileResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	// make the browser download the content to a file
	w.Header().Add("Content-Disposition", `attachment; filename="mdmlab-mdm-enrollment-profile.mobileconfig"`)
	// explicitly set the content length before the write, so the caller can
	// detect short writes (if it fails to send the full content properly)
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Profile)), 10))
	// this content type will make macos open the profile with the proper application
	w.Header().Set("Content-Type", "application/x-apple-aspen-config; charset=utf-8")
	// prevent detection of content, obey the provided content-type
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if n, err := w.Write(r.Profile); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func (r getDeviceMDMManualEnrollProfileResponse) error() error { return r.Err }

func getDeviceMDMManualEnrollProfileEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	// this call ensures that the authentication was done, no need to actually
	// use the host
	if _, ok := hostctx.FromContext(ctx); !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}

	profile, err := svc.GetDeviceMDMAppleEnrollmentProfile(ctx)
	if err != nil {
		return getDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}
	return getDeviceMDMManualEnrollProfileResponse{Profile: profile}, nil
}

func (svc *Service) GetDeviceMDMAppleEnrollmentProfile(ctx context.Context) ([]byte, error) {
	// must be device-authenticated, no additional authorization is required
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewPermissionError("forbidden: only device-authenticated hosts can access this endpoint"))
	}

	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching app config")
	}

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
	}

	tmSecrets, err := svc.ds.GetEnrollSecrets(ctx, host.TeamID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, ctxerr.Wrap(ctx, err, "getting host team enroll secrets")
	}
	if len(tmSecrets) == 0 && host.TeamID != nil {
		tmSecrets, err = svc.ds.GetEnrollSecrets(ctx, nil)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, err, "getting no team enroll secrets")
		}
	}
	if len(tmSecrets) == 0 {
		return nil, &mdmlab.BadRequestError{Message: "unable to find an enroll secret to generate enrollment profile"}
	}

	enrollSecret := tmSecrets[0].Secret
	profBytes, err := apple_mdm.GenerateOTAEnrollmentProfileMobileconfig(cfg.OrgInfo.OrgName, cfg.MDMUrl(), enrollSecret)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating ota mobileconfig file for manual enrollment")
	}

	signed, err := mdmcrypto.Sign(ctx, profBytes, svc.ds)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing profile")
	}

	return signed, nil
}

////////////////////////////////////////////////////////////////////////////////
// Signal start of mdm migration on a device
////////////////////////////////////////////////////////////////////////////////

type deviceMigrateMDMRequest struct {
	Token string `url:"token"`
}

func (r *deviceMigrateMDMRequest) deviceAuthToken() string {
	return r.Token
}

type deviceMigrateMDMResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deviceMigrateMDMResponse) error() error { return r.Err }

func (r deviceMigrateMDMResponse) Status() int { return http.StatusNoContent }

func migrateMDMDeviceEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return deviceMigrateMDMResponse{Err: err}, nil
	}

	if err := svc.TriggerMigrateMDMDevice(ctx, host); err != nil {
		return deviceMigrateMDMResponse{Err: err}, nil
	}
	return deviceMigrateMDMResponse{}, nil
}

func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *mdmlab.Host) error {
	return mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Trigger linux key escrow
////////////////////////////////////////////////////////////////////////////////

type triggerLinuxDiskEncryptionEscrowRequest struct {
	Token string `url:"token"`
}

func (r *triggerLinuxDiskEncryptionEscrowRequest) deviceAuthToken() string {
	return r.Token
}

type triggerLinuxDiskEncryptionEscrowResponse struct {
	Err error `json:"error,omitempty"`
}

func (r triggerLinuxDiskEncryptionEscrowResponse) error() error { return r.Err }

func (r triggerLinuxDiskEncryptionEscrowResponse) Status() int { return http.StatusNoContent }

func triggerLinuxDiskEncryptionEscrowEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return triggerLinuxDiskEncryptionEscrowResponse{Err: err}, nil
	}

	if err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, host); err != nil {
		return triggerLinuxDiskEncryptionEscrowResponse{Err: err}, nil
	}
	return triggerLinuxDiskEncryptionEscrowResponse{}, nil
}

func (svc *Service) TriggerLinuxDiskEncryptionEscrow(ctx context.Context, host *mdmlab.Host) error {
	return mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Software
////////////////////////////////////////////////////////////////////////////////

type getDeviceSoftwareRequest struct {
	Token string `url:"token"`
	mdmlab.HostSoftwareTitleListOptions
}

func (r *getDeviceSoftwareRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceSoftwareResponse struct {
	Software []*mdmlab.HostSoftwareWithInstaller `json:"software"`
	Count    int                                `json:"count"`
	Meta     *mdmlab.PaginationMetadata          `json:"meta,omitempty"`
	Err      error                              `json:"error,omitempty"`
}

func (r getDeviceSoftwareResponse) error() error { return r.Err }

func getDeviceSoftwareEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, mdmlab.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceSoftwareResponse{Err: err}, nil
	}

	req := request.(*getDeviceSoftwareRequest)
	res, meta, err := svc.ListHostSoftware(ctx, host.ID, req.HostSoftwareTitleListOptions)
	if err != nil {
		return getDeviceSoftwareResponse{Err: err}, nil
	}
	if res == nil {
		res = []*mdmlab.HostSoftwareWithInstaller{}
	}
	return getDeviceSoftwareResponse{Software: res, Meta: meta, Count: int(meta.TotalResults)}, nil //nolint:gosec // dismiss G115
}
