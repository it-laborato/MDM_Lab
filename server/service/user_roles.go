package service

import (
	"context"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"gopkg.in/guregu/null.v3"
)

type applyUserRoleSpecsRequest struct {
	Spec *mdmlab.UsersRoleSpec `json:"spec"`
}

type applyUserRoleSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyUserRoleSpecsResponse) error() error { return r.Err }

func applyUserRoleSpecsEndpoint(ctx context.Context, request interface{}, svc mdmlab.Service) (errorer, error) {
	req := request.(*applyUserRoleSpecsRequest)
	err := svc.ApplyUserRolesSpecs(ctx, *req.Spec)
	if err != nil {
		return applyUserRoleSpecsResponse{Err: err}, nil
	}
	return applyUserRoleSpecsResponse{}, nil
}

func (svc *Service) ApplyUserRolesSpecs(ctx context.Context, specs mdmlab.UsersRoleSpec) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.User{}, mdmlab.ActionWrite); err != nil {
		return err
	}

	var users []*mdmlab.User
	for email, spec := range specs.Roles {
		user, err := svc.ds.UserByEmail(ctx, email)
		if err != nil {
			return err
		}
		// If an admin is downgraded, make sure there is at least one other admin
		err = svc.checkAtLeastOneAdmin(ctx, user, spec, email)
		if err != nil {
			return err
		}
		user.GlobalRole = spec.GlobalRole
		var teams []mdmlab.UserTeam
		for _, team := range spec.Teams {
			t, err := svc.ds.TeamByName(ctx, team.Name)
			if err != nil {
				if mdmlab.IsNotFound(err) {
					return &mdmlab.BadRequestError{
						Message:     err.Error(),
						InternalErr: err,
					}
				}
				return err
			}
			teams = append(teams, mdmlab.UserTeam{
				Team: *t,
				Role: team.Role,
			})
		}
		user.Teams = teams
		users = append(users, user)
	}

	return svc.ds.SaveUsers(ctx, users)
}

func (svc *Service) checkAtLeastOneAdmin(ctx context.Context, user *mdmlab.User, spec *mdmlab.UserRoleSpec, email string) error {
	if null.StringFromPtr(user.GlobalRole).ValueOrZero() == mdmlab.RoleAdmin &&
		null.StringFromPtr(spec.GlobalRole).ValueOrZero() != mdmlab.RoleAdmin {
		users, err := svc.ds.ListUsers(ctx, mdmlab.UserListOptions{})
		if err != nil {
			return err
		}
		adminsExceptCurrent := 0
		for _, u := range users {
			if u.Email == email {
				continue
			}
			if null.StringFromPtr(u.GlobalRole).ValueOrZero() == mdmlab.RoleAdmin {
				adminsExceptCurrent++
			}
		}
		if adminsExceptCurrent == 0 {
			return mdmlab.NewError(mdmlab.ErrNoOneAdminNeeded, "You need at least one admin")
		}
	}
	return nil
}
