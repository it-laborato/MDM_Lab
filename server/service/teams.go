package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/text/unicode/norm"

	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

////////////////////////////////////////////////////////////////////////////////
// List Teams
////////////////////////////////////////////////////////////////////////////////

type listTeamsRequest struct {
	ListOptions mdmlab.ListOptions `url:"list_options"`
}

type listTeamsResponse struct {
	Teams []mdmlab.Team `json:"teams"`
	Err   error        `json:"error,omitempty"`
}

func (r listTeamsResponse) error() error { return r.Err }

func listTeamsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*listTeamsRequest)
	teams, err := svc.ListTeams(ctx, req.ListOptions)
	if err != nil {
		return listTeamsResponse{Err: err}, nil
	}

	resp := listTeamsResponse{Teams: []mdmlab.Team{}}
	for _, team := range teams {
		resp.Teams = append(resp.Teams, *team)
	}
	return resp, nil
}

func (svc *Service) ListTeams(ctx context.Context, opt mdmlab.ListOptions) ([]*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get Team
////////////////////////////////////////////////////////////////////////////////

type getTeamRequest struct {
	ID uint `url:"id"`
}

type getTeamResponse struct {
	Team *mdmlab.Team `json:"team"`
	Err  error       `json:"error,omitempty"`
}

func (r getTeamResponse) error() error { return r.Err }

func getTeamEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*getTeamRequest)
	team, err := svc.GetTeam(ctx, req.ID)
	if err != nil {
		return getTeamResponse{Err: err}, nil
	}
	return getTeamResponse{Team: team}, nil
}

func (svc *Service) GetTeam(ctx context.Context, tid uint) (*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Create Team
////////////////////////////////////////////////////////////////////////////////

type createTeamRequest struct {
	mdmlab.TeamPayload
}

type teamResponse struct {
	Team *mdmlab.Team `json:"team,omitempty"`
	Err  error       `json:"error,omitempty"`
}

func (r teamResponse) error() error { return r.Err }

func createTeamEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*createTeamRequest)

	team, err := svc.NewTeam(ctx, req.TeamPayload)
	if err != nil {
		return teamResponse{Err: err}, nil
	}
	return teamResponse{Team: team}, nil
}

func (svc *Service) NewTeam(ctx context.Context, p mdmlab.TeamPayload) (*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Modify Team
////////////////////////////////////////////////////////////////////////////////

type modifyTeamRequest struct {
	ID uint `json:"-" url:"id"`
	mdmlab.TeamPayload
}

func modifyTeamEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*modifyTeamRequest)
	team, err := svc.ModifyTeam(ctx, req.ID, req.TeamPayload)
	if err != nil {
		return teamResponse{Err: err}, nil
	}
	return teamResponse{Team: team}, err
}

func (svc *Service) ModifyTeam(ctx context.Context, id uint, payload mdmlab.TeamPayload) (*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Delete Team
////////////////////////////////////////////////////////////////////////////////

type deleteTeamRequest struct {
	ID uint `url:"id"`
}

type deleteTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteTeamResponse) error() error { return r.Err }

func deleteTeamEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*deleteTeamRequest)
	err := svc.DeleteTeam(ctx, req.ID)
	if err != nil {
		return deleteTeamResponse{Err: err}, nil
	}
	return deleteTeamResponse{}, nil
}

func (svc *Service) DeleteTeam(ctx context.Context, tid uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Apply Team Specs
////////////////////////////////////////////////////////////////////////////////

type applyTeamSpecsRequest struct {
	Force             bool                              `json:"-" query:"force,optional"`   // if true, bypass strict incoming json validation
	DryRun            bool                              `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	DryRunAssumptions *mdmlab.TeamSpecsDryRunAssumptions `json:"dry_run_assumptions,omitempty"`
	Specs             []*mdmlab.TeamSpec                 `json:"specs"`
}

func (req *applyTeamSpecsRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	if err := mdmlab.JSONStrictDecode(r, req); err != nil {
		err = mdmlab.NewUserMessageError(err, http.StatusBadRequest)
		if !req.Force || !mdmlab.IsJSONUnknownFieldError(err) {
			// only unknown field errors can be forced at this point (other errors
			// can be forced later, after agent options' validations)
			return ctxerr.Wrap(ctx, err, "strict decode team specs")
		}
	}

	// the MacOSSettings field must be validated separately, since it
	// JSON-decodes into a free-form map.
	for _, spec := range req.Specs {
		if spec == nil || spec.MDM.MacOSSettings == nil {
			continue
		}

		var macOSSettings mdmlab.MacOSSettings
		validMap := macOSSettings.ToMap()

		// the keys provided must be valid
		for k := range spec.MDM.MacOSSettings {
			if _, ok := validMap[k]; !ok {
				return ctxerr.Wrap(ctx, mdmlab.NewUserMessageError(
					fmt.Errorf("json: unknown field %q", k),
					http.StatusBadRequest), "strict decode team specs")
			}
		}
	}
	return nil
}

type applyTeamSpecsResponse struct {
	Err           error           `json:"error,omitempty"`
	TeamIDsByName map[string]uint `json:"team_ids_by_name,omitempty"`
}

func (r applyTeamSpecsResponse) error() error { return r.Err }

func applyTeamSpecsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*applyTeamSpecsRequest)
	if !req.DryRun {
		req.DryRunAssumptions = nil
	}

	// remove any nil spec (may happen in conversion from YAML to JSON with mdmlabctl, but also
	// with the API should someone send such JSON)
	actualSpecs := make([]*mdmlab.TeamSpec, 0, len(req.Specs))
	for _, spec := range req.Specs {
		if spec != nil {
			// Normalize the team name for full Unicode support to prevent potential issue further in the spec flow
			spec.Name = norm.NFC.String(spec.Name)
			actualSpecs = append(actualSpecs, spec)
		}
	}

	idsByName, err := svc.ApplyTeamSpecs(
		ctx, actualSpecs, mdmlab.ApplyTeamSpecOptions{
			ApplySpecOptions: mdmlab.ApplySpecOptions{
				Force:  req.Force,
				DryRun: req.DryRun,
			},
			DryRunAssumptions: req.DryRunAssumptions,
		})
	if err != nil {
		return applyTeamSpecsResponse{Err: err}, nil
	}
	return applyTeamSpecsResponse{TeamIDsByName: idsByName}, nil
}

func (svc Service) ApplyTeamSpecs(ctx context.Context, _ []*mdmlab.TeamSpec, _ mdmlab.ApplyTeamSpecOptions) (map[string]uint, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Modify Team Agent Options
////////////////////////////////////////////////////////////////////////////////

type modifyTeamAgentOptionsRequest struct {
	ID     uint `json:"-" url:"id"`
	Force  bool `json:"-" query:"force,optional"`   // if true, bypass strict incoming json validation
	DryRun bool `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	json.RawMessage
}

func modifyTeamAgentOptionsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*modifyTeamAgentOptionsRequest)
	team, err := svc.ModifyTeamAgentOptions(ctx, req.ID, req.RawMessage, mdmlab.ApplySpecOptions{
		Force:  req.Force,
		DryRun: req.DryRun,
	})
	if err != nil {
		return teamResponse{Err: err}, nil
	}
	return teamResponse{Team: team}, err
}

