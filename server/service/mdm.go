package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/docker/go-units"
	"github.com/it-laborato/MDM_Lab/pkg/mdmlabhttp"
	"github.com/it-laborato/MDM_Lab/server"
	"github.com/it-laborato/MDM_Lab/server/authz"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/contexts/license"
	"github.com/it-laborato/MDM_Lab/server/contexts/logging"
	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mdm"
	apple_mdm "github.com/it-laborato/MDM_Lab/server/mdm/apple"
	"github.com/it-laborato/MDM_Lab/server/mdm/assets"
	"github.com/it-laborato/MDM_Lab/server/mdm/cryptoutil"
	nanomdm "github.com/it-laborato/MDM_Lab/server/mdm/nanomdm/mdm"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/go-sql-driver/mysql"
)

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple
////////////////////////////////////////////////////////////////////////////////

type getAppleMDMResponse struct {
	*mdmlab.AppleMDM
	Err error `json:"error,omitempty"`
}

func (r getAppleMDMResponse) error() error { return r.Err }

func getAppleMDMEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	appleMDM, err := svc.GetAppleMDM(ctx)
	if err != nil {
		return getAppleMDMResponse{Err: err}, nil
	}

	return getAppleMDMResponse{AppleMDM: appleMDM}, nil
}

func (svc *Service) GetAppleMDM(ctx context.Context) (*mdmlab.AppleMDM, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleMDM{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	apns, err := assets.X509Cert(ctx, svc.ds, mdmlab.MDMAssetAPNSCert)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse certificate")
	}

	appleMDM := &mdmlab.AppleMDM{
		CommonName: apns.Subject.CommonName,
		Issuer:     apns.Issuer.CommonName,
		RenewDate:  apns.NotAfter,
	}
	if apns.SerialNumber != nil {
		appleMDM.SerialNumber = apns.SerialNumber.String()
	}

	return appleMDM, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple_bm
////////////////////////////////////////////////////////////////////////////////

type getAppleBMResponse struct {
	*mdmlab.AppleBM
	Err error `json:"error,omitempty"`
}

func (r getAppleBMResponse) error() error { return r.Err }

func getAppleBMEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	appleBM, err := svc.GetAppleBM(ctx)
	if err != nil {
		return getAppleBMResponse{Err: err}, nil
	}

	return getAppleBMResponse{AppleBM: appleBM}, nil
}

func (svc *Service) GetAppleBM(ctx context.Context) (*mdmlab.AppleBM, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/apple/request_csr
////////////////////////////////////////////////////////////////////////////////

type requestMDMAppleCSRRequest struct {
	EmailAddress string `json:"email_address"`
	Organization string `json:"organization"`
}

type requestMDMAppleCSRResponse struct {
	*mdmlab.AppleCSR
	Err error `json:"error,omitempty"`
}

func (r requestMDMAppleCSRResponse) error() error { return r.Err }

func requestMDMAppleCSREndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*requestMDMAppleCSRRequest)

	csr, err := svc.RequestMDMAppleCSR(ctx, req.EmailAddress, req.Organization)
	if err != nil {
		return requestMDMAppleCSRResponse{Err: err}, nil
	}
	return requestMDMAppleCSRResponse{
		AppleCSR: csr,
	}, nil
}

func (svc *Service) RequestMDMAppleCSR(ctx context.Context, email, org string) (*mdmlab.AppleCSR, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	if err := mdmlab.ValidateEmail(email); err != nil {
		if strings.TrimSpace(email) == "" {
			return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("email_address", "missing email address"))
		}
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("email_address", fmt.Sprintf("invalid email address: %v", err)))
	}
	if strings.TrimSpace(org) == "" {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("organization", "missing organization"))
	}

	// create the raw SCEP CA cert and key (creating before the CSR signing
	// request so that nothing can fail after the request is made, except for the
	// network during the response of course)
	scepCACert, scepCAKey, err := apple_mdm.NewSCEPCACertKey()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate SCEP CA cert and key")
	}

	// create the APNs CSR
	apnsCSR, apnsKey, err := apple_mdm.GenerateAPNSCSRKey(email, org)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate APNs CSR")
	}

	// request the signed APNs CSR from mdmlabdm.com
	client := mdmlabhttp.NewClient(mdmlabhttp.WithTimeout(10 * time.Second))
	if err := apple_mdm.GetSignedAPNSCSR(client, apnsCSR); err != nil {
		if ferr, ok := err.(apple_mdm.MDMlabWebsiteError); ok {
			status := http.StatusBadGateway
			if ferr.Status >= 400 && ferr.Status <= 499 {
				// TODO: mdmlabdm.com returns a genereric "Bad
				// Request" message, we should coordinate and
				// stablish a response schema from which we can get
				// the invalid field and use
				// mdmlab.NewInvalidArgumentError instead
				//
				// For now, since we have already validated
				// everything else, we assume that a 4xx
				// response is an email with an invalid domain
				return nil, ctxerr.Wrap(
					ctx,
					mdmlab.NewInvalidArgumentError(
						"email_address",
						fmt.Sprintf("this email address is not valid: %v", err),
					),
				)
			}
			return nil, ctxerr.Wrap(
				ctx,
				mdmlab.NewUserMessageError(
					fmt.Errorf("MDMlabDM CSR request failed: %w", err),
					status,
				),
			)
		}

		return nil, ctxerr.Wrap(ctx, err, "get signed CSR")
	}

	// PEM-encode the cert and keys
	scepCACertPEM := apple_mdm.EncodeCertPEM(scepCACert)
	scepCAKeyPEM := apple_mdm.EncodePrivateKeyPEM(scepCAKey)
	apnsKeyPEM := apple_mdm.EncodePrivateKeyPEM(apnsKey)

	return &mdmlab.AppleCSR{
		APNsKey:  apnsKeyPEM,
		SCEPCert: scepCACertPEM,
		SCEPKey:  scepCAKeyPEM,
	}, nil
}

func (svc *Service) VerifyMDMAppleConfigured(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return err
	}
	if !appCfg.MDM.EnabledAndConfigured {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return mdmlab.ErrMDMNotConfigured
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/setup/eula
////////////////////////////////////////////////////////////////////////////////

type createMDMEULARequest struct {
	EULA *multipart.FileHeader
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (createMDMEULARequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &mdmlab.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["eula"] == nil {
		return nil, &mdmlab.BadRequestError{
			Message:     "eula multipart field is required",
			InternalErr: err,
		}
	}

	return &createMDMEULARequest{
		EULA: r.MultipartForm.File["eula"][0],
	}, nil
}

type createMDMEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r createMDMEULAResponse) error() error { return r.Err }

func createMDMEULAEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*createMDMEULARequest)
	ff, err := req.EULA.Open()
	if err != nil {
		return createMDMEULAResponse{Err: err}, nil
	}
	defer ff.Close()

	if err := svc.MDMCreateEULA(ctx, req.EULA.Filename, ff); err != nil {
		return createMDMEULAResponse{Err: err}, nil
	}

	return createMDMEULAResponse{}, nil
}

func (svc *Service) MDMCreateEULA(ctx context.Context, name string, file io.ReadSeeker) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/setup/eula?token={token}
////////////////////////////////////////////////////////////////////////////////

type getMDMEULARequest struct {
	Token string `url:"token"`
}

type getMDMEULAResponse struct {
	Err error `json:"error,omitempty"`

	// fields used in hijackRender to build the response
	eula *mdmlab.MDMEULA
}

func (r getMDMEULAResponse) error() error { return r.Err }

func (r getMDMEULAResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.eula.Bytes)))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.eula.Bytes); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func getMDMEULAEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getMDMEULARequest)

	eula, err := svc.MDMGetEULABytes(ctx, req.Token)
	if err != nil {
		return getMDMEULAResponse{Err: err}, nil
	}

	return getMDMEULAResponse{eula: eula}, nil
}

func (svc *Service) MDMGetEULABytes(ctx context.Context, token string) (*mdmlab.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/setup/eula/{token}/metadata
////////////////////////////////////////////////////////////////////////////////

type getMDMEULAMetadataRequest struct{}

type getMDMEULAMetadataResponse struct {
	*mdmlab.MDMEULA
	Err error `json:"error,omitempty"`
}

func (r getMDMEULAMetadataResponse) error() error { return r.Err }

func getMDMEULAMetadataEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	eula, err := svc.MDMGetEULAMetadata(ctx)
	if err != nil {
		return getMDMEULAMetadataResponse{Err: err}, nil
	}

	return getMDMEULAMetadataResponse{MDMEULA: eula}, nil
}

func (svc *Service) MDMGetEULAMetadata(ctx context.Context) (*mdmlab.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/setup/eula
////////////////////////////////////////////////////////////////////////////////

type deleteMDMEULARequest struct {
	Token string `url:"token"`
}

type deleteMDMEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMEULAResponse) error() error { return r.Err }

func deleteMDMEULAEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*deleteMDMEULARequest)
	if err := svc.MDMDeleteEULA(ctx, req.Token); err != nil {
		return deleteMDMEULAResponse{Err: err}, nil
	}
	return deleteMDMEULAResponse{}, nil
}

func (svc *Service) MDMDeleteEULA(ctx context.Context, token string) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Windows MDM Middleware
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) VerifyMDMWindowsConfigured(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return err
	}

	// Windows MDM configuration setting
	if !appCfg.MDM.WindowsEnabledAndConfigured {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return mdmlab.ErrMDMNotConfigured
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Apple or Windows MDM Middleware
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) VerifyMDMAppleOrWindowsConfigured(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return err
	}

	// Apple or Windows MDM configuration setting
	if !appCfg.MDM.EnabledAndConfigured && !appCfg.MDM.WindowsEnabledAndConfigured {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return mdmlab.ErrMDMNotConfigured
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Run Apple or Windows MDM Command
////////////////////////////////////////////////////////////////////////////////

type runMDMCommandRequest struct {
	Command   string   `json:"command"`
	HostUUIDs []string `json:"host_uuids"`
}

type runMDMCommandResponse struct {
	*mdmlab.CommandEnqueueResult
	Err error `json:"error,omitempty"`
}

func (r runMDMCommandResponse) error() error { return r.Err }

func runMDMCommandEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*runMDMCommandRequest)
	result, err := svc.RunMDMCommand(ctx, req.Command, req.HostUUIDs)
	if err != nil {
		return runMDMCommandResponse{Err: err}, nil
	}
	return runMDMCommandResponse{
		CommandEnqueueResult: result,
	}, nil
}

