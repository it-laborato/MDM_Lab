package service

import (
	"context"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

// This file exports internal functions and methods only for testing purposes.
// Those are used by mdm_external_test.go which runs the tests as an external
// package to avoid import cycles, and as such needs to be able to call these
// unexported symbols.

func (svc *Service) GetOrCreatePreassignTeam(ctx context.Context, groups []string) (*mdmlab.Team, error) {
	return svc.getOrCreatePreassignTeam(ctx, groups)
}

func TeamNameFromPreassignGroups(groups []string) string {
	return teamNameFromPreassignGroups(groups)
}

type NotFoundError = notFoundError

var (
	TestCert = testCert
	TestKey  = testKey
)
