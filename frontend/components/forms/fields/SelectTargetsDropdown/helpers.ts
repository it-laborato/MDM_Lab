import {
  ISelectTargetsEntity,
  ISelectNode,
  ISelectLabel,
  ISelectTeam,
} from "interfaces/target";

export const isTargetNode = (
  target: ISelectTargetsEntity
): target is ISelectNode => {
  return target.target_type === "nodes";
};

export const isTargetLabel = (
  target: ISelectTargetsEntity
): target is ISelectLabel => {
  return target.target_type === "labels";
};

export const isTargetTeam = (
  target: ISelectTargetsEntity
): target is ISelectTeam => {
  return target.target_type === "teams";
};