func (svc *Service) RunMDMCommand(ctx context.Context, rawBase64Cmd string, hostUUIDs []string) (result *mdmlab.CommandEnqueueResult, err error) {
	hosts, err := svc.authorizeAllHostsTeams(ctx, hostUUIDs, mdmlab.ActionWrite, &mdmlab.MDMCommandAuthz{})
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		err := mdmlab.NewInvalidArgumentError("host_uuids", "No hosts targeted. Make sure you provide a valid UUID.").WithStatus(http.StatusNotFound)
		return nil, ctxerr.Wrap(ctx, err, "no host received")
	}

	connectedMap, err := svc.ds.AreHostsConnectedToMDMlabMDM(ctx, hosts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking if hosts are connected to MDMlab")
	}

	platforms := make(map[string]bool)
	for _, h := range hosts {
		if !connectedMap[h.UUID] {
			err := mdmlab.NewInvalidArgumentError("host_uuids", "Can't run the MDM command because one or more hosts have MDM turned off. Run the following command to see a list of hosts with MDM on: mdmlabctl get hosts --mdm.").WithStatus(http.StatusPreconditionFailed)
			return nil, ctxerr.Wrap(ctx, err, "check host mdm enrollment")
		}
		platforms[h.MDMlabPlatform()] = true
	}
	if len(platforms) != 1 {
		err := mdmlab.NewInvalidArgumentError("host_uuids", "All hosts must be on the same platform.")
		return nil, ctxerr.Wrap(ctx, err, "check host platform")
	}

	// it's a for loop but at this point it's guaranteed that the map has a single value.
	var commandPlatform string
	for platform := range platforms {
		commandPlatform = platform
	}
	if !mdmlab.MDMSupported(commandPlatform) {
		err := mdmlab.NewInvalidArgumentError("host_uuids", "Invalid platform. You can only run MDM commands on Windows or Apple hosts.")
		return nil, ctxerr.Wrap(ctx, err, "check host platform")
	}

	// check that the platform-specific MDM is enabled (not sure this check can
	// ever happen, since we verify that the hosts are enrolled, but just to be
	// safe)
	switch commandPlatform {
	case "windows":
		if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
			err := mdmlab.NewInvalidArgumentError("host_uuids", mdmlab.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			return nil, ctxerr.Wrap(ctx, err, "check windows MDM enabled")
		}
	default:
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			err := mdmlab.NewInvalidArgumentError("host_uuids", mdmlab.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			return nil, ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
		}
	}

	// We're supporting both padded and unpadded base64.
	rawXMLCmd, err := server.Base64DecodePaddingAgnostic(rawBase64Cmd)
	if err != nil {
		err = mdmlab.NewInvalidArgumentError("command", "unable to decode base64 command").WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "decode base64 command")
	}

	// the rest is platform-specific (validation of command payload, enqueueing, etc.)
	switch commandPlatform {
	case "windows":
		return svc.enqueueMicrosoftMDMCommand(ctx, rawXMLCmd, hostUUIDs)
	default:
		return svc.enqueueAppleMDMCommand(ctx, rawXMLCmd, hostUUIDs)
	}
}

var appleMDMPremiumCommands = map[string]bool{
	"EraseDevice": true,
	"DeviceLock":  true,
}

func (svc *Service) enqueueAppleMDMCommand(ctx context.Context, rawXMLCmd []byte, deviceIDs []string) (result *mdmlab.CommandEnqueueResult, err error) {
	cmd, err := nanomdm.DecodeCommand(rawXMLCmd)
	if err != nil {
		err = mdmlab.NewInvalidArgumentError("command", "unable to decode plist command").WithStatus(http.StatusUnsupportedMediaType)
		return nil, ctxerr.Wrap(ctx, err, "decode plist command")
	}

	if appleMDMPremiumCommands[strings.TrimSpace(cmd.Command.RequestType)] {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return nil, mdmlab.ErrMissingLicense
		}
	}

	if err := svc.mdmAppleCommander.EnqueueCommand(ctx, deviceIDs, string(rawXMLCmd)); err != nil {
		// if at least one UUID enqueued properly, return success, otherwise return
		// error
		var apnsErr *apple_mdm.APNSDeliveryError
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &apnsErr) {
			failedUUIDs := apnsErr.FailedUUIDs()
			if len(failedUUIDs) < len(deviceIDs) {
				// some hosts properly received the command, so return success, with the list
				// of failed uuids.
				return &mdmlab.CommandEnqueueResult{
					CommandUUID: cmd.CommandUUID,
					RequestType: cmd.Command.RequestType,
					FailedUUIDs: failedUUIDs,
				}, nil
			}
			// push failed for all hosts
			err := mdmlab.NewBadGatewayError("Apple push notificiation service", err)
			return nil, ctxerr.Wrap(ctx, err, "enqueue command")

		} else if errors.As(err, &mysqlErr) {
			// enqueue may fail with a foreign key constraint error 1452 when one of
			// the hosts provided is not enrolled in nano_enrollments. Detect when
			// that's the case and add information to the error.
			if mysqlErr.Number == mysqlerr.ER_NO_REFERENCED_ROW_2 {
				err := mdmlab.NewInvalidArgumentError(
					"device_ids",
					fmt.Sprintf("at least one of the hosts is not enrolled in MDM or is not an elegible device: %v", err),
				).WithStatus(http.StatusBadRequest)
				return nil, ctxerr.Wrap(ctx, err, "enqueue command")
			}
		}

		return nil, ctxerr.Wrap(ctx, err, "enqueue command")
	}
	return &mdmlab.CommandEnqueueResult{
		CommandUUID: cmd.CommandUUID,
		RequestType: cmd.Command.RequestType,
		Platform:    "darwin",
	}, nil
}

func (svc *Service) enqueueMicrosoftMDMCommand(ctx context.Context, rawXMLCmd []byte, deviceIDs []string) (result *mdmlab.CommandEnqueueResult, err error) {
	cmdMsg, err := mdmlab.ParseWindowsMDMCommand(rawXMLCmd)
	if err != nil {
		err = mdmlab.NewInvalidArgumentError("command", err.Error())
		return nil, ctxerr.Wrap(ctx, err, "decode SyncML command")
	}

	if cmdMsg.IsPremium() {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return nil, mdmlab.ErrMissingLicense
		}
	}

	winCmd := &mdmlab.MDMWindowsCommand{
		// TODO: using the provided ID to mimic Apple, but seems better if
		// we're full in control of it, what we should do?
		CommandUUID:  cmdMsg.CmdID.Value,
		RawCommand:   rawXMLCmd,
		TargetLocURI: cmdMsg.GetTargetURI(),
	}
	if err := svc.ds.MDMWindowsInsertCommandForHosts(ctx, deviceIDs, winCmd); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert pending windows mdm command")
	}

	return &mdmlab.CommandEnqueueResult{
		CommandUUID: winCmd.CommandUUID,
		RequestType: winCmd.TargetLocURI,
		Platform:    "windows",
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/commandresults
////////////////////////////////////////////////////////////////////////////////

type getMDMCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type getMDMCommandResultsResponse struct {
	Results []*mdmlab.MDMCommandResult `json:"results,omitempty"`
	Err     error                     `json:"error,omitempty"`
}

func (r getMDMCommandResultsResponse) error() error { return r.Err }

func getMDMCommandResultsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getMDMCommandResultsRequest)
	results, err := svc.GetMDMCommandResults(ctx, req.CommandUUID)
	if err != nil {
		return getMDMCommandResultsResponse{
			Err: err,
		}, nil
	}

	return getMDMCommandResultsResponse{
		Results: results,
	}, nil
}

func (svc *Service) GetMDMCommandResults(ctx context.Context, commandUUID string) ([]*mdmlab.MDMCommandResult, error) {
	// first, authorize that the user has the right to list hosts
	if err := svc.authz.Authorize(ctx, &mdmlab.Host{}, mdmlab.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}

	// check that command exists first, to return 404 on invalid commands
	// (the command may exist but have no results yet).
	p, err := svc.ds.GetMDMCommandPlatform(ctx, commandUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	var results []*mdmlab.MDMCommandResult
	switch p {
	case "darwin":
		results, err = svc.ds.GetMDMAppleCommandResults(ctx, commandUUID)
	case "windows":
		results, err = svc.ds.GetMDMWindowsCommandResults(ctx, commandUUID)
	default:
		// this should never happen, but just in case
		level.Debug(svc.logger).Log("msg", "unknown MDM command platform", "platform", p)
	}

	if err != nil {
		return nil, err
	}

	// now we can load the hosts (lite) corresponding to those command results,
	// and do the final authorization check with the proper team(s). Include observers,
	// as they are able to view command results for their teams' hosts.
	filter := mdmlab.TeamFilter{User: vc.User, IncludeObserver: true}
	hostUUIDs := make([]string, len(results))
	for i, res := range results {
		hostUUIDs[i] = res.HostUUID
	}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, hostUUIDs)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		// do not return 404 here, as it's possible for a command to not have
		// results yet
		return nil, nil
	}

	// collect the team IDs and verify that the user has access to view commands
	// on all affected teams. Index the hosts by uuid for easly lookup as
	// afterwards we'll want to store the hostname on the returned results.
	hostsByUUID := make(map[string]*mdmlab.Host, len(hosts))
	teamIDs := make(map[uint]bool)
	for _, h := range hosts {
		var id uint
		if h.TeamID != nil {
			id = *h.TeamID
		}
		teamIDs[id] = true
		hostsByUUID[h.UUID] = h
	}

	var commandAuthz mdmlab.MDMCommandAuthz
	for tmID := range teamIDs {
		commandAuthz.TeamID = &tmID
		if tmID == 0 {
			commandAuthz.TeamID = nil
		}

		if err := svc.authz.Authorize(ctx, commandAuthz, mdmlab.ActionRead); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	// add the hostnames to the results
	for _, res := range results {
		if h := hostsByUUID[res.HostUUID]; h != nil {
			res.Hostname = hostsByUUID[res.HostUUID].Hostname
		}
	}
	return results, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/commands
////////////////////////////////////////////////////////////////////////////////

type listMDMCommandsRequest struct {
	ListOptions    mdmlab.ListOptions `url:"list_options"`
	HostIdentifier string            `query:"host_identifier,optional"`
	RequestType    string            `query:"request_type,optional"`
}

type listMDMCommandsResponse struct {
	Results []*mdmlab.MDMCommand `json:"results"`
	Err     error               `json:"error,omitempty"`
}

func (r listMDMCommandsResponse) error() error { return r.Err }

func listMDMCommandsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*listMDMCommandsRequest)
	results, err := svc.ListMDMCommands(ctx, &mdmlab.MDMCommandListOptions{
		ListOptions: req.ListOptions,
		Filters:     mdmlab.MDMCommandFilters{HostIdentifier: req.HostIdentifier, RequestType: req.RequestType},
	})
	if err != nil {
		return listMDMCommandsResponse{
			Err: err,
		}, nil
	}

	return listMDMCommandsResponse{
		Results: results,
	}, nil
}

