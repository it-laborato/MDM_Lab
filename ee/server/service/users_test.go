package service

import (
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestRolesChanged(t *testing.T) {
	for _, tc := range []struct {
		name string

		oldGlobal *string
		oldTeams  []mdmlab.UserTeam
		newGlobal *string
		newTeams  []mdmlab.UserTeam

		expectedRolesChanged bool
	}{
		{
			name:                 "no roles",
			expectedRolesChanged: false,
		},
		{
			name:                 "no-role-to-global-role",
			newGlobal:            ptr.String("admin"),
			expectedRolesChanged: true,
		},
		{
			name:                 "global-role-to-no-role",
			oldGlobal:            ptr.String("admin"),
			expectedRolesChanged: true,
		},
		{
			name:                 "global-role-unchanged",
			oldGlobal:            ptr.String("admin"),
			newGlobal:            ptr.String("admin"),
			expectedRolesChanged: false,
		},
		{
			name:                 "global-role-to-other-role",
			oldGlobal:            ptr.String("admin"),
			newGlobal:            ptr.String("maintainer"),
			expectedRolesChanged: true,
		},
		{
			name:      "global-role-to-team-role",
			oldGlobal: ptr.String("admin"),
			newTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "change-role-in-teams",
			oldTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "maintainer",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			newTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "remove-from-team",
			oldTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "maintainer",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			newTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "no-change-teams",
			oldTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			newTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: false,
		},
		{
			name: "added-to-teams",
			newTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
		{
			name: "removed-from-a-team-and-added-to-another",
			oldTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: mdmlab.Team{ID: 3},
					Role: "observer",
				},
			},
			newTeams: []mdmlab.UserTeam{
				{
					Team: mdmlab.Team{ID: 1},
					Role: "admin",
				},
				{
					Team: mdmlab.Team{ID: 2},
					Role: "maintainer",
				},
			},
			expectedRolesChanged: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedRolesChanged, rolesChanged(tc.oldGlobal, tc.oldTeams, tc.newGlobal, tc.newTeams))
		})
	}
}
