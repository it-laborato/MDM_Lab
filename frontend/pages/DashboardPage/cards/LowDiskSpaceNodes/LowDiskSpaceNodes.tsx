import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import SummaryTile from "../NodesSummary/SummaryTile";

const baseClass = "nodes-low-space";

interface INodeSummaryProps {
  lowDiskSpaceGb: number;
  lowDiskSpaceCount: number;
  isLoadingNodes: boolean;
  showNodesUI: boolean;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
  notSupported: boolean;
}

const LowDiskSpaceNodes = ({
  lowDiskSpaceGb,
  lowDiskSpaceCount,
  isLoadingNodes,
  showNodesUI,
  selectedPlatformLabelId,
  currentTeamId,
  notSupported = false, // default to supporting this feature
}: INodeSummaryProps): JSX.Element => {
  // build the manage nodes URL filtered by low disk space only
  // currently backend cannot filter by both low disk space and label
  const queryParams = {
    low_disk_space: lowDiskSpaceGb,
    team_id: currentTeamId,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  const tooltipText = notSupported
    ? "Disk space info is not available for Chromebooks."
    : `Nodes that have ${lowDiskSpaceGb} GB or less disk space available.`;

  return (
    <div className={baseClass}>
      <SummaryTile
        iconName="low-disk-space-nodes"
        count={lowDiskSpaceCount}
        isLoading={isLoadingNodes}
        showUI={showNodesUI}
        title="Low disk space nodes"
        tooltip={tooltipText}
        path={path}
        notSupported={notSupported}
      />
    </div>
  );
};

export default LowDiskSpaceNodes;
