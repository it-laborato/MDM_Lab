import React, { useCallback } from "react";
import PATHS from "router/paths";

import { PLATFORM_NAME_TO_LABEL_NAME } from "pages/DashboardPage/helpers";
import DataError from "components/DataError";
import { INodeSummary } from "interfaces/node_summary";
import { PlatformValueOptions } from "utilities/constants";

import SummaryTile from "./SummaryTile";

const baseClass = "nodes-summary";

interface INodeSummaryProps {
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  chromeCount: number;
  iosCount: number;
  ipadosCount: number;
  isLoadingNodesSummary: boolean;
  builtInLabels?: INodeSummary["builtin_labels"];
  showNodesUI: boolean;
  errorNodes: boolean;
  selectedPlatform?: PlatformValueOptions;
}

const NodesSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  chromeCount,
  iosCount,
  ipadosCount,
  isLoadingNodesSummary,
  builtInLabels,
  showNodesUI,
  errorNodes,
  selectedPlatform,
}: INodeSummaryProps): JSX.Element => {
  // Renders semi-transparent screen as node information is loading
  let opacity = { opacity: 0 };
  if (showNodesUI) {
    opacity = isLoadingNodesSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  const getBuiltinLabelId = useCallback(
    (platformName: keyof typeof PLATFORM_NAME_TO_LABEL_NAME) =>
      builtInLabels?.find(
        (builtin) => builtin.name === PLATFORM_NAME_TO_LABEL_NAME[platformName]
      )?.id,
    [builtInLabels]
  );

  // const renderMacCount = (teamId?: number) => {
  //   const macLabelId = getBuiltinLabelId("darwin");
  //
  //   if (isLoadingNodesSummary || macLabelId === undefined) {
  //     return <></>;
  //   }
  //
  //   return (
  //     <SummaryTile
  //       iconName="darwin"
  //       count={macCount}
  //       isLoading={isLoadingNodesSummary}
  //       showUI={showNodesUI}
  //       title={`macOS node${macCount === 1 ? "" : "s"}`}
  //       path={PATHS.MANAGE_HOSTS_LABEL(macLabelId).concat(
  //         teamId !== undefined ? `?team_id=${teamId}` : ""
  //       )}
  //     />
  //   );
  // };

  const renderWindowsCount = (teamId?: number) => {
    const windowsLabelId = getBuiltinLabelId("windows");

    if (isLoadingNodesSummary || windowsLabelId === undefined) {
      return <></>;
    }
    return (
      <SummaryTile
        iconName="windows"
        count={windowsCount}
        isLoading={isLoadingNodesSummary}
        showUI={showNodesUI}
        title={`Windows node${windowsCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(windowsLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  // const renderLinuxCount = (teamId?: number) => {
  //   const linuxLabelId = getBuiltinLabelId("linux");
  //
  //   if (isLoadingNodesSummary || linuxLabelId === undefined) {
  //     return <></>;
  //   }
  //   return (
  //     <SummaryTile
  //       iconName="linux"
  //       count={linuxCount}
  //       isLoading={isLoadingNodesSummary}
  //       showUI={showNodesUI}
  //       title={`Linux node${linuxCount === 1 ? "" : "s"}`}
  //       path={PATHS.MANAGE_HOSTS_LABEL(linuxLabelId).concat(
  //         teamId !== undefined ? `?team_id=${teamId}` : ""
  //       )}
  //     />
  //   );
  // };

  // const renderChromeCount = (teamId?: number) => {
  //   const chromeLabelId = getBuiltinLabelId("chrome");
  //
  //   if (isLoadingNodesSummary || chromeLabelId === undefined) {
  //     return <></>;
  //   }
  //
  //   return (
  //     <SummaryTile
  //       iconName="chrome"
  //       count={chromeCount}
  //       isLoading={isLoadingNodesSummary}
  //       showUI={showNodesUI}
  //       title={`Chromebook${chromeCount === 1 ? "" : "s"}`}
  //       path={PATHS.MANAGE_HOSTS_LABEL(chromeLabelId).concat(
  //         teamId !== undefined ? `?team_id=${teamId}` : ""
  //       )}
  //     />
  //   );
  // };

  // const renderIosCount = (teamId?: number) => {
  //   const iosLabelId = getBuiltinLabelId("ios");
  //
  //   if (isLoadingNodesSummary || iosLabelId === undefined) {
  //     return <></>;
  //   }
  //
  //   return (
  //     <SummaryTile
  //       iconName="iOS"
  //       count={iosCount}
  //       isLoading={isLoadingNodesSummary}
  //       showUI={showNodesUI}
  //       title={`iPhone${iosCount === 1 ? "" : "s"}`}
  //       path={PATHS.MANAGE_HOSTS_LABEL(iosLabelId).concat(
  //         teamId !== undefined ? `?team_id=${teamId}` : ""
  //       )}
  //     />
  //   );
  // };

  // const renderIpadosCount = (teamId?: number) => {
  //   const ipadosLabelId = getBuiltinLabelId("ipados");
  //
  //   if (isLoadingNodesSummary || ipadosLabelId === undefined) {
  //     return <></>;
  //   }
  //
  //   return (
  //     <SummaryTile
  //       iconName="iPadOS"
  //       count={ipadosCount}
  //       isLoading={isLoadingNodesSummary}
  //       showUI={showNodesUI}
  //       title={`iPad${ipadosCount === 1 ? "" : "s"}`}
  //       path={PATHS.MANAGE_HOSTS_LABEL(ipadosLabelId).concat(
  //         teamId !== undefined ? `?team_id=${teamId}` : ""
  //       )}
  //     />
  //   );
  // };

  const renderCounts = (teamId?: number) => {
    switch (selectedPlatform) {
      // case "darwin":
        // return renderMacCount(teamId);
      case "windows":
        return renderWindowsCount(teamId);
      // case "linux":
      //   return renderLinuxCount(teamId);
      // case "chrome":
      //   return renderChromeCount(teamId);
      // case "ios":
      //   return renderIosCount(teamId);
      // case "ipados":
      //   return renderIpadosCount(teamId);
      default:
        return (
          <>
            {renderWindowsCount(teamId)}
          </>
        );
    }
  };

  if (errorNodes && !isLoadingNodesSummary) {
    return <DataError card />;
  }

  return (
    <div
      className={`${baseClass} ${
        selectedPlatform !== "all" ? "single-platform" : ""
      }`}
      style={opacity}
    >
      {renderCounts(currentTeamId)}
    </div>
  );
};

export default NodesSummary;
