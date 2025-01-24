package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com:it-laborato/MDM_Lab/server/authz"
	"github.com:it-laborato/MDM_Lab/server/contexts/ctxerr"
	"github.com:it-laborato/MDM_Lab/server/mdmlab"
	"github.com:it-laborato/MDM_Lab/server/mdm/apple/itunes"
	"github.com:it-laborato/MDM_Lab/server/mdm/apple/vpp"
)

// Used for overriding the env var value in testing
var testSetEmptyPrivateKey bool

// getVPPToken returns the base64 encoded VPP token, ready for use in requests to Apple's VPP API.
// It returns an error if the token is expired.
func (svc *Service) getVPPToken(ctx context.Context, teamID *uint) (string, error) {
	token, err := svc.ds.GetVPPTokenByTeamID(ctx, teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", mdmlab.NewUserMessageError(errors.New("No available VPP Token"), http.StatusUnprocessableEntity)
		}
		return "", ctxerr.Wrap(ctx, err, "fetching vpp token")
	}

	if time.Now().After(token.RenewDate) {
		return "", mdmlab.NewUserMessageError(errors.New("Couldn't install. VPP token expired."), http.StatusUnprocessableEntity)
	}

	return token.Token, nil
}

func (svc *Service) BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []mdmlab.VPPBatchPayload, dryRun bool) ([]mdmlab.VPPAppResponse, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.Team{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	var teamID *uint
	if teamName != "" {
		tm, err := svc.ds.TeamByName(ctx, teamName)
		if err != nil {
			// If this is a dry run, the team may not have been created yet
			if dryRun && mdmlab.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		teamID = &tm.ID
	}

	if err := svc.authz.Authorize(ctx, &mdmlab.SoftwareInstaller{TeamID: teamID}, mdmlab.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating authorization")
	}

	// Adding VPP apps will add them to all available platforms per decision:
	// https://github.com/mdmlabdm/mdmlab/issues/19447#issuecomment-2256598681
	// The code is already here to support individual platforms, so we can easily enable it later.

	payloadsWithPlatform := make([]mdmlab.VPPBatchPayloadWithPlatform, 0, len(payloads))
	for _, payload := range payloads {
		// Currently only macOS is supported for self-service. Don't
		// import vpp apps as self-service for ios or ipados
		payloadsWithPlatform = append(payloadsWithPlatform, []mdmlab.VPPBatchPayloadWithPlatform{{
			AppStoreID:  payload.AppStoreID,
			SelfService: false,
			Platform:    mdmlab.IOSPlatform,
		}, {
			AppStoreID:  payload.AppStoreID,
			SelfService: false,
			Platform:    mdmlab.IPadOSPlatform,
		}, {
			AppStoreID:         payload.AppStoreID,
			SelfService:        payload.SelfService,
			Platform:           mdmlab.MacOSPlatform,
			InstallDuringSetup: payload.InstallDuringSetup,
		}}...)
	}

	if dryRun {
		// On dry runs return early because the VPP token might not exist yet
		// and we don't want to apply the VPP apps.
		return nil, nil
	}

	var vppAppTeams []mdmlab.VPPAppTeam
	// Don't check for token if we're only disassociating assets
	if len(payloads) > 0 {
		token, err := svc.getVPPToken(ctx, teamID)
		if err != nil {
			return nil, mdmlab.NewUserMessageError(ctxerr.Wrap(ctx, err, "could not retrieve vpp token"), http.StatusUnprocessableEntity)
		}

		for _, payload := range payloadsWithPlatform {
			if payload.Platform == "" {
				payload.Platform = mdmlab.MacOSPlatform
			}
			if payload.Platform != mdmlab.IOSPlatform && payload.Platform != mdmlab.IPadOSPlatform && payload.Platform != mdmlab.MacOSPlatform {
				return nil, mdmlab.NewInvalidArgumentError("app_store_apps.platform",
					fmt.Sprintf("platform must be one of '%s', '%s', or '%s", mdmlab.IOSPlatform, mdmlab.IPadOSPlatform, mdmlab.MacOSPlatform))
			}
			vppAppTeams = append(vppAppTeams, mdmlab.VPPAppTeam{
				VPPAppID: mdmlab.VPPAppID{
					AdamID:   payload.AppStoreID,
					Platform: payload.Platform,
				},
				SelfService:        payload.SelfService,
				InstallDuringSetup: payload.InstallDuringSetup,
			})
		}

		var missingAssets []string

		assets, err := vpp.GetAssets(token, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unable to retrieve assets")
		}

		assetMap := map[string]struct{}{}
		for _, asset := range assets {
			assetMap[asset.AdamID] = struct{}{}
		}

		for _, vppAppID := range vppAppTeams {
			if _, ok := assetMap[vppAppID.AdamID]; !ok {
				missingAssets = append(missingAssets, vppAppID.AdamID)
			}
		}

		if len(missingAssets) != 0 {
			reqErr := ctxerr.Errorf(ctx, "requested app not available on vpp account: %s", strings.Join(missingAssets, ","))
			return nil, mdmlab.NewUserMessageError(reqErr, http.StatusUnprocessableEntity)
		}
	}

	if len(vppAppTeams) > 0 {
		apps, err := getVPPAppsMetadata(ctx, vppAppTeams)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "refreshing VPP app metadata")
		}
		if len(apps) == 0 {
			return nil, mdmlab.NewInvalidArgumentError("app_store_apps",
				"no valid apps found matching the provided app store IDs and platforms")
		}

		if err := svc.ds.BatchInsertVPPApps(ctx, apps); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "inserting vpp app metadata")
		}
		// Filter out the apps with invalid platforms
		if len(apps) != len(vppAppTeams) {
			vppAppTeams = make([]mdmlab.VPPAppTeam, 0, len(apps))
			for _, app := range apps {
				vppAppTeams = append(vppAppTeams, app.VPPAppTeam)
			}
		}

	}
	if err := svc.ds.SetTeamVPPApps(ctx, teamID, vppAppTeams); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, mdmlab.NewUserMessageError(ctxerr.Wrap(ctx, err, "no vpp token to set team vpp assets"), http.StatusUnprocessableEntity)
		}
		return nil, ctxerr.Wrap(ctx, err, "set team vpp assets")
	}

	if len(vppAppTeams) == 0 {
		return []mdmlab.VPPAppResponse{}, nil
	}

	return svc.ds.GetVPPApps(ctx, teamID)
}

