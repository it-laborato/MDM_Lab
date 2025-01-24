package main

import (
	"github.com/it-laborato/MDM_Lab/server/config"
	"github.com/it-laborato/MDM_Lab/server/version"
	"github.com/spf13/cobra"
)

func createVersionCmd(configManager config.Manager) *cobra.Command {
	// flags
	var (
		fFull bool
	)
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print MDMlab version",
		Long: `
Print version information and related build info`,
		Run: func(cmd *cobra.Command, args []string) {
			if fFull {
				version.PrintFull()
				return
			}
			version.Print()
		},
	}

	versionCmd.PersistentFlags().BoolVar(&fFull, "full", false, "print full version information")

	return versionCmd
}
