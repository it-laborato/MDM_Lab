import React from "react";
import { noop } from "lodash";
import AceEditor from "react-ace";
import classnames from "classnames";

import { humanNodeMemory } from "utilities/helpers";
import MdmlabIcon from "components/icons/MdmlabIcon";
import PlatformIcon from "components/icons/PlatformIcon";
import { ISelectNode, ISelectLabel, ISelectTeam } from "interfaces/target";

import { isTargetNode, isTargetTeam, isTargetLabel } from "../helpers";

const baseClass = "target-details";

interface ITargetDetailsProps {
  target: ISelectNode | ISelectTeam | ISelectLabel; // Replace with Target
  className?: string;
  handleBackToResults?: () => void;
}

const TargetDetails = ({
  target,
  className = "",
  handleBackToResults = noop,
}: ITargetDetailsProps): JSX.Element => {
  const renderNode = (nodeTarget: ISelectNode) => {
    const {
      display_text: displayText,
      primary_mac: nodeMac,
      primary_ip: nodeIpAddress,
      memory,
      osquery_version: osqueryVersion,
      os_version: osVersion,
      platform,
      status,
    } = nodeTarget;
    const nodeBaseClass = "node-target";
    const isOnline = status === "online";
    const isOffline = status === "offline";
    const statusClassName = classnames(
      `${nodeBaseClass}__status`,
      { [`${nodeBaseClass}__status--is-online`]: isOnline },
      { [`${nodeBaseClass}__status--is-offline`]: isOffline }
    );

    return (
      <div className={`${nodeBaseClass} ${className}`}>
        <button
          className={`button button--unstyled ${nodeBaseClass}__back`}
          onClick={handleBackToResults}
        >
          <MdmlabIcon name="chevronleft" />
          Back
        </button>

        <p className={`${nodeBaseClass}__display-text`}>
          <MdmlabIcon name="single-node" className={`${nodeBaseClass}__icon`} />
          <span>{displayText}</span>
        </p>
        <p className={statusClassName}>
          {isOnline && (
            <MdmlabIcon
              name="success-check"
              className={`${nodeBaseClass}__icon ${nodeBaseClass}__icon--online`}
            />
          )}
          {isOffline && (
            <MdmlabIcon
              name="offline"
              className={`${nodeBaseClass}__icon ${nodeBaseClass}__icon--offline`}
            />
          )}
          <span>{status}</span>
        </p>
        <table className={`${baseClass}__table`}>
          <tbody>
            <tr>
              <th>Private IP address</th>
              <td>{nodeIpAddress}</td>
            </tr>
            <tr>
              <th>MAC address</th>
              <td>
                <span className={`${nodeBaseClass}__mac-address`}>
                  {nodeMac}
                </span>
              </td>
            </tr>
            <tr>
              <th>Platform</th>
              <td>
                <PlatformIcon name={platform} title={platform} />
                <span className={`${nodeBaseClass}__platform-text`}>
                  {" "}
                  {platform}
                </span>
              </td>
            </tr>
            <tr>
              <th>Operating system</th>
              <td>{osVersion}</td>
            </tr>
            <tr>
              <th>Osquery version</th>
              <td>{osqueryVersion}</td>
            </tr>
            <tr>
              <th>Memory</th>
              <td>{humanNodeMemory(memory)}</td>
            </tr>
          </tbody>
        </table>
      </div>
    );
  };

  const renderLabel = (labelTarget: ISelectLabel) => {
    const {
      count,
      description,
      display_text: displayText,
      query,
    } = labelTarget;

    const labelBaseClass = "label-target";
    return (
      <div className={`${labelBaseClass} ${className}`}>
        <button
          className={`button button--unstyled ${labelBaseClass}__back`}
          onClick={handleBackToResults}
        >
          <MdmlabIcon name="chevronleft" /> Back
        </button>

        <p className={`${labelBaseClass}__display-text`}>
          <MdmlabIcon name="label" fw className={`${labelBaseClass}__icon`} />
          <span>{displayText}</span>
        </p>

        <p className={`${labelBaseClass}__nodes`}>
          <span className={`${labelBaseClass}__nodes-count`}>
            <strong>{count}</strong>HOSTS
          </span>
        </p>

        <p className={`${labelBaseClass}__description`}>
          {description || "No Description"}
        </p>
        <div className={`${labelBaseClass}__editor`}>
          <AceEditor
            editorProps={{ $blockScrolling: Infinity }}
            mode="mdmlab"
            minLines={1}
            maxLines={20}
            name="label-query"
            readOnly
            setOptions={{ wrap: true }}
            showGutter={false}
            showPrintMargin={false}
            theme="mdmlab"
            value={query}
            width="100%"
            fontSize={14}
          />
        </div>
      </div>
    );
  };

  const renderTeam = (teamTarget: ISelectTeam) => {
    const { count, display_text: displayText } = teamTarget;
    const labelBaseClass = "label-target";

    return (
      <div className={`${labelBaseClass} ${className}`}>
        <p className={`${labelBaseClass}__display-text`}>
          <MdmlabIcon
            name="all-nodes"
            fw
            className={`${labelBaseClass}__icon`}
          />
          <span>{displayText}</span>
        </p>

        <p className={`${labelBaseClass}__nodes`}>
          <span className={`${labelBaseClass}__nodes-count`}>
            <strong>{count}</strong>HOSTS
          </span>
        </p>
      </div>
    );
  };

  if (!target) {
    return <></>;
  }

  if (isTargetNode(target)) {
    return renderNode(target);
  }

  if (isTargetLabel(target)) {
    return renderLabel(target);
  }

  if (isTargetTeam(target)) {
    return renderTeam(target);
  }
  return <></>;
};

export default TargetDetails;
