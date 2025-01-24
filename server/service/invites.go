package service

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"strings"

	"github.com:it-laborato/MDM_Lab/server"
	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com:it-laborato/MDM_Lab/server/contexts/logging"
	"github.com:it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mail"
)

////////////////////////////////////////////////////////////////////////////////
// Create invite
////////////////////////////////////////////////////////////////////////////////

type createInviteRequest struct {
	mdmlab.InvitePayload
}

type createInviteResponse struct {
	Invite *mdmlab.Invite `json:"invite,omitempty"`
	Err    error         `json:"error,omitempty"`
}

func (r createInviteResponse) error() error { return r.Err }

func createInviteEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*createInviteRequest)
	invite, err := svc.InviteNewUser(ctx, req.InvitePayload)
	if err != nil {
		return createInviteResponse{Err: err}, nil
	}
	return createInviteResponse{invite, nil}, nil
}

var SSOMFAConflict = &mdmlab.ConflictError{Message: "MDMlab MFA is is not applicable to SSO users"}

func (svc *Service) InviteNewUser(ctx context.Context, payload mdmlab.InvitePayload) (*mdmlab.Invite, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Invite{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	if payload.Email == nil {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("email", "missing required argument"))
	}
	*payload.Email = strings.ToLower(*payload.Email)

	// verify that the user with the given email does not already exist
	_, err := svc.ds.UserByEmail(ctx, *payload.Email)
	if err == nil {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("email", "a user with this account already exists"))
	}
	var nfe mdmlab.NotFoundError
	if !errors.As(err, &nfe) {
		return nil, err
	}

	// find the user who created the invite
	v, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, errors.New("missing viewer context for create invite")
	}
	inviter := v.User

	token, err := server.GenerateRandomURLSafeText(svc.config.App.TokenKeySize)
	if err != nil {
		return nil, err
	}

	invite := &mdmlab.Invite{
		Email:      *payload.Email,
		InvitedBy:  inviter.ID,
		Token:      token,
		GlobalRole: payload.GlobalRole,
		Teams:      payload.Teams,
	}
	if payload.Position != nil {
		invite.Position = *payload.Position
	}
	if payload.Name != nil {
		invite.Name = *payload.Name
	}
	if payload.SSOEnabled != nil {
		invite.SSOEnabled = *payload.SSOEnabled
	}
	if payload.MFAEnabled != nil {
		invite.MFAEnabled = *payload.MFAEnabled
	}

	if err = svc.ValidateInvite(ctx, *invite); err != nil {
		return nil, err
	}

	invite, err = svc.ds.NewInvite(ctx, invite)
	if err != nil {
		return nil, err
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	invitedBy := inviter.Name
	if invitedBy == "" {
		invitedBy = inviter.Email
	}
	var smtpSettings mdmlab.SMTPSettings
	if config.SMTPSettings != nil {
		smtpSettings = *config.SMTPSettings
	}
	inviteEmail := mdmlab.Email{
		Subject:      "You have been invited to MDMlab!",
		To:           []string{invite.Email},
		ServerURL:    config.ServerSettings.ServerURL,
		SMTPSettings: smtpSettings,
		Mailer: &mail.InviteMailer{
			Invite:    invite,
			BaseURL:   template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL:  getAssetURL(),
			OrgName:   config.OrgInfo.OrgName,
			InvitedBy: invitedBy,
		},
	}

	err = svc.mailService.SendEmail(inviteEmail)
	if err != nil {
		return nil, err
	}
	return invite, nil
}

func (svc *Service) ValidateInvite(ctx context.Context, invite mdmlab.Invite) error {
	if !invite.MFAEnabled {
		return nil
	}

	lic, err := svc.License(ctx)
	if err != nil {
		return err
	}
	if lic == nil || !lic.IsPremium() {
		return mdmlab.ErrMissingLicense
	}
	if invite.SSOEnabled {
		return SSOMFAConflict
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// List invites
////////////////////////////////////////////////////////////////////////////////

type listInvitesRequest struct {
	ListOptions mdmlab.ListOptions `url:"list_options"`
}

type listInvitesResponse struct {
	Invites []mdmlab.Invite `json:"invites"`
	Err     error          `json:"error,omitempty"`
}

func (r listInvitesResponse) error() error { return r.Err }

func listInvitesEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*listInvitesRequest)
	invites, err := svc.ListInvites(ctx, req.ListOptions)
	if err != nil {
		return listInvitesResponse{Err: err}, nil
	}

	resp := listInvitesResponse{Invites: []mdmlab.Invite{}}
	for _, invite := range invites {
		resp.Invites = append(resp.Invites, *invite)
	}
	return resp, nil
}

