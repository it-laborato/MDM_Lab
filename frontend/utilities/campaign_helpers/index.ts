import { ICampaign, ICampaignState } from "interfaces/campaign";
import { INode } from "interfaces/node";
import { useContext } from "react";
import { NotificationContext } from "context/notification";

interface IResult {
  type: "result";
  data: {
    distributed_query_execution_id: number;
    error: string | null;
    node: INode;
    rows: Record<string, unknown>[];
  };
}

interface IStatus {
  type: "status";
  data: {
    actual_results: number;
    expected_result: number;
    status: string;
  };
}

interface ITotals {
  type: "totals";
  data: {
    count: number;
    missing_in_action: number;
    offline: number;
    online: number;
  };
}

interface IError {
  type: "error";
  data: string;
}

type ISocketData = IResult | IStatus | ITotals | IError;

const updateCampaignStateFromTotals = (
  campaign: ICampaign,
  { data: totals }: ITotals
) => {
  return {
    campaign: { ...campaign, totals },
  };
};

const updateCampaignStateFromResults = (
  campaign: ICampaign,
  { data }: IResult
) => {
  const {
    errors = [],
    nodes = [],
    nodes_count: nodesCount = { total: 0, failed: 0, successful: 0 },
    query_results: queryResults = [],
  } = campaign;
  const { error, node, rows = [] } = data;

  let newErrors;
  let newNodes;
  let newNodesCount;

  if (error || error === "") {
    const newFailed = nodesCount.failed + 1;
    const newTotal = nodesCount.successful + newFailed;

    newErrors = errors.concat([
      {
        node_display_name: node?.display_name,
        osquery_version: node?.osquery_version,
        error:
          error ||
          // Nodes with osquery version below 4.4.0 receive an empty error message
          // when the live query fails so we create our own message.
          "Error details require osquery 4.4.0+ (Launcher does not provide error details)",
      },
    ]);
    newNodesCount = {
      successful: nodesCount.successful,
      failed: newFailed,
      total: newTotal,
    };
    newNodes = nodes;
  } else {
    const newSuccessful = nodesCount.successful + 1;
    const newTotal = nodesCount.failed + newSuccessful;

    newErrors = [...errors];
    newNodesCount = {
      successful: newSuccessful,
      failed: nodesCount.failed,
      total: newTotal,
    };
    const newNode = { ...node, query_results: rows };
    newNodes = nodes.concat(newNode);
  }

  return {
    campaign: {
      ...campaign,
      errors: newErrors,
      nodes: newNodes,
      nodes_count: newNodesCount,
      query_results: [...queryResults, ...rows],
    },
  };
};

const updateCampaignStateFromStatus = (
  campaign: ICampaign,
  { data: { status } }: IStatus
) => {
  return {
    campaign: { ...campaign, status },
    queryIsRunning: status !== "finished",
  };
};

export const updateCampaignState = (socketData: ISocketData) => {
  return ({ campaign }: ICampaignState) => {
    const { renderFlash } = useContext(NotificationContext);
    switch (socketData.type) {
      case "totals":
        return updateCampaignStateFromTotals(campaign, socketData);
      case "result":
        return updateCampaignStateFromResults(campaign, socketData);
      case "status":
        return updateCampaignStateFromStatus(campaign, socketData);
      case "error":
        if (socketData.data.includes("unexpected exit in receiveMessages")) {
          const campaignID = socketData.data.substring(
            socketData.data.indexOf("=") + 1
          );
          renderFlash(
            "error",
            `Mdmlab's connection to Redis failed (campaign ID ${campaignID}). If this issue persists, please contact your administrator.`
          );
        }
        return { campaign };
      default:
        return { campaign };
    }
  };
};

export default { updateCampaignState };
