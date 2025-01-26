import React from "react";
import classnames from "classnames";

import { ISelectTargetsEntity } from "interfaces/target";
// @ts-ignore
import TargetIcon from "./TargetIcon";
import { isTargetNode, isTargetLabel, isTargetTeam } from "../helpers";

const baseClass = "target-option";

interface ITargetOptionProps {
  onMoreInfoClick: (
    target: ISelectTargetsEntity
  ) => (event: React.MouseEvent) => void;
  onSelect: (target: ISelectTargetsEntity, event: React.MouseEvent) => void;
  target: ISelectTargetsEntity;
}

const TargetOption = ({
  onMoreInfoClick,
  onSelect,
  target,
}: ITargetOptionProps): JSX.Element => {
  const handleSelect = (evt: React.MouseEvent) => {
    return onSelect(target, evt);
  };

  const renderTargetDetail = () => {
    if (isTargetNode(target)) {
      const { primary_ip: nodeIpAddress } = target;

      if (!nodeIpAddress) {
        return null;
      }

      return (
        <span>
          <span className={`${baseClass}__ip`}>{nodeIpAddress}</span>
        </span>
      );
    }

    if (isTargetTeam(target) || isTargetLabel(target)) {
      return (
        <span className={`${baseClass}__count`}>{target.count} nodes</span>
      );
    }

    return <></>;
  };

  const { display_text: displayText, target_type: targetType } = target;
  const wrapperClassName = classnames(`${baseClass}__wrapper`, {
    "is-team": targetType === "teams",
    "is-label": targetType === "labels",
    "is-node": targetType === "nodes",
  });

  return (
    <div className={wrapperClassName}>
      <button
        className={`button button--unstyled ${baseClass}__target-content`}
        onClick={onMoreInfoClick(target)}
      >
        <div>
          <TargetIcon target={target} />
          <span className={`${baseClass}__label-label`}>
            {displayText !== "All Nodes" ? displayText : "All nodes"}
          </span>
        </div>
        {renderTargetDetail()}
      </button>
      <button
        className={`button button--unstyled ${baseClass}__add-btn`}
        onClick={handleSelect}
      />
    </div>
  );
};

export default TargetOption;
