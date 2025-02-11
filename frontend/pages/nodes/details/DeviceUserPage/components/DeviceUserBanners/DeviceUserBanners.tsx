import React from "react";

import InfoBanner from "components/InfoBanner";
import Button from "components/buttons/Button";
import { MacDiskEncryptionActionRequired } from "interfaces/node";
import { INodeBannersBaseProps } from "pages/nodes/details/NodeDetailsPage/components/NodeDetailsBanners/NodeDetailsBanners";

import { isDiskEncryptionSupportedLinuxPlatform } from "interfaces/platform";

const baseClass = "device-user-banners";

interface IDeviceUserBannersProps extends INodeBannersBaseProps {
  mdmEnabledAndConfigured: boolean;
  diskEncryptionActionRequired: MacDiskEncryptionActionRequired | null;
  onTurnOnMdm: () => void;
  onTriggerEscrowLinuxKey: () => void;
}

const DeviceUserBanners = ({
  nodePlatform,
  nodeOsVersion,
  mdmEnrollmentStatus,
  mdmEnabledAndConfigured,
  connectedToMdmlabMdm,
  macDiskEncryptionStatus,
  diskEncryptionActionRequired,
  onTurnOnMdm,
  diskEncryptionOSSetting,
  diskIsEncrypted,
  diskEncryptionKeyAvailable,
  onTriggerEscrowLinuxKey,
}: IDeviceUserBannersProps) => {
  const isMdmUnenrolled =
    mdmEnrollmentStatus === "Off" || mdmEnrollmentStatus === null;

  const mdmEnabledAndConnected = mdmEnabledAndConfigured && connectedToMdmlabMdm;

  const showTurnOnAppleMdmBanner =
    nodePlatform === "darwin" && isMdmUnenrolled && mdmEnabledAndConfigured;

  const showMacDiskEncryptionKeyResetRequired =
    mdmEnabledAndConnected &&
    macDiskEncryptionStatus === "action_required" &&
    diskEncryptionActionRequired === "rotate_key";

  const turnOnMdmButton = (
    <Button variant="unstyled" onClick={onTurnOnMdm}>
      <b>Turn on MDM</b>
    </Button>
  );

  const renderBanner = () => {
    if (showTurnOnAppleMdmBanner) {
      return (
        <InfoBanner color="yellow" cta={turnOnMdmButton}>
          Mobile device management (MDM) is off. MDM allows your organization to
          enforce settings, OS updates, disk encryption, and more. This lets
          your organization keep your device up to date so you don&apos;t have
          to.
        </InfoBanner>
      );
    }

    if (showMacDiskEncryptionKeyResetRequired) {
      return (
        <InfoBanner color="yellow">
          Disk encryption: Log out of your device or restart it to safeguard
          your data in case your device is lost or stolen. After, select{" "}
          <strong>Refetch</strong> to clear this banner.
        </InfoBanner>
      );
    }

    // setting applies to a supported Linux node
    if (
      nodePlatform &&
      isDiskEncryptionSupportedLinuxPlatform(
        nodePlatform,
        nodeOsVersion ?? ""
      ) &&
      diskEncryptionOSSetting?.status
    ) {
      // node not in compliance with setting
      if (!diskIsEncrypted) {
        // banner 1
        return (
          <InfoBanner
            
            color="yellow"
          >
            Disk encryption: Follow the instructions in the guide to encrypt
            your device. This lets your organization help you unlock your device
            if you forget your password.
          </InfoBanner>
        );
      }
      // node disk is encrypted, so in compliance with the setting
      if (!diskEncryptionKeyAvailable) {
        // key is not escrowed: banner 3
        return (
          <InfoBanner
            cta={
              <Button
                variant="unstyled"
                onClick={onTriggerEscrowLinuxKey}
                className="create-key-button"
              >
                Create key
              </Button>
            }
            color="yellow"
          >
            Disk encryption: Create a new disk encryption key. This lets your
            organization help you unlock your device if you forget your
            passphrase.
          </InfoBanner>
        );
      }
    }

    return null;
  };

  return <div className={baseClass}>{renderBanner()}</div>;
};

export default DeviceUserBanners;
