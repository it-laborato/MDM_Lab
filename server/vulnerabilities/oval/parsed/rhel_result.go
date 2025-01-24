package oval_parsed

import (
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
)

type RhelResult struct {
	Definitions        []Definition
	RpmInfoTests       map[int]*RpmInfoTest
	RpmVerifyFileTests map[int]*RpmVerifyFileTest
}

// NewRhelResult is the result of parsing an OVAL file that targets a Rhel based distro.
func NewRhelResult() *RhelResult {
	return &RhelResult{
		RpmInfoTests:       make(map[int]*RpmInfoTest),
		RpmVerifyFileTests: make(map[int]*RpmVerifyFileTest),
	}
}

func (r RhelResult) Eval(ver mdmlab.OSVersion, software []mdmlab.Software) ([]mdmlab.SoftwareVulnerability, error) {
	// Rpm Info Test Id => Matching software
	pkgTstResults := make(map[int][]mdmlab.Software)
	for i, t := range r.RpmInfoTests {
		rEval, err := t.Eval(software)
		if err != nil {
			return nil, err
		}
		pkgTstResults[i] = rEval
	}

	// Evaluate RpmVerifyFileTests, which are used to make assertions against the installed OS
	OSTstResults := make(map[int]bool)
	for i, t := range r.RpmVerifyFileTests {
		rEval, err := t.Eval(ver)
		if err != nil {
			return nil, err
		}
		OSTstResults[i] = rEval
	}

	vuln := make([]mdmlab.SoftwareVulnerability, 0)
	for _, d := range r.Definitions {
		if !d.Eval(OSTstResults, pkgTstResults) {
			continue
		}

		for _, tId := range d.CollectTestIds() {
			for _, software := range pkgTstResults[tId] {
				for _, v := range d.CveVulnerabilities() {
					vuln = append(vuln, mdmlab.SoftwareVulnerability{
						SoftwareID: software.ID,
						CVE:        v,
					})
				}
			}
		}
	}

	return vuln, nil
}

// EvalUname is not implemented for Rhel based distros
func (r RhelResult) EvalKernel(software []mdmlab.Software) ([]mdmlab.SoftwareVulnerability, error) {
	return nil, nil
}
