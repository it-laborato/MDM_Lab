package s3

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
)

const (
	desktopPath = "desktop"
	executable  = "mdmlab-osquery"
)

type installerNotFoundError struct{}

var _ mdmlab.NotFoundError = (*installerNotFoundError)(nil)

func (p installerNotFoundError) Error() string {
	return "installer not found"
}

func (p installerNotFoundError) IsNotFound() bool {
	return true
}

// InstallerStore contains methods to retrieve installers from S3
type InstallerStore struct {
	*s3store
}

// NewInstallerStore creates a new instance with the given S3 config
func NewInstallerStore(config config.S3Config) (*InstallerStore, error) {
	s3store, err := newS3store(config.CarvesToInternalCfg())
	if err != nil {
		return nil, err
	}
	return &InstallerStore{s3store}, nil
}

// Get retrieves the requested installer from S3
func (i *InstallerStore) Get(ctx context.Context, installer mdmlab.Installer) (io.ReadCloser, int64, error) {
	key := i.keyForInstaller(installer)
	req, err := i.s3client.GetObject(&s3.GetObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket, "NotFound":
				return nil, int64(0), installerNotFoundError{}
			}
		}

		return nil, int64(0), ctxerr.Wrap(ctx, err, "retrieving installer from store")
	}

	return req.Body, *req.ContentLength, nil
}

// Put uploads an installer to S3
func (i *InstallerStore) Put(ctx context.Context, installer mdmlab.Installer) (string, error) {
	key := i.keyForInstaller(installer)
	_, err := i.s3client.PutObject(&s3.PutObjectInput{
		Bucket: &i.bucket,
		Body:   installer.Content,
		Key:    &key,
	})
	return key, err
}

// Exists checks if an installer exists in the S3 bucket
func (i *InstallerStore) Exists(ctx context.Context, installer mdmlab.Installer) (bool, error) {
	key := i.keyForInstaller(installer)
	_, err := i.s3client.HeadObject(&s3.HeadObjectInput{Bucket: &i.bucket, Key: &key})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey, s3.ErrCodeNoSuchBucket, "NotFound":
				return false, nil
			}
		}

		return false, ctxerr.Wrap(ctx, err, "checking existence on file store")
	}

	return true, nil
}

// keyForInstaller builds an S3 key to search for the installer
func (i *InstallerStore) keyForInstaller(installer mdmlab.Installer) string {
	file := fmt.Sprintf("%s.%s", executable, installer.Kind)
	dir := ""
	if installer.Desktop {
		dir = desktopPath
	}
	return path.Join(i.prefix, installer.EnrollSecret, dir, file)
}