func (svc *Service) ListMDMCommands(ctx context.Context, opts *mdmlab.MDMCommandListOptions) ([]*mdmlab.MDMCommand, error) {
	// first, authorize that the user has the right to list hosts
	if err := svc.authz.Authorize(ctx, &mdmlab.Host{}, mdmlab.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}

	// get the list of commands so we know what hosts (and therefore what teams)
	// we're dealing with. Including the observers as they are allowed to view
	// MDM Apple commands.
	results, err := svc.ds.ListMDMCommands(ctx, mdmlab.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	}, opts)
	if err != nil {
		return nil, err
	}

	// collect the different team IDs and verify that the user has access to view
	// commands on all affected teams, do not assume that ListMDMCommands
	// only returned hosts that the user is authorized to view the command
	// results of (that is, always verify with our rego authz policy).
	teamIDs := make(map[uint]bool)
	for _, res := range results {
		var id uint
		if res.TeamID != nil {
			id = *res.TeamID
		}
		teamIDs[id] = true
	}

	// instead of returning an authz error if the user is not authorized for a
	// team, we remove those commands from the results (as we want to return
	// whatever the user is allowed to see). Since this can only be done after
	// retrieving the list of commands, this may result in returning less results
	// than requested, but it's ok - it's expected that the results retrieved
	// from the datastore will all be authorized for the user.
	var commandAuthz mdmlab.MDMCommandAuthz
	var authzErr error
	for tmID := range teamIDs {
		commandAuthz.TeamID = &tmID
		if tmID == 0 {
			commandAuthz.TeamID = nil
		}
		if err := svc.authz.Authorize(ctx, commandAuthz, mdmlab.ActionRead); err != nil {
			if authzErr == nil {
				authzErr = err
			}
			teamIDs[tmID] = false
		}
	}

	if authzErr != nil {
		level.Error(svc.logger).Log("err", "unauthorized to view some team commands", "details", authzErr)

		// filter-out the teams that the user is not allowed to view
		allowedResults := make([]*mdmlab.MDMCommand, 0, len(results))
		for _, res := range results {
			var id uint
			if res.TeamID != nil {
				id = *res.TeamID
			}
			if teamIDs[id] {
				allowedResults = append(allowedResults, res)
			}
		}
		results = allowedResults
	}

	if len(results) == 0 && opts.Filters.HostIdentifier != "" {
		_, err := svc.ds.HostLiteByIdentifier(ctx, opts.Filters.HostIdentifier)
		var nve mdmlab.NotFoundError
		if errors.As(err, &nve) {
			return nil, mdmlab.NewInvalidArgumentError("Invalid Host", mdmlab.HostIdentiferNotFound).WithStatus(http.StatusNotFound)
		}
	}

	return results, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/disk_encryption/summary
////////////////////////////////////////////////////////////////////////////////

type getMDMDiskEncryptionSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMDiskEncryptionSummaryResponse struct {
	*mdmlab.MDMDiskEncryptionSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMDiskEncryptionSummaryResponse) error() error { return r.Err }

func getMDMDiskEncryptionSummaryEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getMDMDiskEncryptionSummaryRequest)

	des, err := svc.GetMDMDiskEncryptionSummary(ctx, req.TeamID)
	if err != nil {
		return getMDMDiskEncryptionSummaryResponse{Err: err}, nil
	}

	return &getMDMDiskEncryptionSummaryResponse{
		MDMDiskEncryptionSummary: des,
	}, nil
}

func (svc *Service) GetMDMDiskEncryptionSummary(ctx context.Context, teamID *uint) (*mdmlab.MDMDiskEncryptionSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/profiles/summary (deprecated)
// GET /configuration_profiles/summary
////////////////////////////////////////////////////////////////////////////////

type getMDMProfilesSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMProfilesSummaryResponse struct {
	mdmlab.MDMProfilesSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMProfilesSummaryResponse) error() error { return r.Err }

func getMDMProfilesSummaryEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getMDMProfilesSummaryRequest)
	res := getMDMProfilesSummaryResponse{}

	as, err := svc.GetMDMAppleProfilesSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMAppleProfilesSummaryResponse{Err: err}, nil
	}

	ws, err := svc.GetMDMWindowsProfilesSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMProfilesSummaryResponse{Err: err}, nil
	}

	ls, err := svc.GetMDMLinuxProfilesSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMProfilesSummaryResponse{Err: err}, nil
	}

	res.Verified = as.Verified + ws.Verified + ls.Verified
	res.Verifying = as.Verifying + ws.Verifying
	res.Failed = as.Failed + ws.Failed + ls.Failed
	res.Pending = as.Pending + ws.Pending + ls.Pending

	return &res, nil
}

// authorizeAllHostsTeams is a helper function that loads the hosts
// corresponding to the hostUUIDs and authorizes the context user to execute
// the specified authzAction (e.g. mdmlab.ActionWrite) for all the hosts' teams
// with the specified authorizer, which is typically a struct that can set a
// TeamID field and defines an authorization subject, such as
// mdmlab.MDMCommandAuthz.
//
// On success, the list of hosts is returned (which may be empty, it is up to
// the caller to return an error if needed when no hosts are found).
func (svc *Service) authorizeAllHostsTeams(ctx context.Context, hostUUIDs []string, authzAction any, authorizer mdmlab.TeamIDSetter) ([]*mdmlab.Host, error) {
	// load hosts (lite) by uuids, check that the user has the rights to run
	// commands for every affected team.
	if err := svc.authz.Authorize(ctx, &mdmlab.Host{}, mdmlab.ActionSelectiveList); err != nil {
		return nil, err
	}

	// here we use a global admin as filter because we want to get all hosts that
	// correspond to those uuids. Only after we get those hosts will we check
	// authorization for the current user, for all teams affected by that host.
	// Without this, only hosts that the user can view would be returned and the
	// actual authorization check might only be done on a subset of the requsted
	// hosts.
	filter := mdmlab.TeamFilter{User: &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, hostUUIDs)
	if err != nil {
		return nil, err
	}

	// collect the team IDs and verify that the user has access to run commands
	// on all affected teams.
	teamIDs := make(map[uint]bool, len(hosts))
	for _, h := range hosts {
		var id uint
		if h.TeamID != nil {
			id = *h.TeamID
		}
		teamIDs[id] = true
	}

	for tmID := range teamIDs {
		authzTeamID := &tmID
		if tmID == 0 {
			authzTeamID = nil
		}
		authorizer.SetTeamID(authzTeamID)

		if err := svc.authz.Authorize(ctx, authorizer, authzAction); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}
	return hosts, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/profiles/{uuid}
////////////////////////////////////////////////////////////////////////////////

type getMDMConfigProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
	Alt         string `query:"alt,optional"`
}

type getMDMConfigProfileResponse struct {
	*mdmlab.MDMConfigProfilePayload
	Err error `json:"error,omitempty"`
}

func (r getMDMConfigProfileResponse) error() error { return r.Err }

func getMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getMDMConfigProfileRequest)

	downloadRequested := req.Alt == "media"
	var err error
	if isAppleProfileUUID(req.ProfileUUID) {
		// Apple config profile
		cp, err := svc.GetMDMAppleConfigProfile(ctx, req.ProfileUUID)
		if err != nil {
			return &getMDMConfigProfileResponse{Err: err}, nil
		}

		if downloadRequested {
			return downloadFileResponse{
				content:     cp.Mobileconfig,
				contentType: "application/x-apple-aspen-config",
				filename:    fmt.Sprintf("%s_%s.mobileconfig", time.Now().Format("2006-01-02"), strings.ReplaceAll(cp.Name, " ", "_")),
			}, nil
		}
		return &getMDMConfigProfileResponse{
			MDMConfigProfilePayload: mdmlab.NewMDMConfigProfilePayloadFromApple(cp),
		}, nil
	}

	if isAppleDeclarationUUID(req.ProfileUUID) {
		// TODO: we could potentially combined with the other service methods
		decl, err := svc.GetMDMAppleDeclaration(ctx, req.ProfileUUID)
		if err != nil {
			return &getMDMConfigProfileResponse{Err: err}, nil
		}

		if downloadRequested {
			return downloadFileResponse{
				content:     decl.RawJSON,
				contentType: "application/json",
				filename:    fmt.Sprintf("%s_%s.json", time.Now().Format("2006-01-02"), strings.ReplaceAll(decl.Name, " ", "_")),
			}, nil
		}
		return &getMDMConfigProfileResponse{
			MDMConfigProfilePayload: mdmlab.NewMDMConfigProfilePayloadFromAppleDDM(decl),
		}, nil
	}

	// Windows config profile
	cp, err := svc.GetMDMWindowsConfigProfile(ctx, req.ProfileUUID)
	if err != nil {
		return &getMDMConfigProfileResponse{Err: err}, nil
	}

	if downloadRequested {
		return downloadFileResponse{
			content:     cp.SyncML,
			contentType: "application/octet-stream", // not using the XML MIME type as a profile is not valid XML (a list of <Replace> elements)
			filename:    fmt.Sprintf("%s_%s.xml", time.Now().Format("2006-01-02"), strings.ReplaceAll(cp.Name, " ", "_")),
		}, nil
	}
	return &getMDMConfigProfileResponse{
		MDMConfigProfilePayload: mdmlab.NewMDMConfigProfilePayloadFromWindows(cp),
	}, nil
}

func (svc *Service) GetMDMWindowsConfigProfile(ctx context.Context, profileUUID string) (*mdmlab.MDMWindowsConfigProfile, error) {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &mdmlab.Team{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	cp, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// now we can do a specific authz check based on team id of profile before we
	// return the profile.
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: cp.TeamID}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	return cp, nil
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/profiles/{uuid}
////////////////////////////////////////////////////////////////////////////////

type deleteMDMConfigProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
}

type deleteMDMConfigProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMConfigProfileResponse) error() error { return r.Err }

func deleteMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*deleteMDMConfigProfileRequest)

	var err error
	if isAppleProfileUUID(req.ProfileUUID) { //nolint:gocritic // ignore ifElseChain
		err = svc.DeleteMDMAppleConfigProfile(ctx, req.ProfileUUID)
	} else if isAppleDeclarationUUID(req.ProfileUUID) {
		// TODO: we could potentially combined with the other service methods
		err = svc.DeleteMDMAppleDeclaration(ctx, req.ProfileUUID)
	} else {
		err = svc.DeleteMDMWindowsConfigProfile(ctx, req.ProfileUUID)
	}
	return &deleteMDMConfigProfileResponse{Err: err}, nil
}

