import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import React from "react";
import { Link } from "react-router";

const baseClass = "team-node-expiry-toggle";

interface ITeamNodeExpiryToggle {
  globalNodeExpiryEnabled: boolean;
  globalNodeExpiryWindow?: number;
  teamExpiryEnabled: boolean;
  setTeamExpiryEnabled: (value: boolean) => void;
}

const TeamNodeExpiryToggle = ({
  globalNodeExpiryEnabled,
  globalNodeExpiryWindow,
  teamExpiryEnabled,
  setTeamExpiryEnabled,
}: ITeamNodeExpiryToggle) => {
  const renderHelpText = () =>
    // this will never be rendered while globalNodeExpiryWindow is undefined
    globalNodeExpiryEnabled ? (
      <div className="help-text">
        Node expiry is globally enabled in organization settings. By default,
        nodes expire after {globalNodeExpiryWindow} days.{" "}
        {!teamExpiryEnabled && (
          <Link
            to=""
            onClick={(e: React.MouseEvent) => {
              e.preventDefault();
              setTeamExpiryEnabled(true);
            }}
            className={`${baseClass}__add-custom-window`}
          >
            <>
              Add custom expiry window
              <Icon name="chevron-right" color="core-mdmlab-blue" size="small" />
            </>
          </Link>
        )}
      </div>
    ) : (
      <></>
    );
  return (
    <div className={`${baseClass}`}>
      <Checkbox
        name="enableNodeExpiry"
        onChange={setTeamExpiryEnabled}
        value={teamExpiryEnabled || globalNodeExpiryEnabled} // Still shows checkmark if global expiry is enabled though the checkbox will be disabled.
        disabled={globalNodeExpiryEnabled}
        helpText={renderHelpText()}
        tooltipContent={
          <>
            When enabled, allows automatic cleanup of
            <br />
            nodes that have not communicated with Mdmlab in
            <br />
            the number of days specified in the{" "}
            <strong>
              Node expiry
              <br />
              window
            </strong>{" "}
            setting.{" "}
            <em>
              (Default: <strong>Off</strong>)
            </em>
          </>
        }
      >
        Enable node expiry
      </Checkbox>
    </div>
  );
};

export default TeamNodeExpiryToggle;
