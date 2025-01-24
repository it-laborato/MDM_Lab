package service

import (
	"context"
	"errors"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/contexts/viewer"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/mock"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/assert"
)

func TestCreateSecretVariables(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []mdmlab.SecretVariable) error {
		return nil
	}

	t.Run("authorization checks", func(t *testing.T) {
		testCases := []struct {
			name       string
			user       *mdmlab.User
			shouldFail bool
		}{
			{
				name:       "global admin",
				user:       &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)},
				shouldFail: false,
			},
			{
				name:       "global maintainer",
				user:       &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleMaintainer)},
				shouldFail: false,
			},
			{
				name:       "global gitops",
				user:       &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)},
				shouldFail: false,
			},
			{
				name:       "global observer",
				user:       &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserver)},
				shouldFail: true,
			},
			{
				name:       "global observer+",
				user:       &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleObserverPlus)},
				shouldFail: true,
			},
			{
				name:       "team admin",
				user:       &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleAdmin}}},
				shouldFail: true,
			},
			{
				name:       "team maintainer",
				user:       &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleMaintainer}}},
				shouldFail: true,
			},
			{
				name:       "team observer",
				user:       &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserver}}},
				shouldFail: true,
			},
			{
				name:       "team observer+",
				user:       &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleObserverPlus}}},
				shouldFail: true,
			},
			{
				name:       "team gitops",
				user:       &mdmlab.User{Teams: []mdmlab.UserTeam{{Team: mdmlab.Team{ID: 1}, Role: mdmlab.RoleGitOps}}},
				shouldFail: true,
			},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ctx = viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

				err := svc.CreateSecretVariables(ctx, []mdmlab.SecretVariable{{Name: "foo", Value: "bar"}}, false)
				checkAuthErr(t, tt.shouldFail, err)
			})
		}
	})

	t.Run("failure test", func(t *testing.T) {
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleGitOps)}})
		testSetEmptyPrivateKey = true
		t.Cleanup(func() {
			testSetEmptyPrivateKey = false
		})
		err := svc.CreateSecretVariables(ctx, []mdmlab.SecretVariable{{Name: "foo", Value: "bar"}}, true)
		assert.ErrorContains(t, err, "Couldn't save secret variables. Missing required private key")
		testSetEmptyPrivateKey = false

		ds.UpsertSecretVariablesFunc = func(ctx context.Context, secrets []mdmlab.SecretVariable) error {
			return errors.New("test error")
		}
		err = svc.CreateSecretVariables(ctx, []mdmlab.SecretVariable{{Name: "foo", Value: "bar"}}, false)
		assert.ErrorContains(t, err, "test error")
	})

}