func (svc *Service) ListInvites(ctx context.Context, opt mdmlab.ListOptions) ([]*mdmlab.Invite, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Invite{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.ListInvites(ctx, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Update invite
////////////////////////////////////////////////////////////////////////////////

type updateInviteRequest struct {
	ID uint `url:"id"`
	mdmlab.InvitePayload
}

type updateInviteResponse struct {
	Invite *mdmlab.Invite `json:"invite"`
	Err    error         `json:"error,omitempty"`
}

func (r updateInviteResponse) error() error { return r.Err }

func updateInviteEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*updateInviteRequest)
	invite, err := svc.UpdateInvite(ctx, req.ID, req.InvitePayload)
	if err != nil {
		return updateInviteResponse{Err: err}, nil
	}

	return updateInviteResponse{Invite: invite}, nil
}

func (svc *Service) UpdateInvite(ctx context.Context, id uint, payload mdmlab.InvitePayload) (*mdmlab.Invite, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Invite{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	invite, err := svc.ds.Invite(ctx, id)
	if err != nil {
		return nil, err
	}

	if payload.Email != nil && *payload.Email != invite.Email {
		switch _, err := svc.ds.UserByEmail(ctx, *payload.Email); {
		case err == nil:
			return nil, ctxerr.Wrap(ctx, newAlreadyExistsError())
		case errors.Is(err, sql.ErrNoRows):
			// OK
		default:
			return nil, ctxerr.Wrap(ctx, err)
		}

		switch _, err = svc.ds.InviteByEmail(ctx, *payload.Email); {
		case err == nil:
			return nil, ctxerr.Wrap(ctx, newAlreadyExistsError())
		case errors.Is(err, sql.ErrNoRows):
			// OK
		default:
			return nil, ctxerr.Wrap(ctx, err)
		}

		invite.Email = *payload.Email
	}
	if payload.Name != nil {
		invite.Name = *payload.Name
	}
	if payload.Position != nil {
		invite.Position = *payload.Position
	}
	if payload.SSOEnabled != nil {
		invite.SSOEnabled = *payload.SSOEnabled
	}
	if payload.MFAEnabled != nil {
		invite.MFAEnabled = *payload.MFAEnabled
	}

	if err = svc.ValidateInvite(ctx, *invite); err != nil {
		return nil, err
	}

	if payload.GlobalRole.Valid || len(payload.Teams) > 0 {
		if err := mdmlab.ValidateRole(payload.GlobalRole.Ptr(), payload.Teams); err != nil {
			return nil, err
		}
		invite.GlobalRole = payload.GlobalRole
		invite.Teams = payload.Teams
	}

	return svc.ds.UpdateInvite(ctx, id, invite)
}

////////////////////////////////////////////////////////////////////////////////
// Delete invite
////////////////////////////////////////////////////////////////////////////////

type deleteInviteRequest struct {
	ID uint `url:"id"`
}

type deleteInviteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteInviteResponse) error() error { return r.Err }

func deleteInviteEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*deleteInviteRequest)
	err := svc.DeleteInvite(ctx, req.ID)
	if err != nil {
		return deleteInviteResponse{Err: err}, nil
	}
	return deleteInviteResponse{}, nil
}

func (svc *Service) DeleteInvite(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.Invite{}, mdmlab.ActionWrite); err != nil {
		return err
	}
	return svc.ds.DeleteInvite(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Verify invite
////////////////////////////////////////////////////////////////////////////////

type verifyInviteRequest struct {
	Token string `url:"token"`
}

type verifyInviteResponse struct {
	Invite *mdmlab.Invite `json:"invite"`
	Err    error         `json:"error,omitempty"`
}

func (r verifyInviteResponse) error() error { return r.Err }

func verifyInviteEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*verifyInviteRequest)
	invite, err := svc.VerifyInvite(ctx, req.Token)
	if err != nil {
		return verifyInviteResponse{Err: err}, nil
	}
	return verifyInviteResponse{Invite: invite}, nil
}

func (svc *Service) VerifyInvite(ctx context.Context, token string) (*mdmlab.Invite, error) {
	// skipauth: There is no viewer context at this point. We rely on verifying
	// the invite for authNZ.
	svc.authz.SkipAuthorization(ctx)

	logging.WithExtras(ctx, "token", token)

	invite, err := svc.ds.InviteByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if invite.Token != token {
		return nil, mdmlab.NewInvalidArgumentError("invite_token", "Invite Token does not match Email Address.")
	}

	expiresAt := invite.CreatedAt.Add(svc.config.App.InviteTokenValidityPeriod)
	if svc.clock.Now().After(expiresAt) {
		return nil, mdmlab.NewInvalidArgumentError("invite_token", "Invite token has expired.")
	}

	return invite, nil
}