func (svc *Service) ModifyTeamAgentOptions(ctx context.Context, id uint, teamOptions json.RawMessage, applyOptions mdmlab.ApplySpecOptions) (*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// List Team Users
////////////////////////////////////////////////////////////////////////////////

type listTeamUsersRequest struct {
	TeamID      uint              `url:"id"`
	ListOptions mdmlab.ListOptions `url:"list_options"`
}

func listTeamUsersEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*listTeamUsersRequest)
	users, err := svc.ListTeamUsers(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return listUsersResponse{Err: err}, nil
	}

	resp := listUsersResponse{Users: []mdmlab.User{}}
	for _, user := range users {
		resp.Users = append(resp.Users, *user)
	}
	return resp, nil
}

func (svc *Service) ListTeamUsers(ctx context.Context, teamID uint, opt mdmlab.ListOptions) ([]*mdmlab.User, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Add / Delete Team Users
////////////////////////////////////////////////////////////////////////////////

// same request struct for add and delete
type modifyTeamUsersRequest struct {
	TeamID uint `json:"-" url:"id"`
	// User ID and role must be specified for add users, user ID must be
	// specified for delete users.
	Users []mdmlab.TeamUser `json:"users"`
}

func addTeamUsersEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*modifyTeamUsersRequest)
	team, err := svc.AddTeamUsers(ctx, req.TeamID, req.Users)
	if err != nil {
		return teamResponse{Err: err}, nil
	}
	return teamResponse{Team: team}, err
}

func (svc *Service) AddTeamUsers(ctx context.Context, teamID uint, users []mdmlab.TeamUser) (*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

func deleteTeamUsersEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*modifyTeamUsersRequest)
	team, err := svc.DeleteTeamUsers(ctx, req.TeamID, req.Users)
	if err != nil {
		return teamResponse{Err: err}, nil
	}
	return teamResponse{Team: team}, err
}

func (svc *Service) DeleteTeamUsers(ctx context.Context, teamID uint, users []mdmlab.TeamUser) (*mdmlab.Team, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get enroll secrets for team
////////////////////////////////////////////////////////////////////////////////

type teamEnrollSecretsRequest struct {
	TeamID uint `url:"id"`
}

type teamEnrollSecretsResponse struct {
	Secrets []*mdmlab.EnrollSecret `json:"secrets"`
	Err     error                 `json:"error,omitempty"`
}

func (r teamEnrollSecretsResponse) error() error { return r.Err }

func teamEnrollSecretsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*teamEnrollSecretsRequest)
	secrets, err := svc.TeamEnrollSecrets(ctx, req.TeamID)
	if err != nil {
		return teamEnrollSecretsResponse{Err: err}, nil
	}

	return teamEnrollSecretsResponse{Secrets: secrets}, err
}

func (svc *Service) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*mdmlab.EnrollSecret, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Modify enroll secrets for team
////////////////////////////////////////////////////////////////////////////////

type modifyTeamEnrollSecretsRequest struct {
	TeamID  uint                 `url:"team_id"`
	Secrets []mdmlab.EnrollSecret `json:"secrets"`
}

func modifyTeamEnrollSecretsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*modifyTeamEnrollSecretsRequest)
	secrets, err := svc.ModifyTeamEnrollSecrets(ctx, req.TeamID, req.Secrets)
	if err != nil {
		return teamEnrollSecretsResponse{Err: err}, nil
	}

	return teamEnrollSecretsResponse{Secrets: secrets}, err
}

func (svc *Service) ModifyTeamEnrollSecrets(ctx context.Context, teamID uint, secrets []mdmlab.EnrollSecret) ([]*mdmlab.EnrollSecret, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, mdmlab.ErrMissingLicense
}
