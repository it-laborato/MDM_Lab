package service

import (
	"context"

	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

func (svc *Service) LinuxHostDiskEncryptionStatus(ctx context.Context, host mdmlab.Host) (mdmlab.HostMDMDiskEncryption, error) {
	if !host.IsLUKSSupported() {
		return mdmlab.HostMDMDiskEncryption{}, nil
	}

	actionRequired := mdmlab.DiskEncryptionActionRequired
	verified := mdmlab.DiskEncryptionVerified
	failed := mdmlab.DiskEncryptionFailed

	key, err := svc.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	if err != nil {
		if mdmlab.IsNotFound(err) {
			return mdmlab.HostMDMDiskEncryption{
				Status: &actionRequired,
			}, nil
		}
		return mdmlab.HostMDMDiskEncryption{}, err
	}

	if key.ClientError != "" {
		return mdmlab.HostMDMDiskEncryption{
			Status: &failed,
			Detail: key.ClientError,
		}, nil
	}

	if key.Base64Encrypted == "" {
		return mdmlab.HostMDMDiskEncryption{
			Status: &actionRequired,
		}, nil
	}

	return mdmlab.HostMDMDiskEncryption{
		Status: &verified,
	}, nil
}

func (svc *Service) GetMDMLinuxProfilesSummary(ctx context.Context, teamId *uint) (summary mdmlab.MDMProfilesSummary, err error) {
	if err = svc.authz.Authorize(ctx, mdmlab.MDMConfigProfileAuthz{TeamID: teamId}, mdmlab.ActionRead); err != nil {
		return summary, ctxerr.Wrap(ctx, err)
	}

	// Linux doesn't have configuration profiles, so if we aren't enforcing disk encryption we have nothing to report
	includeDiskEncryptionStats, err := svc.ds.GetConfigEnableDiskEncryption(ctx, teamId)
	if err != nil {
		return summary, ctxerr.Wrap(ctx, err)
	} else if !includeDiskEncryptionStats {
		return summary, nil
	}

	counts, err := svc.ds.GetLinuxDiskEncryptionSummary(ctx, teamId)
	if err != nil {
		return summary, ctxerr.Wrap(ctx, err)
	}

	return mdmlab.MDMProfilesSummary{
		Verified: counts.Verified,
		Pending:  counts.ActionRequired,
		Failed:   counts.Failed,
	}, nil
}
