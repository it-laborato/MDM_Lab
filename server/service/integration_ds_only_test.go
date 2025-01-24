package service

import (
	"net/http"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type integrationDSTestSuite struct {
	withDS
	suite.Suite
}

func TestIntegrationsDSTestSuite(t *testing.T) {
	testingSuite := new(integrationDSTestSuite)
	testingSuite.withDS.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationDSTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationDSTestSuite")
}

func (s *integrationDSTestSuite) TestLicenseExpiration() {
	testCases := []struct {
		name             string
		tier             string
		expiration       time.Time
		shouldHaveHeader bool
	}{
		{"basic expired", mdmlab.TierPremium, time.Now().Add(-24 * time.Hour), true},
		{"basic not expired", mdmlab.TierPremium, time.Now().Add(24 * time.Hour), false},
		{"core expired", mdmlab.TierFree, time.Now().Add(-24 * time.Hour), false},
		{"core not expired", mdmlab.TierFree, time.Now().Add(24 * time.Hour), false},
	}

	createTestUsers(s.T(), s.ds)
	for _, tt := range testCases {
		s.Run(tt.name, func() {
			t := s.T()

			license := &mdmlab.LicenseInfo{Tier: tt.tier, Expiration: tt.expiration}
			_, server := RunServerForTestsWithDS(t, s.ds, &TestServerOpts{License: license, SkipCreateTestUsers: true})

			ts := withServer{server: server}
			ts.s = &s.Suite
			ts.token = ts.getTestAdminToken()

			resp := ts.Do("GET", "/api/latest/mdmlab/config", nil, http.StatusOK)
			if tt.shouldHaveHeader {
				require.Equal(t, mdmlab.HeaderLicenseValueExpired, resp.Header.Get(mdmlab.HeaderLicenseKey))
			} else {
				require.Equal(t, "", resp.Header.Get(mdmlab.HeaderLicenseKey))
			}
		})
	}
}
