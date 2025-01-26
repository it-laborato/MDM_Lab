import PropTypes from "prop-types";
import { INode } from "./node";

export default PropTypes.shape({
  nodes_count: PropTypes.shape({
    total: PropTypes.number,
    successful: PropTypes.number, // Does not include ChromeOS results that are partially successful
    failed: PropTypes.number,
  }),
  id: PropTypes.number,
  online: PropTypes.number,
});

export interface ICampaignError {
  node_display_name: string;
  osquery_version: string;
  error: string;
}

export interface ICampaign {
  // upstream websocket and services methods return any
  // so narrower typing at this level is not actually guaranteed
  Metrics?: {
    [key: string]: any;
  };
  created_at: string;
  errors: ICampaignError[];
  nodes: INode[];
  nodes_count: {
    total: number;
    successful: number; // Does not include ChromeOS results that are partially successful
    failed: number;
  };
  id: number;
  query_id: number;
  query_results: Record<string, unknown>[];
  status: string;
  totals: {
    count: number;
    missing_in_action: number;
    offline: number;
    online: number;
  };
  updated_at: string;
  user_id: number;
}

// TODO: review use of ICampaignState to see if legacy code can be removed
export interface ICampaignState {
  campaign: ICampaign;
  observerShowSql: boolean;
  queryIsRunning: boolean;
  queryPosition: {
    [key: string]: any;
  };
  queryResultsToggle: any;
  runQueryMilliseconds: number;
  selectRelatedNodeTarget: boolean;
  targetsCount: number;
  targetsError: any;
}
