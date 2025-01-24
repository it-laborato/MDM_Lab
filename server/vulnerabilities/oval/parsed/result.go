package oval_parsed

import "github.com:it-laborato/MDM_Lab/server/mdmlab"

type Result interface {
	// Eval evaluates the current OVAL definition against an OS version and a list of installed software, returns all software
	// vulnerabilities found.
	Eval(mdmlab.OSVersion, []mdmlab.Software) ([]mdmlab.SoftwareVulnerability, error)

	// EvalKernel evaluates the current OVAL definition against a list of installed kernel-image software,
	// returns all kernel-image vulnerabilities found.  Currently only used for Ubuntu.
	EvalKernel([]mdmlab.Software) ([]mdmlab.SoftwareVulnerability, error)
}
