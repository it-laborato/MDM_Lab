import React from "react";
import classNames from "classnames";

import { IPolicySoftwareToInstall } from "interfaces/policy";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import { buildQueryStringFromParams } from "utilities/url";
import paths from "router/paths";

interface IPlatformSelectorProps {
  baseClass?: string;
  // checkDarwin: boolean;
  checkWindows: boolean;
  // checkLinux: boolean;
  // checkChrome: boolean;
  // setCheckDarwin: (val: boolean) => void;
  setCheckWindows: (val: boolean) => void;
  // setCheckLinux: (val: boolean) => void;
  // setCheckChrome: (val: boolean) => void;
  disabled?: boolean;
  installSoftware?: IPolicySoftwareToInstall;
  currentTeamId?: number;
}

export const PlatformSelector = ({
  baseClass: parentClass,
  // checkDawin,
  checkWindows,
  // checkLinux,
  // checkChrome,
  // setCheckDarwin,
  setCheckWindows,
  // setCheckLinux,
  // setCheckChrome,
  disabled = false,
  installSoftware,
  currentTeamId,
}: IPlatformSelectorProps): JSX.Element => {
  const baseClass = "platform-selector";

  const labelClasses = classNames("form-field__label", {
    [`form-field__label--disabled`]: disabled,
  });

  const renderInstallSoftwareHelpText = () => {
    if (!installSoftware) {
      return null;
    }
    const softwareName = installSoftware.name;
    const softwareId = installSoftware.software_title_id.toString();
    const softwareLink = `${paths.SOFTWARE_TITLE_DETAILS(
      softwareId
    )}?${buildQueryStringFromParams({ team_id: currentTeamId })}`;

    return (
      <span className={`${baseClass}__install-software`}>
       
        <TooltipWrapper
          tipContent={
            <>
              To see targets, select{" "}
              <b>{softwareName} &gt; Actions &gt; Edit</b>. Currently, nodes
              that aren&apos;t targeted show an empty (---) policy status.
            </>
          }
        >
          targeted nodes
        </TooltipWrapper>
        .
      </span>
    );
  };

  return (
    <div className={`${parentClass}__${baseClass} ${baseClass} form-field`}>
      <span className={labelClasses}>Target:</span>
      <span className={`${baseClass}__checkboxes`}>
        <Checkbox
          value={checkWindows}
          onChange={(value: boolean) => setCheckWindows(value)}
          wrapperClassName={`${baseClass}__platform-checkbox-wrapper`}
          disabled={disabled}
        >
          Windows
        </Checkbox>
	  </span>
      <div className="form-field__help-text">
        Policy runs on all nodes with these platform(s).
        {renderInstallSoftwareHelpText()}
      </div>
    </div>
  );
};

export default PlatformSelector;