func (svc *Service) GetAppStoreApps(ctx context.Context, teamID *uint) ([]*mdmlab.VPPApp, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.VPPApp{TeamID: teamID}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	vppToken, err := svc.getVPPToken(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving VPP token")
	}

	assets, err := vpp.GetAssets(vppToken, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching Apple VPP assets")
	}

	if len(assets) == 0 {
		return []*mdmlab.VPPApp{}, nil
	}

	var adamIDs []string
	for _, a := range assets {
		adamIDs = append(adamIDs, a.AdamID)
	}

	assetMetadata, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "software"})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	assignedApps, err := svc.ds.GetAssignedVPPApps(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving assigned VPP apps")
	}

	var apps []*mdmlab.VPPApp
	var appsToUpdate []*mdmlab.VPPApp
	for _, a := range assets {
		m, ok := assetMetadata[a.AdamID]
		if !ok {
			// Then this adam_id is not a VPP software entity, so skip it.
			continue
		}

		platforms := getPlatformsFromSupportedDevices(m.SupportedDevices)

		for platform := range platforms {
			vppAppID := mdmlab.VPPAppID{
				AdamID:   a.AdamID,
				Platform: platform,
			}
			vppAppTeam := mdmlab.VPPAppTeam{
				VPPAppID: vppAppID,
			}
			app := &mdmlab.VPPApp{
				VPPAppTeam:       vppAppTeam,
				BundleIdentifier: m.BundleID,
				IconURL:          m.ArtworkURL,
				Name:             m.TrackName,
				LatestVersion:    m.Version,
			}

			if appMDMlab, ok := assignedApps[vppAppID]; ok {
				// Then this is already assigned, so filter it out.
				app.SelfService = appMDMlab.SelfService
				appsToUpdate = append(appsToUpdate, app)
				continue
			}

			apps = append(apps, app)
		}
	}

	if len(appsToUpdate) > 0 {
		if err := svc.ds.BatchInsertVPPApps(ctx, appsToUpdate); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "updating existing VPP apps")
		}
	}

	// Sort apps by name and by platform
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Name != apps[j].Name {
			return apps[i].Name < apps[j].Name
		}
		return apps[i].Platform < apps[j].Platform
	})

	return apps, nil
}

func getPlatformsFromSupportedDevices(supportedDevices []string) map[mdmlab.AppleDevicePlatform]struct{} {
	platforms := make(map[mdmlab.AppleDevicePlatform]struct{}, 1)
	if len(supportedDevices) == 0 {
		platforms[mdmlab.MacOSPlatform] = struct{}{}
		return platforms
	}
	for _, device := range supportedDevices {
		// It is rare that a single app supports all platforms, but it is possible.
		switch {
		case strings.HasPrefix(device, "iPhone"):
			platforms[mdmlab.IOSPlatform] = struct{}{}
		case strings.HasPrefix(device, "iPad"):
			platforms[mdmlab.IPadOSPlatform] = struct{}{}
		case strings.HasPrefix(device, "Mac"):
			platforms[mdmlab.MacOSPlatform] = struct{}{}
		}
	}
	return platforms
}

