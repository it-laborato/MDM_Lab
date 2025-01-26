import React, { useCallback, useContext, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import nodeAPI, {
  IGetNodeSoftwareResponse,
  INodeSoftwareQueryKey,
} from "services/entities/nodes";
import deviceAPI, {
  IDeviceSoftwareQueryKey,
  IGetDeviceSoftwareResponse,
} from "services/entities/device_user";
import { INodeSoftware, ISoftware } from "interfaces/software";
import { NodePlatform, isIPadOrIPhone } from "interfaces/platform";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Card from "components/Card/Card";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import { generateSoftwareTableHeaders as generateNodeSoftwareTableConfig } from "./NodeSoftwareTableConfig";
import { generateSoftwareTableHeaders as generateDeviceSoftwareTableConfig } from "./DeviceSoftwareTableConfig";
import NodeSoftwareTable from "./NodeSoftwareTable";
import { getInstallErrorMessage, getUninstallErrorMessage } from "./helpers";

const baseClass = "software-card";

export interface ITableSoftware extends Omit<ISoftware, "vulnerabilities"> {
  vulnerabilities: string[]; // for client-side search purposes, we only want an array of cve strings
}

interface INodeSoftwareProps {
  /** This is the node id or the device token */
  id: number | string;
  platform?: NodePlatform;
  softwareUpdatedAt?: string;
  nodeCanWriteSoftware: boolean;
  router: InjectedRouter;
  queryParams: ReturnType<typeof parseNodeSoftwareQueryParams>;
  pathname: string;
  nodeTeamId: number;
  onShowSoftwareDetails: (software: INodeSoftware) => void;
  isSoftwareEnabled?: boolean;
  nodeScriptsEnabled?: boolean;
  isMyDevicePage?: boolean;
  nodeMDMEnrolled?: boolean;
}

const DEFAULT_SEARCH_QUERY = "";
const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 20;

export const parseNodeSoftwareQueryParams = (queryParams: {
  page?: string;
  query?: string;
  order_key?: string;
  order_direction?: "asc" | "desc";
  vulnerable?: string;
  available_for_install?: string;
}) => {
  const searchQuery = queryParams?.query ?? DEFAULT_SEARCH_QUERY;
  const sortHeader = queryParams?.order_key ?? DEFAULT_SORT_HEADER;
  const sortDirection = queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION;
  const page = queryParams?.page
    ? parseInt(queryParams.page, 10)
    : DEFAULT_PAGE;
  const pageSize = DEFAULT_PAGE_SIZE;
  const vulnerable = queryParams.vulnerable === "true";
  const availableForInstall = queryParams.available_for_install === "true";

  return {
    page,
    query: searchQuery,
    order_key: sortHeader,
    order_direction: sortDirection,
    per_page: pageSize,
    vulnerable,
    available_for_install: availableForInstall,
  };
};

const NodeSoftware = ({
  id,
  platform,
  softwareUpdatedAt,
  nodeCanWriteSoftware,
  nodeScriptsEnabled,
  router,
  queryParams,
  pathname,
  nodeTeamId = 0,
  onShowSoftwareDetails,
  isSoftwareEnabled = false,
  isMyDevicePage = false,
  nodeMDMEnrolled,
}: INodeSoftwareProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const vulnFilterAndNotSupported =
    isIPadOrIPhone(platform ?? "") && queryParams.vulnerable;
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);

  // disables install/uninstall actions after click
  const [softwareIdActionPending, setSoftwareIdActionPending] = useState<
    number | null
  >(null);

  const {
    data: nodeSoftwareRes,
    isLoading: nodeSoftwareLoading,
    isError: nodeSoftwareError,
    isFetching: nodeSoftwareFetching,
    refetch: refetchNodeSoftware,
  } = useQuery<
    IGetNodeSoftwareResponse,
    AxiosError,
    IGetNodeSoftwareResponse,
    INodeSoftwareQueryKey[]
  >(
    [
      {
        scope: "node_software",
        id: id as number,
        softwareUpdatedAt,
        ...queryParams,
      },
    ],
    ({ queryKey }) => {
      return nodeAPI.getNodeSoftware(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled:
        isSoftwareEnabled && !isMyDevicePage && !vulnFilterAndNotSupported,
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const {
    data: deviceSoftwareRes,
    isLoading: deviceSoftwareLoading,
    isError: deviceSoftwareError,
    isFetching: deviceSoftwareFetching,
    refetch: refetchDeviceSoftware,
  } = useQuery<
    IGetDeviceSoftwareResponse,
    AxiosError,
    IGetDeviceSoftwareResponse,
    IDeviceSoftwareQueryKey[]
  >(
    [
      {
        scope: "device_software",
        id: id as string,
        softwareUpdatedAt,
        ...queryParams,
      },
    ],
    ({ queryKey }) => deviceAPI.getDeviceSoftware(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isSoftwareEnabled && isMyDevicePage, // if disabled, we'll always show a generic "No software detected" message. No DUP for iPad/iPhone
      keepPreviousData: true,
      staleTime: 7000,
    }
  );

  const refetchSoftware = useMemo(
    () => (isMyDevicePage ? refetchDeviceSoftware : refetchNodeSoftware),
    [isMyDevicePage, refetchDeviceSoftware, refetchNodeSoftware]
  );

  const userHasSWWritePermission = Boolean(
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer
  );

  const installNodeSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setSoftwareIdActionPending(softwareId);
      try {
        await nodeAPI.installNodeSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          "Software is installing or will install when the node comes online."
        );
      } catch (e) {
        renderFlash("error", getInstallErrorMessage(e));
      }
      setSoftwareIdActionPending(null);
      refetchSoftware();
    },
    [id, renderFlash, refetchSoftware]
  );

  const uninstallNodeSoftwarePackage = useCallback(
    async (softwareId: number) => {
      setSoftwareIdActionPending(softwareId);
      try {
        await nodeAPI.uninstallNodeSoftwarePackage(id as number, softwareId);
        renderFlash(
          "success",
          <>
            Software is uninstalling or will uninstall when the node comes
            online. To see details, go to <b>Details &gt; Activity</b>.
          </>
        );
      } catch (e) {
        renderFlash("error", getUninstallErrorMessage(e));
      }
      setSoftwareIdActionPending(null);
      refetchSoftware();
    },
    [id, renderFlash, refetchSoftware]
  );

  const onSelectAction = useCallback(
    (software: INodeSoftware, action: string) => {
      switch (action) {
        case "install":
          installNodeSoftwarePackage(software.id);
          break;
        case "uninstall":
          uninstallNodeSoftwarePackage(software.id);
          break;
        case "showDetails":
          onShowSoftwareDetails?.(software);
          break;
        default:
          break;
      }
    },
    [
      installNodeSoftwarePackage,
      onShowSoftwareDetails,
      uninstallNodeSoftwarePackage,
    ]
  );

  const tableConfig = useMemo(() => {
    return isMyDevicePage
      ? generateDeviceSoftwareTableConfig()
      : generateNodeSoftwareTableConfig({
          userHasSWWritePermission,
          nodeScriptsEnabled,
          nodeCanWriteSoftware,
          nodeMDMEnrolled,
          softwareIdActionPending,
          router,
          teamId: nodeTeamId,
          onSelectAction,
        });
  }, [
    isMyDevicePage,
    router,
    softwareIdActionPending,
    userHasSWWritePermission,
    nodeScriptsEnabled,
    onSelectAction,
    nodeTeamId,
    nodeCanWriteSoftware,
    nodeMDMEnrolled,
  ]);

  const isLoading = isMyDevicePage
    ? deviceSoftwareLoading
    : nodeSoftwareLoading;

  const isError = isMyDevicePage ? deviceSoftwareError : nodeSoftwareError;

  const data = isMyDevicePage ? deviceSoftwareRes : nodeSoftwareRes;

  const getNodeSoftwareFilterFromQueryParams = () => {
    const { vulnerable, available_for_install } = queryParams;
    if (available_for_install) {
      return "installableSoftware";
    }
    if (vulnerable) {
      return "vulnerableSoftware";
    }
    return "allSoftware";
  };

  const renderNodeSoftware = () => {
    if (isLoading) {
      return <Spinner />;
    }
    // will never be the case - to handle `platform` typing discrepancy with DeviceUserPage
    if (!platform) {
      return null;
    }
    return (
      <>
        {isError && <DataError />}
        {!isError && (
          <NodeSoftwareTable
            isLoading={
              isMyDevicePage ? deviceSoftwareFetching : nodeSoftwareFetching
            }
            // this could be cleaner, however, we are going to revert this commit anyway once vulns are
            // supported for iPad/iPhone, by the end of next sprint
            data={
              vulnFilterAndNotSupported
                ? ({
                    count: 0,
                    meta: {
                      has_next_results: false,
                      has_previous_results: false,
                    },
                  } as IGetNodeSoftwareResponse)
                : data
            } // eshould be mpty for iPad/iPhone since API call is disabled, but to be sure to trigger empty state
            platform={platform}
            router={router}
            tableConfig={tableConfig}
            sortHeader={queryParams.order_key}
            sortDirection={queryParams.order_direction}
            searchQuery={queryParams.query}
            page={queryParams.page}
            pagePath={pathname}
            nodeSoftwareFilter={getNodeSoftwareFilterFromQueryParams()}
            pathPrefix={pathname}
            // for my device software details modal toggling
            isMyDevicePage={isMyDevicePage}
            onShowSoftwareDetails={onShowSoftwareDetails}
          />
        )}
      </>
    );
  };

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xxlarge"
      includeShadow
      className={`${baseClass} ${isMyDevicePage ? "device-software" : ""}`}
    >
      <div className={`card-header`}>Software</div>
      {isMyDevicePage && (
        <div className={`card-subheader`}>
          Software installed on your device.
        </div>
      )}
      {renderNodeSoftware()}
    </Card>
  );
};

export default React.memo(NodeSoftware);
