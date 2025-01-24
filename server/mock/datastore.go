package mock

import (
	"context"

	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

//go:generate go run ./mockimpl/impl.go -o datastore_mock.go "s *DataStore" "mdmlab.Datastore"
//go:generate go run ./mockimpl/impl.go -o datastore_installers.go "s *InstallerStore" "mdmlab.InstallerStore"
//go:generate go run ./mockimpl/impl.go -o nanodep/storage.go "s *Storage" "github.com:it-laborato/MDM_Lab/server/mdm/nanodep/storage.AllDEPStorage"
//go:generate go run ./mockimpl/impl.go -o mdm/datastore_mdm_mock.go "fs *MDMAppleStore" "mdmlab.MDMAppleStore"
//go:generate go run ./mockimpl/impl.go -o scep/depot.go "d *Depot" "depot.Depot"
//go:generate go run ./mockimpl/impl.go -o mdm/bootstrap_package_store.go "s *MDMBootstrapPackageStore" "mdmlab.MDMBootstrapPackageStore"
//go:generate go run ./mockimpl/impl.go -o software/software_installer_store.go "s *SoftwareInstallerStore" "mdmlab.SoftwareInstallerStore"

var _ mdmlab.Datastore = (*Store)(nil)

type Store struct {
	DataStore
}

func (m *Store) EnrollOrbit(ctx context.Context, isMDMEnabled bool, orbitHostInfo mdmlab.OrbitHostInfo, orbitNodeKey string, teamID *uint) (*mdmlab.Host, error) {
	return nil, nil
}

func (m *Store) LoadHostByOrbitNodeKey(ctx context.Context, orbitNodeKey string) (*mdmlab.Host, error) {
	return nil, nil
}

func (m *Store) Drop() error                             { return nil }
func (m *Store) MigrateTables(ctx context.Context) error { return nil }
func (m *Store) MigrateData(ctx context.Context) error   { return nil }
func (m *Store) MigrationStatus(ctx context.Context) (*mdmlab.MigrationStatus, error) {
	return &mdmlab.MigrationStatus{}, nil
}
func (m *Store) Name() string { return "mock" }
