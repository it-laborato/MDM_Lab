package service

import (
	"context"
	"testing"
	"time"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mock"
	"github.com:it-laborato/MDM_Lab/server/ptr"
	"github.com/stretchr/testify/assert"
)

func TestLinuxHostDiskEncryptionStatus(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	actionRequired := mdmlab.DiskEncryptionActionRequired
	verified := mdmlab.DiskEncryptionVerified
	failed := mdmlab.DiskEncryptionFailed

	testcases := []struct {
		name              string
		host              mdmlab.Host
		keyExists         bool
		clientErrorExists bool
		status            mdmlab.HostMDMDiskEncryption
		notFound          bool
	}{
		{
			name:              "no key",
			host:              mdmlab.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         false,
			clientErrorExists: false,
			status: mdmlab.HostMDMDiskEncryption{
				Status: &actionRequired,
			},
		},
		{
			name:              "key exists",
			host:              mdmlab.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         true,
			clientErrorExists: false,
			status: mdmlab.HostMDMDiskEncryption{
				Status: &verified,
			},
		},
		{
			name:              "key exists && client error",
			host:              mdmlab.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         true,
			clientErrorExists: true,
			status: mdmlab.HostMDMDiskEncryption{
				Status: &failed,
				Detail: "client error",
			},
		},
		{
			name:              "no key && client error",
			host:              mdmlab.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         false,
			clientErrorExists: true,
			status: mdmlab.HostMDMDiskEncryption{
				Status: &failed,
				Detail: "client error",
			},
		},
		{
			name:              "key not found",
			host:              mdmlab.Host{ID: 1, Platform: "ubuntu"},
			keyExists:         false,
			clientErrorExists: false,
			status: mdmlab.HostMDMDiskEncryption{
				Status: &actionRequired,
			},
			notFound: true,
		},
		{
			name:   "unsupported platform",
			host:   mdmlab.Host{ID: 1, Platform: "amzn"},
			status: mdmlab.HostMDMDiskEncryption{},
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			ds.GetHostDiskEncryptionKeyFunc = func(ctx context.Context, hostID uint) (*mdmlab.HostDiskEncryptionKey, error) {
				var encrypted string
				if tt.keyExists {
					encrypted = "encrypted"
				}

				var clientError string
				if tt.clientErrorExists {
					clientError = "client error"
				}

				var nfe notFoundError
				if tt.notFound {
					return nil, &nfe
				}

				return &mdmlab.HostDiskEncryptionKey{
					HostID:          hostID,
					Base64Encrypted: encrypted,
					Decryptable:     ptr.Bool(true),
					UpdatedAt:       time.Now(),
					ClientError:     clientError,
				}, nil
			}

			status, err := svc.LinuxHostDiskEncryptionStatus(ctx, tt.host)
			assert.Nil(t, err)

			assert.Equal(t, tt.status, status)
		})
	}
}