func (svc *Service) DeleteMDMWindowsConfigProfile(ctx context.Context, profileUUID string) error {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &mdmlab.Team{}, mdmlab.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// check that Windows MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just macOS
	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		err := mdmlab.NewInvalidArgumentError("profile_uuid", mdmlab.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return ctxerr.Wrap(ctx, err, "check windows MDM enabled")
	}

	prof, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	var teamName string
	teamID := *prof.TeamID
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	// now we can do a specific authz check based on team id of profile before we delete the profile
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: prof.TeamID}, mdmlab.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// prevent deleting MDMlab-managed profiles (e.g., Windows OS Updates profile controlled by the OS Updates settings)
	mdmlabNames := mdm.MDMlabReservedProfileNames()
	if _, ok := mdmlabNames[prof.Name]; ok {
		err := &mdmlab.BadRequestError{Message: "Profiles managed by MDMlab can't be deleted using this endpoint."}
		return ctxerr.Wrap(ctx, err, "validate profile")
	}

	if err := svc.ds.DeleteMDMWindowsConfigProfile(ctx, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// cannot use the profile ID as it is now deleted
	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{teamID}, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &mdmlab.ActivityTypeDeletedWindowsProfile{
			TeamID:      actTeamID,
			TeamName:    actTeamName,
			ProfileName: prof.Name,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for delete mdm windows config profile")
	}

	return nil
}

// returns the numeric Apple profile ID and true if it is an Apple identifier,
// or 0 and false otherwise.
func isAppleProfileUUID(profileUUID string) bool {
	return strings.HasPrefix(profileUUID, "a")
}

func isAppleDeclarationUUID(profileUUID string) bool {
	return strings.HasPrefix(profileUUID, mdmlab.MDMAppleDeclarationUUIDPrefix)
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/profiles (Create Apple or Windows MDM Config Profile)
////////////////////////////////////////////////////////////////////////////////

type newMDMConfigProfileRequest struct {
	TeamID           uint
	Profile          *multipart.FileHeader
	LabelsIncludeAll []string
	LabelsIncludeAny []string
	LabelsExcludeAny []string
}

func (newMDMConfigProfileRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := newMDMConfigProfileRequest{}

	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &mdmlab.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	// add team_id
	val, ok := r.MultipartForm.Value["team_id"]
	if !ok || len(val) < 1 {
		// default is no team
		decoded.TeamID = 0
	} else {
		teamID, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, &mdmlab.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = uint(teamID) //nolint:gosec // dismiss G115
	}

	// add profile
	fhs, ok := r.MultipartForm.File["profile"]
	if !ok || len(fhs) < 1 {
		return nil, &mdmlab.BadRequestError{Message: "no file headers for profile"}
	}
	decoded.Profile = fhs[0]

	if decoded.Profile.Size > 1024*1024 {
		return nil, mdmlab.NewInvalidArgumentError("mdm", "maximum configuration profile file size is 1 MB")
	}

	// add labels
	var existsInclAll, existsInclAny, existsExclAny, existsDepr bool
	var deprecatedLabels []string
	decoded.LabelsIncludeAll, existsInclAll = r.MultipartForm.Value[string(mdmlab.LabelsIncludeAll)]
	decoded.LabelsIncludeAny, existsInclAny = r.MultipartForm.Value[string(mdmlab.LabelsIncludeAny)]
	decoded.LabelsExcludeAny, existsExclAny = r.MultipartForm.Value[string(mdmlab.LabelsExcludeAny)]
	deprecatedLabels, existsDepr = r.MultipartForm.Value["labels"]

	// validate that only one of the labels type is provided
	var count int
	for _, b := range []bool{existsInclAll, existsInclAny, existsExclAny, existsDepr} {
		if b {
			count++
		}
	}
	if count > 1 {
		return nil, &mdmlab.BadRequestError{Message: `Only one of "labels_exclude_any", "labels_include_all", "labels_include_any", or "labels" can be included.`}
	}
	if existsDepr {
		decoded.LabelsIncludeAll = deprecatedLabels
	}

	return &decoded, nil
}

type newMDMConfigProfileResponse struct {
	ProfileUUID string `json:"profile_uuid"`
	Err         error  `json:"error,omitempty"`
}

func (r newMDMConfigProfileResponse) error() error { return r.Err }

func newMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*newMDMConfigProfileRequest)

	ff, err := req.Profile.Open()
	if err != nil {
		return &newMDMConfigProfileResponse{Err: err}, nil
	}
	defer ff.Close()

	fileExt := filepath.Ext(req.Profile.Filename)
	profileName := strings.TrimSuffix(filepath.Base(req.Profile.Filename), fileExt)
	isMobileConfig := strings.EqualFold(fileExt, ".mobileconfig")
	isJSON := strings.EqualFold(fileExt, ".json")

	var labels []string
	var labelsMode mdmlab.MDMLabelsMode
	switch {
	case len(req.LabelsIncludeAny) > 0:
		labels = req.LabelsIncludeAny
		labelsMode = mdmlab.LabelsIncludeAny
	case len(req.LabelsExcludeAny) > 0:
		labels = req.LabelsExcludeAny
		labelsMode = mdmlab.LabelsExcludeAny
	default:
		// default include all
		labels = req.LabelsIncludeAll
		labelsMode = mdmlab.LabelsIncludeAll
	}

	if isMobileConfig || isJSON {
		// Then it's an Apple configuration file
		if isJSON {
			decl, err := svc.NewMDMAppleDeclaration(ctx, req.TeamID, ff, labels, profileName, labelsMode)
			if err != nil {
				return &newMDMConfigProfileResponse{Err: err}, nil
			}

			return &newMDMConfigProfileResponse{
				ProfileUUID: decl.DeclarationUUID,
			}, nil

		}

		cp, err := svc.NewMDMAppleConfigProfile(ctx, req.TeamID, ff, labels, labelsMode)
		if err != nil {
			return &newMDMConfigProfileResponse{Err: err}, nil
		}
		return &newMDMConfigProfileResponse{
			ProfileUUID: cp.ProfileUUID,
		}, nil
	}

	if isWindows := strings.EqualFold(fileExt, ".xml"); isWindows {
		cp, err := svc.NewMDMWindowsConfigProfile(ctx, req.TeamID, profileName, ff, labels, labelsMode)
		if err != nil {
			return &newMDMConfigProfileResponse{Err: err}, nil
		}
		return &newMDMConfigProfileResponse{
			ProfileUUID: cp.ProfileUUID,
		}, nil
	}

	err = svc.NewMDMUnsupportedConfigProfile(ctx, req.TeamID, req.Profile.Filename)
	return &newMDMConfigProfileResponse{Err: err}, nil
}

func (svc *Service) NewMDMUnsupportedConfigProfile(ctx context.Context, teamID uint, filename string) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: &teamID}, mdmlab.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// this is required because we need authorize to return the error, and
	// svc.authz is only available on the concrete Service struct, not on the
	// Service interface so it cannot be done in the endpoint itself.
	return &mdmlab.BadRequestError{Message: "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file."}
}

func (svc *Service) NewMDMWindowsConfigProfile(ctx context.Context, teamID uint, profileName string, r io.Reader, labels []string, labelsMembershipMode mdmlab.MDMLabelsMode) (*mdmlab.MDMWindowsConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: &teamID}, mdmlab.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// check that Windows MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just macOS
	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		err := mdmlab.NewInvalidArgumentError("profile", mdmlab.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "check windows MDM enabled")
	}

	var teamName string
	if teamID > 0 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &mdmlab.BadRequestError{
			Message:     "failed to read Windows config profile",
			InternalErr: err,
		})
	}

	cp := mdmlab.MDMWindowsConfigProfile{
		TeamID: &teamID,
		Name:   profileName,
		SyncML: b,
	}
	if err := cp.ValidateUserProvided(); err != nil {
		// this is not great, but since the validations are shared between the CLI
		// and the API, we must make some changes to error message here.
		msg := err.Error()
		if ix := strings.Index(msg, "To control these settings,"); ix >= 0 {
			msg = strings.TrimSpace(msg[:ix])
		}
		err := &mdmlab.BadRequestError{Message: "Couldn't upload. " + msg}
		return nil, ctxerr.Wrap(ctx, err, "validate profile")
	}

	labelMap, err := svc.validateProfileLabels(ctx, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating labels")
	}
	switch labelsMembershipMode {
	case mdmlab.LabelsIncludeAny:
		cp.LabelsIncludeAny = labelMap
	case mdmlab.LabelsExcludeAny:
		cp.LabelsExcludeAny = labelMap
	default:
		// default include all
		cp.LabelsIncludeAll = labelMap
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{string(cp.SyncML)}); err != nil {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("profile", err.Error()))
	}

	err = validateWindowsProfileMDMlabVariables(string(cp.SyncML))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("profile", err.Error()))
	}

	newCP, err := svc.ds.NewMDMWindowsConfigProfile(ctx, cp)
	if err != nil {
		var existsErr existsErrorInterface
		if errors.As(err, &existsErr) {
			err = mdmlab.NewInvalidArgumentError("profile", "Couldn't upload. A configuration profile with this name already exists.").
				WithStatus(http.StatusConflict)
		}
		return nil, ctxerr.Wrap(ctx, err)
	}

	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newCP.ProfileUUID}, nil); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &mdmlab.ActivityTypeCreatedWindowsProfile{
			TeamID:      actTeamID,
			TeamName:    actTeamName,
			ProfileName: newCP.Name,
		}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for create mdm windows config profile")
	}

	return newCP, nil
}

func validateWindowsProfileMDMlabVariables(contents string) error {
	if len(findMDMlabVariables(contents)) > 0 {
		return &mdmlab.BadRequestError{Message: "MDMlab variables ($FLEET_VAR_*) are not currently supported in Windows profiles"}
	}
	return nil
}

func (svc *Service) batchValidateProfileLabels(ctx context.Context, labelNames []string) (map[string]mdmlab.ConfigurationProfileLabel, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}

	labels, err := svc.ds.LabelIDsByName(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}

	uniqueNames := make(map[string]bool)
	for _, entry := range labelNames {
		if _, value := uniqueNames[entry]; !value {
			uniqueNames[entry] = true
		}
	}

	if len(labels) != len(uniqueNames) {
		return nil, &mdmlab.BadRequestError{
			Message:     "some or all the labels provided don't exist",
			InternalErr: fmt.Errorf("names provided: %v", labelNames),
		}
	}

	profLabels := make(map[string]mdmlab.ConfigurationProfileLabel)
	for labelName, labelID := range labels {
		profLabels[labelName] = mdmlab.ConfigurationProfileLabel{
			LabelName: labelName,
			LabelID:   labelID,
		}
	}
	return profLabels, nil
}

func (svc *Service) validateProfileLabels(ctx context.Context, labelNames []string) ([]mdmlab.ConfigurationProfileLabel, error) {
	labelMap, err := svc.batchValidateProfileLabels(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating profile labels")
	}

	var profLabels []mdmlab.ConfigurationProfileLabel
	for _, label := range labelMap {
		profLabels = append(profLabels, label)
	}
	return profLabels, nil
}

////////////////////////////////////////////////////////////////////////////////
// Batch Replace MDM Profiles
////////////////////////////////////////////////////////////////////////////////

