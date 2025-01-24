package update

import "github.com:it-laborato/MDM_Lab/server/mdmlab"

// OrbitConfigFetcher allows fetching Orbit configuration.
type OrbitConfigFetcher interface {
	// GetConfig returns the Orbit configuration.
	GetConfig() (*mdmlab.OrbitConfig, error)
}
