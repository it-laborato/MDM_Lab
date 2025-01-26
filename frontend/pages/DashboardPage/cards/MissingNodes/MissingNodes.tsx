import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import SummaryTile from "../NodesSummary/SummaryTile";

const baseClass = "nodes-missing";

interface INodeSummaryProps {
  missingCount: number;
  isLoadingNodes: boolean;
  showNodesUI: boolean;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
}

const MissingNodes = ({
  missingCount,
  isLoadingNodes,
  showNodesUI,
  selectedPlatformLabelId,
  currentTeamId,
}: INodeSummaryProps): JSX.Element => {
  // build the manage nodes URL filtered by missing and platform
  const queryParams = {
    status: "missing",
    team_id: currentTeamId,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  return (
    <div className={baseClass}>
      <SummaryTile
        iconName="missing-nodes"
        count={missingCount}
        isLoading={isLoadingNodes}
        showUI={showNodesUI}
        title="Missing nodes"
        tooltip="Nodes that have not been online in 30 days or more."
        path={path}
      />
    </div>
  );
};

export default MissingNodes;
