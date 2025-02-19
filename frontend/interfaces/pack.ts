import PropTypes from "prop-types";
import { INode } from "./node";
import { ILabel } from "./label";
import { ITeam } from "./team";
import { ITarget } from "./target";

export default PropTypes.shape({
  created_at: PropTypes.string,
  updated_at: PropTypes.string,
  id: PropTypes.number,
  name: PropTypes.string,
  description: PropTypes.string,
  type: PropTypes.string,
  disabled: PropTypes.bool,
  query_count: PropTypes.number,
  total_node_count: PropTypes.number,
  node_ids: PropTypes.arrayOf(PropTypes.number),
  label_ids: PropTypes.arrayOf(PropTypes.number),
  team_ids: PropTypes.arrayOf(PropTypes.number),
});

export interface IStoredPacksResponse {
  packs: IPack[];
}

export interface IStoredPackResponse {
  pack: IPack;
}

export interface IPack {
  created_at: string;
  updated_at: string;
  id: number;
  name: string;
  description: string;
  type: string;
  disabled?: boolean;
  query_count: number;
  total_nodes_count: number;
  nodes: INode[];
  node_ids: number[];
  labels: ILabel[];
  label_ids: number[];
  teams: ITeam[];
  team_ids: number[];
}

export interface IUpdatePack {
  name?: string;
  description?: string;
  disabled?: boolean;
  targets?: ITarget[];
}

export interface IEditPackFormData {
  name: string;
  description: string;
  targets: ITarget[];
}