type batchSetMDMProfilesRequest struct {
	TeamID        *uint                        `json:"-" query:"team_id,optional"`
	TeamName      *string                      `json:"-" query:"team_name,optional"`
	DryRun        bool                         `json:"-" query:"dry_run,optional"`        // if true, apply validation but do not save changes
	AssumeEnabled *bool                        `json:"-" query:"assume_enabled,optional"` // if true, assume MDM is enabled
	Profiles      backwardsCompatProfilesParam `json:"profiles"`
}

type backwardsCompatProfilesParam []mdmlab.MDMProfileBatchPayload

func (bcp *backwardsCompatProfilesParam) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if lookAhead := bytes.TrimSpace(data); len(lookAhead) > 0 && lookAhead[0] == '[' {
		// use []mdmlab.MDMProfileBatchPayload to prevent infinite recursion if we
		// use `backwardsCompatProfileSlice`
		var profs []mdmlab.MDMProfileBatchPayload
		if err := json.Unmarshal(data, &profs); err != nil {
			return fmt.Errorf("unmarshal profile spec. Error using new format: %w", err)
		}
		*bcp = profs
		return nil
	}

	var backwardsCompat map[string][]byte
	if err := json.Unmarshal(data, &backwardsCompat); err != nil {
		return fmt.Errorf("unmarshal profile spec. Error using old format: %w", err)
	}

	*bcp = make(backwardsCompatProfilesParam, 0, len(backwardsCompat))
	for name, contents := range backwardsCompat {
		*bcp = append(*bcp, mdmlab.MDMProfileBatchPayload{Name: name, Contents: contents})
	}
	return nil
}

type batchSetMDMProfilesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r batchSetMDMProfilesResponse) error() error { return r.Err }

func (r batchSetMDMProfilesResponse) Status() int { return http.StatusNoContent }

func batchSetMDMProfilesEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*batchSetMDMProfilesRequest)
	if err := svc.BatchSetMDMProfiles(
		ctx, req.TeamID, req.TeamName, req.Profiles, req.DryRun, false, req.AssumeEnabled,
	); err != nil {
		return batchSetMDMProfilesResponse{Err: err}, nil
	}
	return batchSetMDMProfilesResponse{}, nil
}

func (svc *Service) BatchSetMDMProfiles(
	ctx context.Context, tmID *uint, tmName *string, profiles []mdmlab.MDMProfileBatchPayload, dryRun, skipBulkPending bool,
	assumeEnabled *bool,
) error {
	var err error
	if tmID, tmName, err = svc.authorizeBatchProfiles(ctx, tmID, tmName); err != nil {
		return err
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	if assumeEnabled != nil {
		appCfg.MDM.WindowsEnabledAndConfigured = *assumeEnabled
	}

	// Process labels first, since we do not need to expand secrets in the profiles for this validation.
	labels := []string{}
	for i := range profiles {
		// from this point on (after this condition), only LabelsIncludeAll, LabelsIncludeAny or
		// LabelsExcludeAny need to be checked.
		if len(profiles[i].Labels) > 0 {
			// must update the struct in the slice directly, because we don't have a
			// pointer to it (it is a slice of structs, not of pointer to structs)
			profiles[i].LabelsIncludeAll = profiles[i].Labels
			profiles[i].Labels = nil
		}
		labels = append(labels, profiles[i].LabelsIncludeAll...)
		labels = append(labels, profiles[i].LabelsIncludeAny...)
		labels = append(labels, profiles[i].LabelsExcludeAny...)
	}
	labelMap, err := svc.batchValidateProfileLabels(ctx, labels)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating labels")
	}

	// We will not validate the profiles containing secret variables during dry run.
	// This is because the secret variables may not be available (or correct) in the gitops dry run.
	if dryRun {
		var profilesWithoutSecrets []mdmlab.MDMProfileBatchPayload
		for _, p := range profiles {
			if len(mdmlab.ContainsPrefixVars(string(p.Contents), mdmlab.ServerSecretPrefix)) == 0 {
				profilesWithoutSecrets = append(profilesWithoutSecrets, p)
			}
		}
		profiles = profilesWithoutSecrets
	}

	// Expand secret variables so that profiles can be properly validated.
	// Important: secret variables should never be exposed or saved in the database unencrypted
	// In order to map the expanded profiles back to the original profiles, we will use the index.
	profilesWithSecrets := make(map[int]mdmlab.MDMProfileBatchPayload, len(profiles))
	for i, p := range profiles {
		expanded, secretsUpdatedAt, err := svc.ds.ExpandEmbeddedSecretsAndUpdatedAt(ctx, string(p.Contents))
		if err != nil {
			return err
		}
		p.SecretsUpdatedAt = secretsUpdatedAt
		pCopy := p
		// If the profile does not contain secrets, then expanded and original content point to the same slice/memory location.
		pCopy.Contents = []byte(expanded)
		profilesWithSecrets[i] = pCopy
	}

	if err := validateProfiles(profilesWithSecrets); err != nil {
		return ctxerr.Wrap(ctx, err, "validating profiles")
	}

	appleProfiles, appleDecls, err := getAppleProfiles(ctx, tmID, appCfg, profilesWithSecrets, labelMap)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating macOS profiles")
	}

	windowsProfiles, err := getWindowsProfiles(ctx, tmID, appCfg, profilesWithSecrets, labelMap)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating Windows profiles")
	}

	if err := svc.validateCrossPlatformProfileNames(ctx, appleProfiles, windowsProfiles, appleDecls); err != nil {
		return ctxerr.Wrap(ctx, err, "validating cross-platform profile names")
	}

	if dryRun {
		return nil
	}

	err = validateMDMlabVariables(ctx, appleProfiles, windowsProfiles, appleDecls)
	if err != nil {
		return err
	}

	// Now that validation is done, we remove the exposed secret variables from the profiles
	appleProfilesSlice := make([]*mdmlab.MDMAppleConfigProfile, 0, len(appleProfiles))
	for i, p := range appleProfiles {
		p.Mobileconfig = profiles[i].Contents
		appleProfilesSlice = append(appleProfilesSlice, p)
	}
	appleDeclsSlice := make([]*mdmlab.MDMAppleDeclaration, 0, len(appleDecls))
	for i, p := range appleDecls {
		p.RawJSON = profiles[i].Contents
		appleDeclsSlice = append(appleDeclsSlice, p)
	}
	windowsProfilesSlice := make([]*mdmlab.MDMWindowsConfigProfile, 0, len(windowsProfiles))
	for i, p := range windowsProfiles {
		p.SyncML = profiles[i].Contents
		windowsProfilesSlice = append(windowsProfilesSlice, p)
	}

	var profUpdates mdmlab.MDMProfilesUpdates
	if profUpdates, err = svc.ds.BatchSetMDMProfiles(ctx, tmID, appleProfilesSlice, windowsProfilesSlice, appleDeclsSlice); err != nil {
		return ctxerr.Wrap(ctx, err, "setting config profiles")
	}

	// set pending status for windows profiles
	winProfUUIDs := []string{}
	for _, p := range windowsProfiles {
		winProfUUIDs = append(winProfUUIDs, p.ProfileUUID)
	}
	winUpdates, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, winProfUUIDs, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending windows host profiles")
	}

	// set pending status for apple profiles
	appleProfUUIDs := []string{}
	for _, p := range appleProfiles {
		appleProfUUIDs = append(appleProfUUIDs, p.ProfileUUID)
	}
	appleUpdates, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, appleProfUUIDs, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending apple host profiles")
	}
	updates := mdmlab.MDMProfilesUpdates{
		AppleConfigProfile:   profUpdates.AppleConfigProfile || winUpdates.AppleConfigProfile || appleUpdates.AppleConfigProfile,
		WindowsConfigProfile: profUpdates.WindowsConfigProfile || winUpdates.WindowsConfigProfile || appleUpdates.WindowsConfigProfile,
		AppleDeclaration:     profUpdates.AppleDeclaration || winUpdates.AppleDeclaration || appleUpdates.AppleDeclaration,
	}

	if updates.AppleConfigProfile {
		if err := svc.NewActivity(
			ctx, authz.UserFromContext(ctx), &mdmlab.ActivityTypeEditedMacosProfile{
				TeamID:   tmID,
				TeamName: tmName,
			}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for edited macos profile")
		}
	}
	if updates.WindowsConfigProfile {
		if err := svc.NewActivity(
			ctx, authz.UserFromContext(ctx), &mdmlab.ActivityTypeEditedWindowsProfile{
				TeamID:   tmID,
				TeamName: tmName,
			}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for edited windows profile")
		}
	}
	if updates.AppleDeclaration {
		if err := svc.NewActivity(
			ctx, authz.UserFromContext(ctx), &mdmlab.ActivityTypeEditedDeclarationProfile{
				TeamID:   tmID,
				TeamName: tmName,
			}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for edited macos declarations")
		}
	}

	return nil
}

func validateMDMlabVariables(ctx context.Context, appleProfiles map[int]*mdmlab.MDMAppleConfigProfile,
	windowsProfiles map[int]*mdmlab.MDMWindowsConfigProfile, appleDecls map[int]*mdmlab.MDMAppleDeclaration,
) error {
	var err error

	for _, p := range appleProfiles {
		err = validateConfigProfileMDMlabVariables(string(p.Mobileconfig))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "validating config profile MDMlab variables")
		}
	}
	for _, p := range windowsProfiles {
		err = validateWindowsProfileMDMlabVariables(string(p.SyncML))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "validating Windows profile MDMlab variables")
		}
	}
	for _, p := range appleDecls {
		err = validateDeclarationMDMlabVariables(string(p.RawJSON))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "validating declaration MDMlab variables")
		}
	}
	return nil
}

func (svc *Service) validateCrossPlatformProfileNames(ctx context.Context, appleProfiles map[int]*mdmlab.MDMAppleConfigProfile,
	windowsProfiles map[int]*mdmlab.MDMWindowsConfigProfile, appleDecls map[int]*mdmlab.MDMAppleDeclaration) error {
	// map all profile names to check for duplicates, regardless of platform; key is name, value is one of
	// ".mobileconfig" or ".json" or ".xml"
	extByName := make(map[string]string, len(appleProfiles)+len(windowsProfiles)+len(appleDecls))
	for i, p := range appleProfiles {
		if v, ok := extByName[p.Name]; ok {
			err := mdmlab.NewInvalidArgumentError(fmt.Sprintf("appleProfiles[%d]", i), fmtDuplicateNameErrMsg(p.Name, ".mobileconfig", v))
			return ctxerr.Wrap(ctx, err, "duplicate mobileconfig profile by name")
		}
		extByName[p.Name] = ".mobileconfig"
	}
	for i, p := range windowsProfiles {
		if v, ok := extByName[p.Name]; ok {
			err := mdmlab.NewInvalidArgumentError(fmt.Sprintf("windowsProfiles[%d]", i), fmtDuplicateNameErrMsg(p.Name, ".xml", v))
			return ctxerr.Wrap(ctx, err, "duplicate xml by name")
		}
		extByName[p.Name] = ".xml"
	}
	for i, p := range appleDecls {
		if v, ok := extByName[p.Name]; ok {
			err := mdmlab.NewInvalidArgumentError(fmt.Sprintf("appleDecls[%d]", i), fmtDuplicateNameErrMsg(p.Name, ".json", v))
			return ctxerr.Wrap(ctx, err, "duplicate json by name")
		}
		extByName[p.Name] = ".json"
	}

	return nil
}

