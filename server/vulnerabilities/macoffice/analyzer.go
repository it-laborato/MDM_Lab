package macoffice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/it-laborato/MDM_Lab/server/vulnerabilities/io"
	"github.com/it-laborato/MDM_Lab/server/vulnerabilities/utils"
)

// getLatestReleaseNotes returns the most recent Mac Office release notes asset (based on the date in the
// filename) contained in 'vulnPath'
func getLatestReleaseNotes(vulnPath string) (ReleaseNotes, error) {
	fs := io.NewFSClient(vulnPath)

	files, err := fs.MacOfficeReleaseNotes()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Slice(files, func(i, j int) bool { return files[j].Before(files[i]) })
	filePath := filepath.Join(vulnPath, files[0].String())

	payload, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	relNotes := ReleaseNotes{}
	err = json.Unmarshal(payload, &relNotes)
	if err != nil {
		return nil, err
	}

	// Ensure the release notes are sorted by release date, this is because the vuln. processing
	// algo. will stop when a release note older than the current software version is found.
	sort.Slice(relNotes, func(i, j int) bool { return relNotes[j].Date.Before(relNotes[i].Date) })

	return relNotes, nil
}

// collectVulnerabilities compares 'software' against all 'release notes' returning all detected
// vulnerabilities.
func collectVulnerabilities(
	software *mdmlab.Software,
	product ProductType,
	relNotes ReleaseNotes,
) []mdmlab.SoftwareVulnerability {
	var vulns []mdmlab.SoftwareVulnerability
	for _, relNote := range relNotes {
		// We only care about release notes with set versions and with security updates,
		// 'relNotes' should only contain valid release notes, but this check not expensive.
		if !relNote.Valid() {
			continue
		}

		if relNote.CmpVersion(software.Version) <= 0 {
			return vulns
		}

		for _, cve := range relNote.CollectVulnerabilities(product) {
			vulns = append(vulns, mdmlab.SoftwareVulnerability{
				SoftwareID: software.ID,
				CVE:        cve,
			})
		}
	}
	return vulns
}

// getStoredVulnerabilities return all stored vulnerabilities for 'softwareID'
func getStoredVulnerabilities(
	ctx context.Context,
	ds mdmlab.Datastore,
	softwareID uint,
) ([]mdmlab.SoftwareVulnerability, error) {
	storedSoftware, err := ds.SoftwareByID(ctx, softwareID, nil, false, nil)
	if err != nil {
		return nil, err
	}

	var result []mdmlab.SoftwareVulnerability
	for _, v := range storedSoftware.Vulnerabilities {
		result = append(result, mdmlab.SoftwareVulnerability{
			SoftwareID: storedSoftware.ID,
			CVE:        v.CVE,
		})
	}
	return result, nil
}

func updateVulnsInDB(
	ctx context.Context,
	ds mdmlab.Datastore,
	detected []mdmlab.SoftwareVulnerability,
	existing []mdmlab.SoftwareVulnerability,
) ([]mdmlab.SoftwareVulnerability, error) {
	toInsert, toDelete := utils.VulnsDelta(detected, existing)

	// Remove any possible dups...
	toInsertSet := make(map[string]mdmlab.SoftwareVulnerability, len(toInsert))
	for _, i := range toInsert {
		toInsertSet[i.Key()] = i
	}

	err := ds.DeleteSoftwareVulnerabilities(ctx, toDelete)
	if err != nil {
		return nil, err
	}

	inserted := make([]mdmlab.SoftwareVulnerability, 0, len(toInsertSet))
	err = utils.BatchProcess(toInsertSet, func(vulns []mdmlab.SoftwareVulnerability) error {
		for _, v := range vulns {
			ok, err := ds.InsertSoftwareVulnerability(ctx, v, mdmlab.MacOfficeReleaseNotesSource)
			if err != nil {
				return err
			}

			if ok {
				inserted = append(inserted, v)
			}
		}

		return nil
		// Since we are only detecting Mac Office vulnerabilities 'toInsertSet' should be small, so
		// inserting the whole batch in one go should be ok.
	}, len(toInsertSet))
	if err != nil {
		return nil, err
	}

	return inserted, nil
}

// Analyze uses the most recent Mac Office release notes asset in 'vulnPath' for detecting
// vulnerabilities on Mac Office apps.
func Analyze(
	ctx context.Context,
	ds mdmlab.Datastore,
	vulnPath string,
	collectVulns bool,
) ([]mdmlab.SoftwareVulnerability, error) {
	relNotes, err := getLatestReleaseNotes(vulnPath)
	if err != nil {
		return nil, err
	}

	if len(relNotes) == 0 {
		return nil, nil
	}

	queryParams := mdmlab.SoftwareIterQueryOptions{IncludedSources: []string{"apps"}}
	iter, err := ds.AllSoftwareIterator(ctx, queryParams)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var vulnerabilities []mdmlab.SoftwareVulnerability
	for iter.Next() {
		software, err := iter.Value()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting software from iterator")
		}

		product, ok := OfficeProductFromBundleId(software.BundleIdentifier)
		// If we don't have an Office Product ...
		if !ok {
			continue
		}

		detected := collectVulnerabilities(software, product, relNotes)
		// The 'software' instance we get back from the iterator does not include vulnerabilities...
		existing, err := getStoredVulnerabilities(ctx, ds, software.ID)
		if err != nil {
			return nil, err
		}

		inserted, err := updateVulnsInDB(ctx, ds, detected, existing)
		if err != nil {
			return nil, err
		}

		if collectVulns {
			vulnerabilities = append(vulnerabilities, inserted...)
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("iter: %w", err)
	}

	return vulnerabilities, nil
}
