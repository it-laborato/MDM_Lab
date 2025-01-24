//go:build linux

package table

import (
	"github.com:it-laborato/MDM_Lab/orbit/pkg/table/crowdstrike/falcon_kernel_check"
	"github.com:it-laborato/MDM_Lab/orbit/pkg/table/crowdstrike/falconctl"
	"github.com:it-laborato/MDM_Lab/orbit/pkg/table/cryptsetup"
	"github.com:it-laborato/MDM_Lab/orbit/pkg/table/dataflattentable"
	"github.com/rs/zerolog/log"

	"github.com/osquery/osquery-go"
)

func PlatformTables(_ PluginOpts) ([]osquery.OsqueryPlugin, error) {
	return []osquery.OsqueryPlugin{
		cryptsetup.TablePlugin(log.Logger),            // table name is "cryptsetup_status"
		falconctl.NewFalconctlOptionTable(log.Logger), // table name is "falconctl_option"
		falcon_kernel_check.TablePlugin(log.Logger),   // table name is "falcon_kernel_check"
		dataflattentable.TablePluginExec(log.Logger, "nftables", dataflattentable.JsonType, []string{"nft", "-jat", "list", "ruleset"}, dataflattentable.WithBinDirs("/usr/bin", "/usr/sbin")), // -j (json) -a (show object handles) -t (terse, omit set contents)
	}, nil
}
