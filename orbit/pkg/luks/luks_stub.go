//go:build !linux
// +build !linux

package luks

import "github.com/it-laborato/MDM_Lab/server/mdmlab"

// Run is a placeholder method for non-Linux builds.
func (lr *LuksRunner) Run(oc *mdmlab.OrbitConfig) error {
	return nil
}
