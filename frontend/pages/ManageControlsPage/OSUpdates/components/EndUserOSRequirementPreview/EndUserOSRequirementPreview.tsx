import React from "react";


import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

import MacOSUpdateScreenshot from "../../../../../../assets/images/macos-updates-preview.png";
import WindowsUpdateScreenshot from "../../../../../../assets/images/windows-nudge-screenshot.png";
import IOSUpdateScreenshot from "../../../../../../assets/images/ios-updates-preview.png";
import IPadOSUpdateScreenshot from "../../../../../../assets/images/ipados-updates-preview.png";

const baseClass = "os-requirement-preview";

interface IEndUserOSRequirementPreviewProps {
  platform: OSUpdatesSupportedPlatform;
}
const OSRequirementDescription = ({
  platform,
}: IEndUserOSRequirementPreviewProps) => {
  switch (platform) {
    case "windows":
      return (
        <>
          <h3>End user experience on Windows</h3>
          <p>
            When a Windows node becomes aware of a new update, end users are
            able to defer restarts. Automatic restarts happen before 8am and
            after 5pm (end user&apos;s local time). After the deadline, restarts
            are forced regardless of active hours.
          </p>
          
        </>
      );
    default:
      return <></>;
  }
};

const OSRequirementImage = ({
  platform,
}: IEndUserOSRequirementPreviewProps) => {
  const getScreenshot = () => {
    switch (platform) {
      case "windows":
        return WindowsUpdateScreenshot;
      default:
        WindowsUpdateScreenshot;
    }
  };

  return (
    <img
      className={`${baseClass}__preview-img`}
      src={getScreenshot()}
      alt="OS update preview screenshot"
    />
  );
};

const EndUserOSRequirementPreview = ({
  platform,
}: IEndUserOSRequirementPreviewProps) => {
  // FIXME: on slow connection the image loads after the text which looks weird and can cause a
  // mismatch between the text and the image when switching between platforms. We should load the
  // image first and then the text.
  return (
    <div className={baseClass}>
      <OSRequirementDescription platform={platform} />
      <OSRequirementImage platform={platform} />
    </div>
  );
};

export default EndUserOSRequirementPreview;