func fmtDuplicateNameErrMsg(name, fileType1, fileType2 string) string {
	var part1 string
	switch fileType1 {
	case ".xml":
		part1 = "Windows .xml file name"
	case ".mobileconfig":
		part1 = "macOS .mobileconfig PayloadDisplayName"
	case ".json":
		part1 = "macOS .json file name"
	}

	var part2 string
	switch fileType2 {
	case ".xml":
		part2 = "Windows .xml file name"
	case ".mobileconfig":
		part2 = "macOS .mobileconfig PayloadDisplayName"
	case ".json":
		part2 = "macOS .json file name"
	}

	base := fmt.Sprintf(`Couldn’t edit custom_settings. More than one configuration profile have the same name '%s'`, name)
	detail := ` (%s).`
	switch {
	case part1 == part2:
		return fmt.Sprintf(base+detail, part1)
	case part1 != "" && part2 != "":
		return fmt.Sprintf(base+detail, fmt.Sprintf("%s or %s", part1, part2))
	case part1 != "" || part2 != "":
		return fmt.Sprintf(base+detail, part1+part2)
	default:
		return base + "." // should never happen
	}
}

func (svc *Service) authorizeBatchProfiles(ctx context.Context, tmID *uint, tmName *string) (*uint, *string, error) {
	if tmID != nil && tmName != nil {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return nil, nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("team_name", "cannot specify both team_id and team_name"))
	}
	if tmID != nil || tmName != nil {
		license, _ := license.FromContext(ctx)
		if !license.IsPremium() {
			field := "team_id"
			if tmName != nil {
				field = "team_name"
			}
			svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
			return nil, nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError(field, ErrMissingLicense.Error()))
		}
	}

	// if the team name is provided, load the corresponding team to get its id.
	// vice-versa, if the id is provided, load it to get the name (required for
	// the activity).
	if tmName != nil || tmID != nil {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, tmID, tmName)
		if err != nil {
			return nil, nil, err
		}
		if tmID == nil {
			tmID = &tm.ID
		} else {
			tmName = &tm.Name
		}
	}

	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: tmID}, mdmlab.ActionWrite); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err)
	}

	return tmID, tmName, nil
}

func getAppleProfiles(
	ctx context.Context,
	tmID *uint,
	appCfg *mdmlab.AppConfig,
	profiles map[int]mdmlab.MDMProfileBatchPayload,
	labelMap map[string]mdmlab.ConfigurationProfileLabel,
) (map[int]*mdmlab.MDMAppleConfigProfile, map[int]*mdmlab.MDMAppleDeclaration, error) {
	// any duplicate identifier or name in the provided set results in an error
	profs := make(map[int]*mdmlab.MDMAppleConfigProfile, len(profiles))
	decls := make(map[int]*mdmlab.MDMAppleDeclaration, len(profiles))
	// we need to keep track of the names and identifiers to check for duplicates so we will use
	// a map where the key is the name oridentifier and the value is either "mobileconfig" or
	// "declaration" to differentiate between the two types of profiles
	byName, byIdent := make(map[string]string, len(profiles)), make(map[string]string, len(profiles))
	for i, prof := range profiles {
		if mdm.GetRawProfilePlatform(prof.Contents) != "darwin" {
			continue
		}

		// Check for DDM files
		if mdm.GuessProfileExtension(prof.Contents) == "json" {
			rawDecl, err := mdmlab.GetRawDeclarationValues(prof.Contents)
			if err != nil {
				return nil, nil, err
			}

			if err := rawDecl.ValidateUserProvided(); err != nil {
				return nil, nil, err
			}

			mdmDecl := mdmlab.NewMDMAppleDeclaration(prof.Contents, tmID, prof.Name, rawDecl.Type, rawDecl.Identifier)
			mdmDecl.SecretsUpdatedAt = prof.SecretsUpdatedAt
			for _, labelName := range prof.LabelsIncludeAll {
				if lbl, ok := labelMap[labelName]; ok {
					declLabel := mdmlab.ConfigurationProfileLabel{
						LabelName:  lbl.LabelName,
						LabelID:    lbl.LabelID,
						RequireAll: true,
					}
					mdmDecl.LabelsIncludeAll = append(mdmDecl.LabelsIncludeAll, declLabel)
				}
			}
			for _, labelName := range prof.LabelsIncludeAny {
				if lbl, ok := labelMap[labelName]; ok {
					declLabel := mdmlab.ConfigurationProfileLabel{
						LabelName: lbl.LabelName,
						LabelID:   lbl.LabelID,
					}
					mdmDecl.LabelsIncludeAny = append(mdmDecl.LabelsIncludeAny, declLabel)
				}
			}
			for _, labelName := range prof.LabelsExcludeAny {
				if lbl, ok := labelMap[labelName]; ok {
					declLabel := mdmlab.ConfigurationProfileLabel{
						LabelName: lbl.LabelName,
						LabelID:   lbl.LabelID,
						Exclude:   true,
					}
					mdmDecl.LabelsExcludeAny = append(mdmDecl.LabelsExcludeAny, declLabel)
				}
			}

			v, ok := byIdent[mdmDecl.Identifier]
			switch {
			case !ok:
				byIdent[mdmDecl.Identifier] = "declaration"
			case v == "mobileconfig":
				return nil, nil, ctxerr.Wrap(ctx,
					mdmlab.NewInvalidArgumentError(mdmDecl.Identifier, "A configuration profile with this identifier already exists."),
					"duplicate mobileconfig profile by identifier")
			case v == "declaration":
				return nil, nil, ctxerr.Wrap(ctx,
					mdmlab.NewInvalidArgumentError(mdmDecl.Identifier, "A declaration profile with this identifier already exists."),
					"duplicate declaration profile by identifier")
			default:
				// this should never happen but just in case
				return nil, nil, ctxerr.Wrap(ctx,
					mdmlab.NewInvalidArgumentError(mdmDecl.Identifier, "A profile with this identifier already exists."),
					"duplicate identifier by identifier")
			}

			decls[i] = mdmDecl

			continue
		}

		mdmProf, err := mdmlab.NewMDMAppleConfigProfile(prof.Contents, tmID)
		mdmProf.SecretsUpdatedAt = prof.SecretsUpdatedAt
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError(prof.Name, err.Error()),
				"invalid mobileconfig profile")
		}

		for _, labelName := range prof.LabelsIncludeAll {
			if lbl, ok := labelMap[labelName]; ok {
				mdmLabel := mdmlab.ConfigurationProfileLabel{
					LabelName:  lbl.LabelName,
					LabelID:    lbl.LabelID,
					RequireAll: true,
				}
				mdmProf.LabelsIncludeAll = append(mdmProf.LabelsIncludeAll, mdmLabel)
			}
		}
		for _, labelName := range prof.LabelsIncludeAny {
			if lbl, ok := labelMap[labelName]; ok {
				mdmLabel := mdmlab.ConfigurationProfileLabel{
					LabelName: lbl.LabelName,
					LabelID:   lbl.LabelID,
				}
				mdmProf.LabelsIncludeAny = append(mdmProf.LabelsIncludeAny, mdmLabel)
			}
		}
		for _, labelName := range prof.LabelsExcludeAny {
			if lbl, ok := labelMap[labelName]; ok {
				mdmLabel := mdmlab.ConfigurationProfileLabel{
					LabelName: lbl.LabelName,
					LabelID:   lbl.LabelID,
					Exclude:   true,
				}
				mdmProf.LabelsExcludeAny = append(mdmProf.LabelsExcludeAny, mdmLabel)
			}
		}

		if err := mdmProf.ValidateUserProvided(); err != nil {
			return nil, nil, ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError(prof.Name, err.Error()))
		}

		if mdmProf.Name != prof.Name {
			return nil, nil, ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError(prof.Name, fmt.Sprintf("Couldn’t edit custom_settings. The name provided for the profile must match the profile PayloadDisplayName: %q", mdmProf.Name)),
				"duplicate mobileconfig profile by name")
		}

		// TODO: confirm error messages
		if _, ok := byName[mdmProf.Name]; ok {
			return nil, nil, ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError(prof.Name, fmt.Sprintf("Couldn’t edit custom_settings. More than one configuration profile have the same name (PayloadDisplayName): %q", mdmProf.Name)),
				"duplicate mobileconfig profile by name")
		}
		byName[mdmProf.Name] = "mobileconfig"

		// TODO: confirm error messages
		if _, ok := byIdent[mdmProf.Identifier]; ok {
			return nil, nil, ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError(prof.Name, fmt.Sprintf("Couldn’t edit custom_settings. More than one configuration profile have the same identifier (PayloadIdentifier): %q", mdmProf.Identifier)),
				"duplicate mobileconfig profile by identifier")
		}
		byIdent[mdmProf.Identifier] = "mobileconfig"

		profs[i] = mdmProf
	}

	if !appCfg.MDM.EnabledAndConfigured {
		// NOTE: in order to prevent an error when MDMlab MDM is not enabled but no
		// profile is provided, which can happen if a user runs `mdmlabctl get
		// config` and tries to apply that YAML, as it will contain an empty/null
		// custom_settings key, we just return a success response in this
		// situation.
		if len(profs) == 0 {
			return nil, nil, nil
		}

		return nil, nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("mdm", "cannot set custom settings: MDMlab MDM is not configured"))
	}

	return profs, decls, nil
}

