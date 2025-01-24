package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/datastore/redis/redistest"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	appleMdm "github.com/it-laborato/MDM_Lab/server/mdm/apple"
	"github.com/it-laborato/MDM_Lab/server/mdm/nanodep/tokenpki"
	"github.com/it-laborato/MDM_Lab/server/ptr"
	"github.com/it-laborato/MDM_Lab/server/service"
	"github.com/it-laborato/MDM_Lab/server/test"
	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsEnterpriseGitops(t *testing.T) {
	testingSuite := new(enterpriseIntegrationGitopsTestSuite)
	testingSuite.suite = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type enterpriseIntegrationGitopsTestSuite struct {
	suite.Suite
	withServer
	mdmlabCfg config.MDMlabConfig
}

func (s *enterpriseIntegrationGitopsTestSuite) SetupSuite() {
	s.withDS.SetupSuite("enterpriseIntegrationGitopsTestSuite")

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	testCert, testKey, err := appleMdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	mdmlabCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &mdmlabCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")
	mdmlabCfg.Osquery.EnrollCooldown = 0

	err = s.ds.InsertMDMConfigAssets(context.Background(), []mdmlab.MDMConfigAsset{
		{Name: mdmlab.MDMAssetAPNSCert, Value: testCertPEM},
		{Name: mdmlab.MDMAssetAPNSKey, Value: testKeyPEM},
		{Name: mdmlab.MDMAssetCACert, Value: testCertPEM},
		{Name: mdmlab.MDMAssetCAKey, Value: testKeyPEM},
	}, nil)
	require.NoError(s.T(), err)

	mdmStorage, err := s.ds.NewMDMAppleMDMStorage()
	require.NoError(s.T(), err)
	depStorage, err := s.ds.NewMDMAppleDEPStorage()
	require.NoError(s.T(), err)
	scepStorage, err := s.ds.NewSCEPDepot()
	require.NoError(s.T(), err)
	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)

	serverConfig := service.TestServerOpts{
		License: &mdmlab.LicenseInfo{
			Tier: mdmlab.TierPremium,
		},
		MDMlabConfig: &mdmlabCfg,
		MDMStorage:  mdmStorage,
		DEPStorage:  depStorage,
		SCEPStorage: scepStorage,
		Pool:        redisPool,
		APNSTopic:   "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
	}
	err = s.ds.InsertMDMConfigAssets(context.Background(), []mdmlab.MDMConfigAsset{
		{Name: mdmlab.MDMAssetSCEPChallenge, Value: []byte("scepchallenge")},
	}, nil)
	require.NoError(s.T(), err)
	users, server := service.RunServerForTestsWithDS(s.T(), s.ds, &serverConfig)
	s.T().Setenv("FLEET_SERVER_ADDRESS", server.URL) // mdmlabctl always uses this env var in tests
	s.server = server
	s.users = users
	s.mdmlabCfg = mdmlabCfg

	appConf, err = s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

func (s *enterpriseIntegrationGitopsTestSuite) TearDownSuite() {
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
}

// TestMDMlabGitops runs `mdmlabctl gitops` command on configs in https://github.com/mdmlabdm/mdmlab-gitops repo.
// Changes to that repo may cause this test to fail.
func (s *enterpriseIntegrationGitopsTestSuite) TestMDMlabGitops() {
	t := s.T()
	const mdmlabGitopsRepo = "https://github.com/mdmlabdm/mdmlab-gitops"

	// Create GitOps user
	user := mdmlab.User{
		Name:       "GitOps User",
		Email:      "mdmlabctl-gitops@example.com",
		GlobalRole: ptr.String(mdmlab.RoleGitOps),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(context.Background(), &user)
	require.NoError(t, err)

	// Create a temporary mdmlabctl config file
	mdmlabctlConfig, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	token := s.getTestToken(user.Email, test.GoodPassword)
	configStr := fmt.Sprintf(
		`
contexts:
  default:
    address: %s
    tls-skip-verify: true
    token: %s
`, s.server.URL, token,
	)
	_, err = mdmlabctlConfig.WriteString(configStr)
	require.NoError(t, err)

	// Clone git repo
	repoDir := t.TempDir()
	_, err = git.PlainClone(
		repoDir, false, &git.CloneOptions{
			ReferenceName: "main",
			SingleBranch:  true,
			Depth:         1,
			URL:           mdmlabGitopsRepo,
			Progress:      os.Stdout,
		},
	)
	require.NoError(t, err)

	// Set the required environment variables
	t.Setenv("FLEET_URL", s.server.URL)
	t.Setenv("FLEET_GLOBAL_ENROLL_SECRET", "global_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_ENROLL_SECRET", "workstations_enroll_secret")
	t.Setenv("FLEET_WORKSTATIONS_CANARY_ENROLL_SECRET", "workstations_canary_enroll_secret")
	globalFile := path.Join(repoDir, "default.yml")
	teamsDir := path.Join(repoDir, "teams")
	teamFiles, err := os.ReadDir(teamsDir)
	require.NoError(t, err)
	teamFileNames := make([]string, 0, len(teamFiles))
	for _, file := range teamFiles {
		if filepath.Ext(file.Name()) == ".yml" {
			teamFileNames = append(teamFileNames, path.Join(teamsDir, file.Name()))
		}
	}

	// Create a team to be deleted.
	deletedTeamFile, err := os.CreateTemp(t.TempDir(), "*.yml")
	require.NoError(t, err)
	const deletedTeamName = "team_to_be_deleted"

	_, err = deletedTeamFile.WriteString(
		fmt.Sprintf(
			`
controls:
software:
queries:
policies:
agent_options:
name: %s
team_settings:
  secrets: [{"secret":"deleted_team_secret"}]
`, deletedTeamName,
		),
	)
	require.NoError(t, err)

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// Apply the team to be deleted
	_ = runAppForTest(t, []string{"gitops", "--config", mdmlabctlConfig.Name(), "-f", deletedTeamFile.Name()})

	// Dry run
	_ = runAppForTest(t, []string{"gitops", "--config", mdmlabctlConfig.Name(), "-f", globalFile, "--dry-run"})
	for _, fileName := range teamFileNames {
		_ = runAppForTest(t, []string{"gitops", "--config", mdmlabctlConfig.Name(), "-f", fileName, "--dry-run"})
	}

	// Dry run with all the files
	args := []string{"gitops", "--config", mdmlabctlConfig.Name(), "--dry-run", "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = runAppForTest(t, args)

	// Real run with all the files, but don't delete other teams
	args = []string{"gitops", "--config", mdmlabctlConfig.Name(), "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = runAppForTest(t, args)

	// Check that all the teams exist
	teamsJSON := runAppForTest(t, []string{"get", "teams", "--config", mdmlabctlConfig.Name(), "--json"})
	assert.Equal(t, 3, strings.Count(teamsJSON, "team_id"))

	// Real run with all the files, and delete other teams
	args = []string{"gitops", "--config", mdmlabctlConfig.Name(), "--delete-other-teams", "-f", globalFile}
	for _, fileName := range teamFileNames {
		args = append(args, "-f", fileName)
	}
	_ = runAppForTest(t, args)

	// Check that only the right teams exist
	teamsJSON = runAppForTest(t, []string{"get", "teams", "--config", mdmlabctlConfig.Name(), "--json"})
	assert.Equal(t, 2, strings.Count(teamsJSON, "team_id"))
	assert.NotContains(t, teamsJSON, deletedTeamName)

	// Real run with one file at a time
	_ = runAppForTest(t, []string{"gitops", "--config", mdmlabctlConfig.Name(), "-f", globalFile})
	for _, fileName := range teamFileNames {
		_ = runAppForTest(t, []string{"gitops", "--config", mdmlabctlConfig.Name(), "-f", fileName})
	}
}
