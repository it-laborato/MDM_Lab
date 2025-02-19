package eemdmlabctl

import "github.com/urfave/cli/v2"

func LocalWixDirFlag(dest *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "local-wix-dir",
		Usage:       "Use local install of WiX instead of Docker Hub (only available on Windows and macOS w/ WiX v3). This functionality is licensed under the MDMlab EE License. Usage requires a current MDMlab EE subscription.",
		Destination: dest,
	}
}