func getWindowsProfiles(
	ctx context.Context,
	tmID *uint,
	appCfg *mdmlab.AppConfig,
	profiles map[int]mdmlab.MDMProfileBatchPayload,
	labelMap map[string]mdmlab.ConfigurationProfileLabel,
) (map[int]*mdmlab.MDMWindowsConfigProfile, error) {
	profs := make(map[int]*mdmlab.MDMWindowsConfigProfile, len(profiles))

	for i, profile := range profiles {
		if mdm.GetRawProfilePlatform(profile.Contents) != "windows" {
			continue
		}

		mdmProf := &mdmlab.MDMWindowsConfigProfile{
			TeamID: tmID,
			Name:   profile.Name,
			SyncML: profile.Contents,
		}
		for _, labelName := range profile.LabelsIncludeAll {
			if lbl, ok := labelMap[labelName]; ok {
				mdmLabel := mdmlab.ConfigurationProfileLabel{
					LabelName:  lbl.LabelName,
					LabelID:    lbl.LabelID,
					RequireAll: true,
				}
				mdmProf.LabelsIncludeAll = append(mdmProf.LabelsIncludeAll, mdmLabel)
			}
		}
		for _, labelName := range profile.LabelsIncludeAny {
			if lbl, ok := labelMap[labelName]; ok {
				mdmLabel := mdmlab.ConfigurationProfileLabel{
					LabelName: lbl.LabelName,
					LabelID:   lbl.LabelID,
				}
				mdmProf.LabelsIncludeAny = append(mdmProf.LabelsIncludeAny, mdmLabel)
			}
		}
		for _, labelName := range profile.LabelsExcludeAny {
			if lbl, ok := labelMap[labelName]; ok {
				mdmLabel := mdmlab.ConfigurationProfileLabel{
					LabelName: lbl.LabelName,
					LabelID:   lbl.LabelID,
					Exclude:   true,
				}
				mdmProf.LabelsExcludeAny = append(mdmProf.LabelsExcludeAny, mdmLabel)
			}
		}

		if err := mdmProf.ValidateUserProvided(); err != nil {
			return nil, ctxerr.Wrap(ctx,
				mdmlab.NewInvalidArgumentError(fmt.Sprintf("profiles[%s]", profile.Name), err.Error()))
		}

		profs[i] = mdmProf
	}

	if !appCfg.MDM.WindowsEnabledAndConfigured {
		// NOTE: in order to prevent an error when MDMlab MDM is not enabled but no
		// profile is provided, which can happen if a user runs `mdmlabctl get
		// config` and tries to apply that YAML, as it will contain an empty/null
		// custom_settings key, we just return a success response in this
		// situation.
		if len(profs) == 0 {
			return nil, nil
		}

		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("mdm", "cannot set custom settings: MDMlab MDM is not configured"))
	}

	return profs, nil
}

func validateProfiles(profiles map[int]mdmlab.MDMProfileBatchPayload) error {
	for _, profile := range profiles {
		// validate that only one of labels, labels_include_all and labels_exclude_any is provided.
		var count int
		for _, b := range []bool{
			len(profile.LabelsIncludeAll) > 0,
			len(profile.LabelsIncludeAny) > 0,
			len(profile.LabelsExcludeAny) > 0,
			len(profile.Labels) > 0,
		} {
			if b {
				count++
			}
		}
		if count > 1 {
			return mdmlab.NewInvalidArgumentError("mdm", `Couldn't edit custom_settings. For each profile, only one of "labels_exclude_any", "labels_include_all", "labels_include_any" or "labels" can be included.`)
		}

		if len(profile.Contents) > 1024*1024 {
			return mdmlab.NewInvalidArgumentError("mdm", "maximum configuration profile file size is 1 MB")
		}

		platform := mdm.GetRawProfilePlatform(profile.Contents)
		if platform != "darwin" && platform != "windows" {
			// We can only display a generic error message here because at this point
			// we don't know the file extension or whether the profile is intended
			// for macos_settings or windows_settings. We should expecte never see this
			// in practice because the client should be validating the profiles
			// before sending them to the server so the client can surface  more helpful
			// error messages to the user. However, we're validating again here just
			// in case the client is not working as expected.
			return mdmlab.NewInvalidArgumentError("mdm", fmt.Sprintf(
				"%s is not a valid macOS or Windows configuration profile. ", profile.Name)+
				"macOS profiles must be valid .mobileconfig or .json files. "+
				"Windows configuration profiles can only have <Replace> or <Add> top level elements.")
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/profiles (List profiles)
////////////////////////////////////////////////////////////////////////////////

type listMDMConfigProfilesRequest struct {
	TeamID      *uint             `query:"team_id,optional"`
	ListOptions mdmlab.ListOptions `url:"list_options"`
}

type listMDMConfigProfilesResponse struct {
	Meta     *mdmlab.PaginationMetadata        `json:"meta"`
	Profiles []*mdmlab.MDMConfigProfilePayload `json:"profiles"`
	Err      error                            `json:"error,omitempty"`
}

func (r listMDMConfigProfilesResponse) error() error { return r.Err }

func listMDMConfigProfilesEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*listMDMConfigProfilesRequest)

	profs, meta, err := svc.ListMDMConfigProfiles(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return &listMDMConfigProfilesResponse{Err: err}, nil
	}

	res := listMDMConfigProfilesResponse{Meta: meta, Profiles: profs}
	if profs == nil {
		// return empty json array instead of json null
		res.Profiles = []*mdmlab.MDMConfigProfilePayload{}
	}
	return &res, nil
}

func (svc *Service) ListMDMConfigProfiles(ctx context.Context, teamID *uint, opt mdmlab.ListOptions) ([]*mdmlab.MDMConfigProfilePayload, *mdmlab.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: teamID}, mdmlab.ActionRead); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err)
	}

	if teamID != nil && *teamID > 0 {
		// confirm that team exists
		if _, err := svc.ds.Team(ctx, *teamID); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err)
		}
	}

	// cursor-based pagination is not supported for profiles
	opt.After = ""
	// custom ordering is not supported, always by name
	opt.OrderKey = "name"
	opt.OrderDirection = mdmlab.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata for profiles
	opt.IncludeMetadata = true

	return svc.ds.ListMDMConfigProfiles(ctx, teamID, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Update MDM Disk encryption
////////////////////////////////////////////////////////////////////////////////

type updateDiskEncryptionRequest struct {
	TeamID               *uint `json:"team_id"`
	EnableDiskEncryption bool  `json:"enable_disk_encryption"`
}

type updateMDMDiskEncryptionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateMDMDiskEncryptionResponse) error() error { return r.Err }

func (r updateMDMDiskEncryptionResponse) Status() int { return http.StatusNoContent }

func updateDiskEncryptionEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*updateDiskEncryptionRequest)
	if err := svc.UpdateMDMDiskEncryption(ctx, req.TeamID, &req.EnableDiskEncryption); err != nil {
		return updateMDMDiskEncryptionResponse{Err: err}, nil
	}
	return updateMDMDiskEncryptionResponse{}, nil
}

func (svc *Service) UpdateMDMDiskEncryption(ctx context.Context, teamID *uint, enableDiskEncryption *bool) error {
	// TODO(mna): this should all move to the ee package when we remove the
	// `PATCH /api/v1/mdmlab/mdm/apple/settings` endpoint, but for now it's better
	// leave here so both endpoints can reuse the same logic.

	lic, _ := license.FromContext(ctx)
	if lic == nil || !lic.IsPremium() {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return mdmlab.ErrMissingLicense
	}

	// for historical reasons (the deprecated PATCH /mdm/apple/settings
	// endpoint), this uses an Apple-specific struct for authorization. Can be improved
	// once we remove the deprecated endpoint.
	if err := svc.authz.Authorize(ctx, mdmlab.MDMAppleSettingsPayload{TeamID: teamID}, mdmlab.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if teamID != nil {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, teamID, nil)
		if err != nil {
			return err
		}
		return svc.EnterpriseOverrides.UpdateTeamMDMDiskEncryption(ctx, tm, enableDiskEncryption)
	}
	return svc.updateAppConfigMDMDiskEncryption(ctx, enableDiskEncryption)
}

////////////////////////////////////////////////////////////////////////////////
// POST /hosts/{id:[0-9]+}/configuration_profiles/{profile_uuid}
////////////////////////////////////////////////////////////////////////////////

type resendHostMDMProfileRequest struct {
	HostID      uint   `url:"host_id"`
	ProfileUUID string `url:"profile_uuid"`
}

type resendHostMDMProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r resendHostMDMProfileResponse) error() error { return r.Err }

func (r resendHostMDMProfileResponse) Status() int { return http.StatusAccepted }

func resendHostMDMProfileEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*resendHostMDMProfileRequest)

	if err := svc.ResendHostMDMProfile(ctx, req.HostID, req.ProfileUUID); err != nil {
		return resendHostMDMProfileResponse{Err: err}, nil
	}

	return resendHostMDMProfileResponse{}, nil
}

func (svc *Service) ResendHostMDMProfile(ctx context.Context, hostID uint, profileUUID string) error {
	// first we perform a perform basic authz check, we use selective list action to include gitops users
	if err := svc.authz.Authorize(ctx, &mdmlab.Host{}, mdmlab.ActionSelectiveList); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// now we can do a specific authz check based on team id of the host before proceeding
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: host.TeamID}, mdmlab.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	var profileTeamID *uint
	var profileName string
	switch {
	case strings.HasPrefix(profileUUID, mdmlab.MDMAppleProfileUUIDPrefix):
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", mdmlab.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest), "check apple mdm enabled")
		}
		if host.Platform != "darwin" && host.Platform != "ios" && host.Platform != "ipados" {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", "Profile is not compatible with host platform."), "check host platform")
		}
		prof, err := svc.ds.GetMDMAppleConfigProfile(ctx, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting apple config profile")
		}
		profileTeamID = prof.TeamID
		profileName = prof.Name

	case strings.HasPrefix(profileUUID, mdmlab.MDMAppleDeclarationUUIDPrefix):
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", mdmlab.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest), "check apple mdm enabled")
		}
		if host.Platform != "darwin" && host.Platform != "ios" && host.Platform != "ipados" {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", "Profile is not compatible with host platform."), "check host platform")
		}
		decl, err := svc.ds.GetMDMAppleDeclaration(ctx, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting apple declaration")
		}
		profileTeamID = decl.TeamID
		profileName = decl.Name

	case strings.HasPrefix(profileUUID, mdmlab.MDMWindowsProfileUUIDPrefix):
		if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", mdmlab.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest), "check windows mdm enabled")
		}
		if host.Platform != "windows" {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", "Profile is not compatible with host platform."), "check host platform")
		}
		prof, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting windows config profile")
		}
		profileTeamID = prof.TeamID
		profileName = prof.Name

	default:
		return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", "Invalid profile UUID prefix.").WithStatus(http.StatusNotFound), "check profile UUID prefix")
	}

	// check again based on team id of profile before we proceeding
	if err := svc.authz.Authorize(ctx, &mdmlab.MDMConfigProfileAuthz{TeamID: profileTeamID}, mdmlab.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "authorizing profile team")
	}

	status, err := svc.ds.GetHostMDMProfileInstallStatus(ctx, host.UUID, profileUUID)
	if err != nil {
		if mdmlab.IsNotFound(err) {
			return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", "Unable to match profile to host.").WithStatus(http.StatusNotFound), "getting host mdm profile status")
		}
		return ctxerr.Wrap(ctx, err, "getting host mdm profile status")
	}
	if status == mdmlab.MDMDeliveryPending || status == mdmlab.MDMDeliveryVerifying {
		return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("HostMDMProfile", "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.").WithStatus(http.StatusConflict), "check profile status")
	}
	if status != mdmlab.MDMDeliveryFailed && status != mdmlab.MDMDeliveryVerified {
		// this should never happen, but just in case
		return ctxerr.Errorf(ctx, "unrecognized profile status %s", status)
	}

	if err := svc.ds.ResendHostMDMProfile(ctx, host.UUID, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "resending host mdm profile")
	}

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &mdmlab.ActivityTypeResentConfigurationProfile{
			HostID:          &host.ID,
			HostDisplayName: ptr.String(host.DisplayName()),
			ProfileName:     profileName,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for resend config profile")
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple/request_csr
////////////////////////////////////////////////////////////////////////////////

// Used for overriding the env var value in testing
var testSetEmptyPrivateKey bool

type getMDMAppleCSRRequest struct{}

type getMDMAppleCSRResponse struct {
	CSR []byte `json:"csr"` // base64 encoded
	Err error  `json:"error,omitempty"`
}

func (r getMDMAppleCSRResponse) error() error { return r.Err }

func getMDMAppleCSREndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	signedCSRB64, err := svc.GetMDMAppleCSR(ctx)
	if err != nil {
		return &getMDMAppleCSRResponse{Err: err}, nil
	}

	return &getMDMAppleCSRResponse{CSR: signedCSRB64}, nil
}

