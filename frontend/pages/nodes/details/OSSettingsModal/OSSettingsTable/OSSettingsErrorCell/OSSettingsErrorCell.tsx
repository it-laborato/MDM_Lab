import React, { useContext, useState } from "react";
import classnames from "classnames";
import { noop } from "lodash";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import nodeAPI from "services/entities/nodes";
import { NotificationContext } from "context/notification";

import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import Button from "components/buttons/Button";
import Icon from "components/Icon";


import { INodeMdmProfileWithAddedStatus } from "../OSSettingsTableConfig";

const baseClass = "os-settings-error-cell";

interface IRefetchButtonProps {
  isFetching: boolean;
  onClick: (evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>) => void;
}

const RefetchButton = ({ isFetching, onClick }: IRefetchButtonProps) => {
  const classNames = classnames(`${baseClass}__resend-button`, "resend-link", {
    [`${baseClass}__resending`]: isFetching,
  });

  const buttonText = isFetching ? "Resending..." : "Resend";

  // add additonal props when we need to display a tooltip for the button

  return (
    <Button
      disabled={isFetching}
      onClick={onClick}
      variant="text-icon"
      className={classNames}
    >
      <Icon name="refresh" color="core-mdmlab-blue" size="small" />
      {buttonText}
    </Button>
  );
};

/**
 * generates the formatted tooltip for the error column.
 * the expected format of the error string is:
 * "key1: value1, key2: value2, key3: value3"
 */
const generateFormattedTooltip = (detail: string) => {
  const keyValuePairs = detail.split(/, */);
  const formattedElements: JSX.Element[] = [];

  // Special case to handle bitlocker error message. It does not follow the
  // expected string format so we will just render the error message as is.
  if (
    detail.includes("BitLocker") ||
    detail.includes("preparing volume for encryption")
  ) {
    return detail;
  }

  keyValuePairs.forEach((pair, i) => {
    const [key, value] = pair.split(/: */);
    if (key && value) {
      formattedElements.push(
        <span key={key}>
          <b>{key.trim()}:</b> {value.trim()}
          {/* dont add the trailing comma for the last element */}
          {i !== keyValuePairs.length - 1 && (
            <>
              ,<br />
            </>
          )}
        </span>
      );
    }
  });

  return formattedElements.length ? <>{formattedElements}</> : detail;
};

/**
 * generates the error tooltip for the error column. This will be formatted or
 * unformatted.
 */
const generateErrorTooltip = (
  cellValue: string,
  profile: INodeMdmProfileWithAddedStatus
) => {
  if (profile.status !== "failed") return null;

  // Special case for creating UI link
  if (profile.detail.includes("There is no IdP email for this node.")) {
    return (
      <>
        There is no IdP email for this node.
        <br />
        Mdmlab couldn&apos;t populate
        <br />
        $mdmlab_VAR_HOST_END_USER_EMAIL_IDP.
        <br />
        
      </>
    );
  }

  if (profile.platform !== "windows") {
    return cellValue;
  }

  return generateFormattedTooltip(profile.detail);
};

interface IOSSettingsErrorCellProps {
  canResendProfiles: boolean;
  nodeId: number;
  profile: INodeMdmProfileWithAddedStatus;
  onProfileResent?: () => void;
}

const OSSettingsErrorCell = ({
  canResendProfiles,
  nodeId,
  profile,
  onProfileResent = noop,
}: IOSSettingsErrorCellProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isLoading, setIsLoading] = useState(false);

  const onResendProfile = async () => {
    setIsLoading(true);
    try {
      await nodeAPI.resendProfile(nodeId, profile.profile_uuid);
      onProfileResent();
    } catch (e) {
      renderFlash("error", "Couldn't resend. Please try again.");
    }
    setIsLoading(false);
  };

  const isFailed = profile.status === "failed";
  const isVerified = profile.status === "verified";
  const showRefetchButton = canResendProfiles && (isFailed || isVerified);
  const value = (isFailed && profile.detail) || DEFAULT_EMPTY_CELL_VALUE;

  const tooltip = generateErrorTooltip(value, profile);

  return (
    <div className={baseClass}>
      <TooltipTruncatedTextCell
        tooltipBreakOnWord
        tooltip={tooltip}
        value={value}
        // we dont want the default "w250" class so we pass in empty string
        classes=""
        className={
          isFailed || showRefetchButton
            ? `${baseClass}__failed-message`
            : undefined
        }
      />
      {showRefetchButton && (
        <RefetchButton isFetching={isLoading} onClick={onResendProfile} />
      )}
    </div>
  );
};

export default OSSettingsErrorCell;
