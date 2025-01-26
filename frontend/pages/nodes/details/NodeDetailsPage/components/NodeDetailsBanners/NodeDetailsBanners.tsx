import React, { useContext } from "react";
import { AppContext } from "context/app";

import { hasLicenseExpired } from "utilities/helpers";

import { DiskEncryptionStatus, MdmEnrollmentStatus } from "interfaces/mdm";
import { IOSSettings } from "interfaces/node";
import {
  NodePlatform,
  isDiskEncryptionSupportedLinuxPlatform,
} from "interfaces/platform";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "node-details-banners";

export interface INodeBannersBaseProps {
  macDiskEncryptionStatus: DiskEncryptionStatus | null | undefined;
  mdmEnrollmentStatus: MdmEnrollmentStatus | null;
  connectedToMdmlabMdm?: boolean;
  nodePlatform?: NodePlatform;
  // used to identify Fedora nodes, whose platform is "rhel"
  nodeOsVersion?: string;
  /** Disk encryption setting status and detail, if any, that apply to this node (via a team or the "no team" team) */
  diskEncryptionOSSetting?: IOSSettings["disk_encryption"];
  /** Whether or not this node's disk is encrypted */
  diskIsEncrypted?: boolean;
  /** Whether or not Mdmlab has escrowed the node's disk encryption key */
  diskEncryptionKeyAvailable?: boolean;
}
/**
 * Handles the displaying of banners on the node details page
 */
const NodeDetailsBanners = ({
  mdmEnrollmentStatus,
  nodePlatform,
  nodeOsVersion,
  connectedToMdmlabMdm,
  macDiskEncryptionStatus,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
}: INodeBannersBaseProps) => {
  const {
    config,
    isPremiumTier,
    isAppleBmExpired,
    isApplePnsExpired,
    isVppExpired,
    needsAbmTermsRenewal,
    willAppleBmExpire,
    willApplePnsExpire,
    willVppExpire,
  } = useContext(AppContext);

  // Checks to see if an app-wide banner is being shown (the ABM terms, ABM expiry,
  // or APNs expiry banner) in a parent component. App-wide banners found in parent
  // component take priority over node details page-level banners.
  const isMdmlabLicenseExpired = hasLicenseExpired(
    config?.license.expiration || ""
  );

  const showingAppWideBanner =
    isPremiumTier &&
    (needsAbmTermsRenewal ||
      isApplePnsExpired ||
      willApplePnsExpire ||
      isAppleBmExpired ||
      willAppleBmExpire ||
      isVppExpired ||
      willVppExpire ||
      isMdmlabLicenseExpired);

  const isMdmUnenrolled = mdmEnrollmentStatus === "Off" || !mdmEnrollmentStatus;

  const showTurnOnMdmInfoBanner =
    !showingAppWideBanner &&
    nodePlatform === "darwin" &&
    isMdmUnenrolled &&
    config?.mdm.enabled_and_configured;

  const showMacDiskEncryptionUserActionRequired =
    !showingAppWideBanner &&
    config?.mdm.enabled_and_configured &&
    connectedToMdmlabMdm &&
    macDiskEncryptionStatus === "action_required";

  if (showTurnOnMdmInfoBanner) {
    return (
      <div className={baseClass}>
        <InfoBanner color="yellow">
          To enforce settings, OS updates, disk encryption, and more, ask the
          end user to follow the <strong>Turn on MDM</strong> instructions on
          their <strong>My device</strong> page.
        </InfoBanner>
      </div>
    );
  }
  if (showMacDiskEncryptionUserActionRequired) {
    return (
      <div className={baseClass}>
        <InfoBanner color="yellow">
          Disk encryption: Requires action from the end user. Ask the end user
          to log out of their device or restart it.
        </InfoBanner>
      </div>
    );
  }
  if (
    nodePlatform &&
    isDiskEncryptionSupportedLinuxPlatform(nodePlatform, nodeOsVersion ?? "") &&
    diskEncryptionOSSetting?.status
  ) {
    // setting applies to a Linux node
    if (!diskIsEncrypted) {
      // linux node not in compliance with setting
      return (
        <div className={baseClass}>
          <InfoBanner
            color="yellow"
            cta={
              <CustomLink
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/mdm-disk-encryption`}
                text="Guide"
                color="core-mdmlab-black"
                iconColor="core-mdmlab-black"
                newTab
              />
            }
          >
            Disk encryption: Disk encryption is off. Currently, to turn on{" "}
            <b>full</b> disk encryption, the end user has to re-install their
            operating system.
          </InfoBanner>
        </div>
      );
    }
    if (!diskEncryptionKeyAvailable) {
      // linux node's disk is encrypted, but Mdmlab doesn't yet have a disk
      // encryption key escrowed (note that this state is also possible for Windows nodes, which we
      // don't show this banner for currently)
      return (
        <div className={baseClass}>
          <InfoBanner color="yellow">
            Disk encryption: Requires action from the end user. Ask the user to
            follow <b>Disk encryption</b> instructions on their <b>My device</b>{" "}
            page.
          </InfoBanner>
        </div>
      );
    }
  }
  return null;
};

export default NodeDetailsBanners;