func (svc *Service) GetMDMAppleCSR(ctx context.Context) ([]byte, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.New(ctx, "Couldn't download signed CSR. Missing required private key. Learn how to configure the private key here: https://mdmlabdm.com/learn-more-about/mdmlab-server-private-key")
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, mdmlab.ErrNoContext
	}

	// Check if we have existing certs and keys
	var apnsKey crypto.PrivateKey
	savedAssets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []mdmlab.MDMAssetName{
		mdmlab.MDMAssetCACert,
		mdmlab.MDMAssetCAKey,
		mdmlab.MDMAssetAPNSKey,
	}, nil)
	if err != nil {
		// allow not found errors as it means we're generating the assets for
		// the first time.
		if !mdmlab.IsNotFound(err) {
			return nil, ctxerr.Wrap(ctx, err, "loading existing assets from the database")
		}
	}

	if len(savedAssets) == 0 {
		// Then we should create them

		scepCert, scepKey, err := apple_mdm.NewSCEPCACertKey()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate SCEP cert and key")
		}

		apnsRSAKey, err := apple_mdm.NewPrivateKey()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate new apns private key")
		}
		apnsKey = apnsRSAKey

		// Store our config assets encrypted
		var assets []mdmlab.MDMConfigAsset
		for k, v := range map[mdmlab.MDMAssetName][]byte{
			mdmlab.MDMAssetCACert:  apple_mdm.EncodeCertPEM(scepCert),
			mdmlab.MDMAssetCAKey:   apple_mdm.EncodePrivateKeyPEM(scepKey),
			mdmlab.MDMAssetAPNSKey: apple_mdm.EncodePrivateKeyPEM(apnsRSAKey),
		} {
			assets = append(assets, mdmlab.MDMConfigAsset{
				Name:  k,
				Value: v,
			})
		}

		if err := svc.ds.InsertMDMConfigAssets(ctx, assets, nil); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "inserting mdm config assets")
		}
	} else {
		rawApnsKey := savedAssets[mdmlab.MDMAssetAPNSKey]
		apnsKey, err = cryptoutil.ParsePrivateKey(rawApnsKey.Value, "APNS private key")
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "parse APNS private key")
		}
	}

	// Generate new APNS CSR every time this is called
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	apnsCSR, err := apple_mdm.GenerateAPNSCSR(appConfig.OrgInfo.OrgName, vc.Email(), apnsKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate APNS cert and key")
	}

	// Submit CSR to mdmlabdm.com for signing
	websiteClient := mdmlabhttp.NewClient(mdmlabhttp.WithTimeout(10 * time.Second))

	signedCSRB64, err := apple_mdm.GetSignedAPNSCSRNoEmail(websiteClient, apnsCSR)
	if err != nil {
		var fwe apple_mdm.MDMlabWebsiteError
		if errors.As(err, &fwe) {
			// From svc.RequestMDMAppleCSR: mdmlabdm.com returns a bad request here if the email is invalid.
			if fwe.Status >= 400 && fwe.Status <= 499 {
				return nil, ctxerr.Wrap(
					ctx,
					mdmlab.NewInvalidArgumentError(
						"email_address",
						fmt.Sprintf("this email address is not valid: %v", err),
					),
				)
			}
			return nil, ctxerr.Wrap(
				ctx,
				mdmlab.NewUserMessageError(
					fmt.Errorf("MDMlabDM CSR request failed: %w", err),
					http.StatusBadGateway,
				),
			)
		}
		return nil, ctxerr.Wrap(ctx, err, "get signed CSR")
	}

	// Return signed CSR
	return signedCSRB64, nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/apple/apns_certificate
////////////////////////////////////////////////////////////////////////////////

type uploadMDMAppleAPNSCertRequest struct {
	File *multipart.FileHeader
}

func (uploadMDMAppleAPNSCertRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadMDMAppleAPNSCertRequest{}
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &mdmlab.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["certificate"] == nil || len(r.MultipartForm.File["certificate"]) == 0 {
		return nil, &mdmlab.BadRequestError{
			Message:     "certificate multipart field is required",
			InternalErr: err,
		}
	}

	decoded.File = r.MultipartForm.File["certificate"][0]

	return &decoded, nil
}

type uploadMDMAppleAPNSCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r uploadMDMAppleAPNSCertResponse) error() error {
	return r.Err
}

func (r uploadMDMAppleAPNSCertResponse) Status() int { return http.StatusAccepted }

func uploadMDMAppleAPNSCertEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*uploadMDMAppleAPNSCertRequest)
	file, err := req.File.Open()
	if err != nil {
		return uploadMDMAppleAPNSCertResponse{Err: err}, nil
	}
	defer file.Close()

	if err := svc.UploadMDMAppleAPNSCert(ctx, file); err != nil {
		return &uploadMDMAppleAPNSCertResponse{Err: err}, nil
	}

	return &uploadMDMAppleAPNSCertResponse{}, nil
}

func (svc *Service) UploadMDMAppleAPNSCert(ctx context.Context, cert io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return ctxerr.New(ctx, "Couldn't upload APNs certificate. Missing required private key. Learn how to configure the private key here: https://mdmlabdm.com/learn-more-about/mdmlab-server-private-key")
	}

	if cert == nil {
		return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("certificate", "Invalid certificate. Please provide a valid certificate from Apple Push Certificate Portal."))
	}

	// Get cert file bytes
	certBytes, err := io.ReadAll(cert)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading apns certificate")
	}

	// Validate cert
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("certificate", "Invalid certificate. Please provide a valid certificate from Apple Push Certificate Portal."))
	}

	if err := svc.authz.Authorize(ctx, &mdmlab.AppleMDM{}, mdmlab.ActionRead); err != nil {
		return err
	}

	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []mdmlab.MDMAssetName{mdmlab.MDMAssetAPNSKey}, nil)
	if err != nil {
		if mdmlab.IsNotFound(err) {
			return ctxerr.Wrap(ctx, &mdmlab.BadRequestError{
				Message: "Please generate a private key first.",
			}, "uploading APNs certificate")
		}

		return ctxerr.Wrap(ctx, err, "retrieving APNs key")
	}

	_, err = tls.X509KeyPair(certBytes, assets[mdmlab.MDMAssetAPNSKey].Value)
	if err != nil {
		return ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("certificate", "Invalid certificate. Please provide a valid certificate from Apple Push Certificate Portal."))
	}

	// delete the old certificate and insert the new one
	// TODO(roberto): replacing the certificate should be done in a single transaction in the DB
	err = svc.ds.DeleteMDMConfigAssetsByName(ctx, []mdmlab.MDMAssetName{mdmlab.MDMAssetAPNSCert})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting old apns cert from db")
	}
	err = svc.ds.InsertMDMConfigAssets(ctx, []mdmlab.MDMConfigAsset{
		{Name: mdmlab.MDMAssetAPNSCert, Value: certBytes},
	}, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "writing apns cert to db")
	}

	// flip the app config flag
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	wasEnabledAndConfigured := appCfg.MDM.EnabledAndConfigured
	appCfg.MDM.EnabledAndConfigured = true
	err = svc.ds.SaveAppConfig(ctx, appCfg)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "saving app config")
	}

	// Disk encryption can be enabled prior to Apple MDM being configured, but we need MDM to be set up to escrow
	// FileVault keys. We handle the other order of operations elsewhere (on encryption enable, after checking to see
	// if Mac MDM is already enabled). We skip this step if we were just re-uploading an APNs cert when MDM was already
	// enabled.
	if wasEnabledAndConfigured {
		return nil
	}

	// Enable FileVault escrow if no-team already has disk encryption enforced
	if appCfg.MDM.EnableDiskEncryption.Value {
		if err := svc.EnterpriseOverrides.MDMAppleEnableFileVaultAndEscrow(ctx, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "enable no-team FileVault escrow")
		}
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), mdmlab.ActivityTypeEnabledMacosDiskEncryption{}); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for enabling no-team macOS disk encryption")
		}
	}
	// Enable FileVault escrow for teams that already have disk encryption enforced
	// For later: add a data store method to avoid making an extra query per team to check whether encryption is enforced
	teams, err := svc.ds.TeamsSummary(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing teams")
	}
	for _, team := range teams {
		isEncryptionEnforced, err := svc.ds.GetConfigEnableDiskEncryption(ctx, &team.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving encryption enforcement status for team")
		}
		if isEncryptionEnforced {
			if err := svc.EnterpriseOverrides.MDMAppleEnableFileVaultAndEscrow(ctx, &team.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "enable FileVault escrow for team")
			}
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), mdmlab.ActivityTypeEnabledMacosDiskEncryption{TeamID: &team.ID, TeamName: &team.Name}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for enabling macOS disk encryption for team")
			}
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/apple/apns_certificate
////////////////////////////////////////////////////////////////////////////////

type deleteMDMAppleAPNSCertRequest struct{}

type deleteMDMAppleAPNSCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMAppleAPNSCertResponse) error() error {
	return r.Err
}

func deleteMDMAppleAPNSCertEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	if err := svc.DeleteMDMAppleAPNSCert(ctx); err != nil {
		return &deleteMDMAppleAPNSCertResponse{Err: err}, nil
	}

	return &deleteMDMAppleAPNSCertResponse{}, nil
}

func (svc *Service) DeleteMDMAppleAPNSCert(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return err
	}

	err := svc.ds.DeleteMDMConfigAssetsByName(ctx, []mdmlab.MDMAssetName{
		mdmlab.MDMAssetAPNSCert,
		mdmlab.MDMAssetAPNSKey,
		mdmlab.MDMAssetCACert,
		mdmlab.MDMAssetCAKey,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting apple mdm assets")
	}

	// flip the app config flag
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	appCfg.MDM.EnabledAndConfigured = false

	return svc.ds.SaveAppConfig(ctx, appCfg)
}
