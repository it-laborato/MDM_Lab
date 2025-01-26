import PropTypes from "prop-types";
import nodeInterface, { INode } from "interfaces/node";
import labelInterface, { ILabel, ILabelSummary } from "interfaces/label";
import teamInterface, { ITeam } from "interfaces/team";

export default PropTypes.oneOfType([
  nodeInterface,
  labelInterface,
  teamInterface,
]);

export type ITarget = INode | ILabel | ITeam;
export interface ITargets {
  nodes: INode[];
  labels: ILabel[];
  teams: ITeam[];
}

export interface ITargetsAPIResponse {
  targets: ITargets;
  targets_count: number;
  targets_missing_in_action: number;
  targets_offline: number;
  targets_online: number;
}

export interface ISelectNode extends INode {
  target_type?: string;
}

export interface ISelectLabel extends ILabelSummary {
  target_type?: string;
  display_text?: string;
  query?: string;
  count?: number;
}

export interface ISelectTeam extends ITeam {
  target_type?: string;
  display_text?: string;
}

export type ISelectTargetsEntity = ISelectNode | ISelectLabel | ISelectTeam;

export interface ISelectedTargetsForApi {
  nodes: number[];
  labels: number[];
  teams: number[];
}

export interface ISelectedTargetsByType {
  nodes: INode[];
  labels: ILabel[];
  teams: ITeam[];
}

export interface IPackTargets {
  node_ids: (number | string)[];
  label_ids: (number | string)[];
  team_ids: (number | string)[];
}

// TODO: Also use for testing
export const DEFAULT_TARGETS: ITarget[] = [];

export const DEFAULT_TARGETS_BY_TYPE: ISelectedTargetsByType = {
  nodes: [],
  labels: [],
  teams: [],
};
