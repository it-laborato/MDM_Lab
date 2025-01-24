package test

import (
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/ptr"
)

var (
	GoodPassword  = "password123#"
	GoodPassword2 = "password123!"
	UserNoRoles   = &mdmlab.User{
		ID: 1,
	}
	UserAdmin = &mdmlab.User{
		ID:         2,
		GlobalRole: ptr.String(mdmlab.RoleAdmin),
		Email:      "useradmin@example.com",
	}
	UserMaintainer = &mdmlab.User{
		ID:         3,
		GlobalRole: ptr.String(mdmlab.RoleMaintainer),
	}
	UserObserver = &mdmlab.User{
		ID:         4,
		GlobalRole: ptr.String(mdmlab.RoleObserver),
	}
	UserTeamAdminTeam1 = &mdmlab.User{
		ID: 5,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 1},
				Role: mdmlab.RoleAdmin,
			},
		},
	}
	UserTeamAdminTeam2 = &mdmlab.User{
		ID: 6,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 2},
				Role: mdmlab.RoleAdmin,
			},
		},
	}
	UserTeamMaintainerTeam1 = &mdmlab.User{
		ID: 7,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 1},
				Role: mdmlab.RoleMaintainer,
			},
		},
	}
	UserTeamMaintainerTeam2 = &mdmlab.User{
		ID: 8,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 2},
				Role: mdmlab.RoleMaintainer,
			},
		},
	}
	UserTeamObserverTeam1 = &mdmlab.User{
		ID: 9,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 1},
				Role: mdmlab.RoleObserver,
			},
		},
	}
	UserTeamObserverTeam2 = &mdmlab.User{
		ID: 10,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 2},
				Role: mdmlab.RoleObserver,
			},
		},
	}
	UserTeamObserverTeam1TeamAdminTeam2 = &mdmlab.User{
		ID: 11,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 1},
				Role: mdmlab.RoleObserver,
			},
			{
				Team: mdmlab.Team{ID: 2},
				Role: mdmlab.RoleAdmin,
			},
		},
	}
	UserObserverPlus = &mdmlab.User{
		ID:         12,
		GlobalRole: ptr.String(mdmlab.RoleObserverPlus),
	}
	UserTeamObserverPlusTeam1 = &mdmlab.User{
		ID: 13,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 1},
				Role: mdmlab.RoleObserverPlus,
			},
		},
	}
	UserTeamObserverPlusTeam2 = &mdmlab.User{
		ID: 14,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 2},
				Role: mdmlab.RoleObserverPlus,
			},
		},
	}
	UserGitOps = &mdmlab.User{
		ID:         15,
		GlobalRole: ptr.String(mdmlab.RoleGitOps),
	}
	UserTeamGitOpsTeam1 = &mdmlab.User{
		ID: 16,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 1},
				Role: mdmlab.RoleGitOps,
			},
		},
	}
	UserTeamGitOpsTeam2 = &mdmlab.User{
		ID: 17,
		Teams: []mdmlab.UserTeam{
			{
				Team: mdmlab.Team{ID: 2},
				Role: mdmlab.RoleGitOps,
			},
		},
	}
)
