package service

import (
	"testing"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestObfuscateSecrets(t *testing.T) {
	buildTeams := func(n int) []*mdmlab.Team {
		r := make([]*mdmlab.Team, 0, n)
		for i := 1; i <= n; i++ {
			r = append(r, &mdmlab.Team{
				ID: uint(i), //nolint:gosec // dismiss G115
				Secrets: []*mdmlab.EnrollSecret{
					{Secret: "abc"},
					{Secret: "123"},
				},
			})
		}
		return r
	}

	t.Run("no user", func(t *testing.T) {
		err := obfuscateSecrets(nil, nil)
		require.Error(t, err)
	})

	t.Run("no teams", func(t *testing.T) {
		user := mdmlab.User{}
		err := obfuscateSecrets(&user, nil)
		require.NoError(t, err)
	})

	t.Run("user is not a global observer", func(t *testing.T) {
		user := mdmlab.User{GlobalRole: ptr.String(mdmlab.RoleAdmin)}
		teams := buildTeams(3)

		err := obfuscateSecrets(&user, teams)
		require.NoError(t, err)

		for _, team := range teams {
			for _, s := range team.Secrets {
				require.NotEqual(t, mdmlab.MaskedPassword, s.Secret)
			}
		}
	})

	t.Run("user is global observer", func(t *testing.T) {
		roles := []*string{ptr.String(mdmlab.RoleObserver), ptr.String(mdmlab.RoleObserverPlus)}
		for _, r := range roles {
			user := &mdmlab.User{GlobalRole: r}
			teams := buildTeams(3)

			err := obfuscateSecrets(user, teams)
			require.NoError(t, err)

			for _, team := range teams {
				for _, s := range team.Secrets {
					require.Equal(t, mdmlab.MaskedPassword, s.Secret)
				}
			}
		}
	})

	t.Run("user is observer in some teams", func(t *testing.T) {
		teams := buildTeams(4)

		// Make user an observer in the 'even' teams
		user := &mdmlab.User{Teams: []mdmlab.UserTeam{
			{
				Team: *teams[1],
				Role: mdmlab.RoleObserver,
			},
			{
				Team: *teams[2],
				Role: mdmlab.RoleAdmin,
			},
			{
				Team: *teams[3],
				Role: mdmlab.RoleObserverPlus,
			},
		}}

		err := obfuscateSecrets(user, teams)
		require.NoError(t, err)

		for i, team := range teams {
			for _, s := range team.Secrets {
				require.Equal(t, mdmlab.MaskedPassword == s.Secret, i == 0 || i == 1 || i == 3)
			}
		}
	})
}