func (svc *Service) AddAppStoreApp(ctx context.Context, teamID *uint, appID mdmlab.VPPAppTeam) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.VPPApp{TeamID: teamID}, mdmlab.ActionWrite); err != nil {
		return err
	}
	// Validate platform
	if appID.Platform == "" {
		appID.Platform = mdmlab.MacOSPlatform
	}
	if appID.Platform != mdmlab.IOSPlatform && appID.Platform != mdmlab.IPadOSPlatform && appID.Platform != mdmlab.MacOSPlatform {
		return mdmlab.NewInvalidArgumentError("platform",
			fmt.Sprintf("platform must be one of '%s', '%s', or '%s", mdmlab.IOSPlatform, mdmlab.IPadOSPlatform, mdmlab.MacOSPlatform))
	}

	var teamName string
	if teamID != nil && *teamID != 0 {
		tm, err := svc.ds.Team(ctx, *teamID)
		if mdmlab.IsNotFound(err) {
			return mdmlab.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		} else if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if team exists")
		}

		teamName = tm.Name
	}

	if appID.SelfService && appID.Platform != mdmlab.MacOSPlatform {
		return mdmlab.NewUserMessageError(errors.New("Currently, self-service only supports macOS"), http.StatusBadRequest)
	}

	vppToken, err := svc.getVPPToken(ctx, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving VPP token")
	}

	assets, err := vpp.GetAssets(vppToken, &vpp.AssetFilter{AdamID: appID.AdamID})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving VPP asset")
	}

	if len(assets) == 0 {
		return ctxerr.New(ctx,
			fmt.Sprintf("Error: Couldn't add software. %s isn't available in Apple Business Manager. Please purchase license in Apple Business Manager and try again.",
				appID.AdamID))
	}

	asset := assets[0]

	assetMetadata, err := itunes.GetAssetMetadata([]string{asset.AdamID}, &itunes.AssetMetadataFilter{Entity: "software"})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	assetMD := assetMetadata[asset.AdamID]

	platforms := getPlatformsFromSupportedDevices(assetMD.SupportedDevices)
	if _, ok := platforms[appID.Platform]; !ok {
		return mdmlab.NewInvalidArgumentError("app_store_id", fmt.Sprintf("%s isn't available for %s", assetMD.TrackName, appID.Platform))
	}

	if appID.Platform == mdmlab.MacOSPlatform {
		// Check if we've already added an installer for this app
		exists, err := svc.ds.UploadedSoftwareExists(ctx, assetMD.BundleID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking existence of VPP app installer")
		}

		if exists {
			return ctxerr.New(ctx,
				fmt.Sprintf("Error: Couldn't add software. %s already has software available for install on the %s team.",
					assetMD.TrackName, teamName))
		}
	}

	app := &mdmlab.VPPApp{
		VPPAppTeam:       appID,
		BundleIdentifier: assetMD.BundleID,
		IconURL:          assetMD.ArtworkURL,
		Name:             assetMD.TrackName,
		LatestVersion:    assetMD.Version,
	}

	addedApp, err := svc.ds.InsertVPPAppWithTeam(ctx, app, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "writing VPP app to db")
	}

	act := mdmlab.ActivityAddedAppStoreApp{
		AppStoreID:      app.AdamID,
		Platform:        app.Platform,
		TeamName:        &teamName,
		SoftwareTitle:   app.Name,
		SoftwareTitleId: addedApp.TitleID,
		TeamID:          teamID,
		SelfService:     app.SelfService,
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), act); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for add app store app")
	}

	return nil
}

