import React from "react";
import classnames from "classnames";

// @ts-ignore
import MdmlabIcon from "components/icons/MdmlabIcon";

import { ISelectTargetsEntity } from "interfaces/target";
import { isTargetLabel, isTargetNode } from "../helpers";

const baseClass = "target-option";

interface ITargetIconProps {
  target: ISelectTargetsEntity;
}

const TargetIcon = ({ target }: ITargetIconProps): JSX.Element => {
  const iconName = (): string => {
    if (isTargetLabel(target)) {
      return target.name === "All Nodes" ? "all-nodes" : "label";
    }
    if (isTargetNode(target)) {
      return target.platform === "darwin" ? "apple" : target.platform;
    }
    return "";
  };

  const targetClasses = classnames(`${baseClass}__icon`, {
    [`${baseClass}__icon--${
      isTargetNode(target) && target.status
    }`]: isTargetNode(target),
  });

  return <MdmlabIcon name={iconName()} className={targetClasses} />;
};

export default TargetIcon;
