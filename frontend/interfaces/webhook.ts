import PropTypes from "prop-types";

export default PropTypes.shape({
  destination_url: PropTypes.string,
  policy_ids: PropTypes.arrayOf(PropTypes.number),
  enable_failing_policies_webhook: PropTypes.bool,
  node_batch_size: PropTypes.number,
});

export interface IWebhookNodeStatus {
  enable_node_status_webhook?: boolean;
  destination_url?: string;
  node_percentage?: number;
  days_count?: number;
}
export interface IWebhookFailingPolicies {
  destination_url?: string;
  policy_ids?: number[];
  enable_failing_policies_webhook?: boolean;
  node_batch_size?: number;
}

export interface IWebhookSoftwareVulnerabilities {
  destination_url?: string;
  enable_vulnerabilities_webhook?: boolean;
  node_batch_size?: number;
}

export interface IWebhookActivities {
  enable_activities_webhook: boolean;
  destination_url: string;
}

export type IWebhook =
  | IWebhookNodeStatus
  | IWebhookFailingPolicies
  | IWebhookSoftwareVulnerabilities
  | IWebhookActivities;