func getVPPAppsMetadata(ctx context.Context, ids []mdmlab.VPPAppTeam) ([]*mdmlab.VPPApp, error) {
	var apps []*mdmlab.VPPApp

	// Map of adamID to platform, then to whether it's available as self-service
	// and installed during setup.
	adamIDMap := make(map[string]map[mdmlab.AppleDevicePlatform]mdmlab.VPPAppTeam)
	for _, id := range ids {
		if _, ok := adamIDMap[id.AdamID]; !ok {
			adamIDMap[id.AdamID] = make(map[mdmlab.AppleDevicePlatform]mdmlab.VPPAppTeam, 1)
			adamIDMap[id.AdamID][id.Platform] = mdmlab.VPPAppTeam{SelfService: id.SelfService, InstallDuringSetup: id.InstallDuringSetup}
		} else {
			adamIDMap[id.AdamID][id.Platform] = mdmlab.VPPAppTeam{SelfService: id.SelfService, InstallDuringSetup: id.InstallDuringSetup}
		}
	}

	var adamIDs []string
	for adamID := range adamIDMap {
		adamIDs = append(adamIDs, adamID)
	}
	assetMetatada, err := itunes.GetAssetMetadata(adamIDs, &itunes.AssetMetadataFilter{Entity: "software"})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching VPP asset metadata")
	}

	for adamID, metadata := range assetMetatada {
		platforms := getPlatformsFromSupportedDevices(metadata.SupportedDevices)
		for platform := range platforms {
			if props, ok := adamIDMap[adamID][platform]; ok {
				app := &mdmlab.VPPApp{
					VPPAppTeam: mdmlab.VPPAppTeam{
						VPPAppID: mdmlab.VPPAppID{
							AdamID:   adamID,
							Platform: platform,
						},
						SelfService:        props.SelfService,
						InstallDuringSetup: props.InstallDuringSetup,
					},
					BundleIdentifier: metadata.BundleID,
					IconURL:          metadata.ArtworkURL,
					Name:             metadata.TrackName,
					LatestVersion:    metadata.Version,
				}
				apps = append(apps, app)
			} else {
				continue
			}
		}
	}

	return apps, nil
}

func (svc *Service) UploadVPPToken(ctx context.Context, token io.ReadSeeker) (*mdmlab.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.New(ctx, "Couldn't upload content token. Missing required private key. Learn how to configure the private key here: https://mdmlabdm.com/learn-more-about/mdmlab-server-private-key")
	}

	if token == nil {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
	}

	tokenBytes, err := io.ReadAll(token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading VPP token")
	}

	locName, err := vpp.GetConfig(string(tokenBytes))
	if err != nil {
		var vppErr *vpp.ErrorResponse
		if errors.As(err, &vppErr) {
			// Per https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			if vppErr.ErrorNumber == 9622 {
				return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "validating VPP token with Apple")
	}

	data := mdmlab.VPPTokenData{
		Token:    string(tokenBytes),
		Location: locName,
	}

	tok, err := svc.ds.InsertVPPToken(ctx, &data)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "writing VPP token to db")
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), mdmlab.ActivityEnabledVPP{
		Location: locName,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity for upload VPP token")
	}

	return tok, nil
}

func (svc *Service) UpdateVPPToken(ctx context.Context, tokenID uint, token io.ReadSeeker) (*mdmlab.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.New(ctx, "Couldn't upload content token. Missing required private key. Learn how to configure the private key here: https://mdmlabdm.com/learn-more-about/mdmlab-server-private-key")
	}

	if token == nil {
		return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
	}

	tokenBytes, err := io.ReadAll(token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading VPP token")
	}

	locName, err := vpp.GetConfig(string(tokenBytes))
	if err != nil {
		var vppErr *vpp.ErrorResponse
		if errors.As(err, &vppErr) {
			// Per https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			if vppErr.ErrorNumber == 9622 {
				return nil, ctxerr.Wrap(ctx, mdmlab.NewInvalidArgumentError("token", "Invalid token. Please provide a valid content token from Apple Business Manager."))
			}
		}
		return nil, ctxerr.Wrap(ctx, err, "validating VPP token with Apple")
	}

	data := mdmlab.VPPTokenData{
		Token:    string(tokenBytes),
		Location: locName,
	}

	tok, err := svc.ds.UpdateVPPToken(ctx, tokenID, &data)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating vpp token")
	}

	return tok, nil
}

func (svc *Service) UpdateVPPTokenTeams(ctx context.Context, tokenID uint, teamIDs []uint) (*mdmlab.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return nil, err
	}

	tok, err := svc.ds.UpdateVPPTokenTeams(ctx, tokenID, teamIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating vpp token team")
	}

	return tok, nil
}

func (svc *Service) GetVPPTokens(ctx context.Context) ([]*mdmlab.VPPTokenDB, error) {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListVPPTokens(ctx)
}

func (svc *Service) DeleteVPPToken(ctx context.Context, tokenID uint) error {
	if err := svc.authz.Authorize(ctx, &mdmlab.AppleCSR{}, mdmlab.ActionWrite); err != nil {
		return err
	}
	tok, err := svc.ds.GetVPPToken(ctx, tokenID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting vpp token")
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), mdmlab.ActivityDisabledVPP{
		Location: tok.Location,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for delete VPP token")
	}

	return svc.ds.DeleteVPPToken(ctx, tokenID)
}
