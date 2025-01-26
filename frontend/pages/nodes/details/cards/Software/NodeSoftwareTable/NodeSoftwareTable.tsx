import React, { useCallback } from "react";
import { InjectedRouter } from "react-router";

import { IGetNodeSoftwareResponse } from "services/entities/nodes";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";
import { getNextLocationPath } from "utilities/helpers";
import { QueryParams } from "utilities/url";

import { INodeSoftwareDropdownFilterVal } from "pages/SoftwarePage/SoftwareTitles/SoftwareTable/helpers";

import {
  ApplePlatform,
  APPLE_PLATFORM_DISPLAY_NAMES,
  NodePlatform,
  isIPadOrIPhone,
} from "interfaces/platform";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import EmptySoftwareTable from "pages/SoftwarePage/components/EmptySoftwareTable";
import TableCount from "components/TableContainer/TableCount";
import { VulnsNotSupported } from "pages/SoftwarePage/components/SoftwareVulnerabilitiesTable/SoftwareVulnerabilitiesTable";
import { Row } from "react-table";
import { INodeSoftware } from "interfaces/software";

const DEFAULT_PAGE_SIZE = 20;

const baseClass = "node-software-table";

const DROPDOWN_OPTIONS = [
  {
    disabled: false,
    label: "All software",
    value: "allSoftware",
    helpText: "All software installed on your nodes.",
  },
  {
    disabled: false,
    label: "Vulnerable software",
    value: "vulnerableSoftware",
    helpText:
      "All software installed on your nodes with detected vulnerabilities.",
  },
  {
    disabled: false,
    label: "Available for install",
    value: "installableSoftware",
    helpText: "Software that can be installed on your nodes.",
  },
];

interface INodeSoftwareRowProps extends Row {
  original: INodeSoftware;
}
interface INodeSoftwareTableProps {
  tableConfig: any; // TODO: type
  data?: IGetNodeSoftwareResponse | IGetDeviceSoftwareResponse;
  platform: NodePlatform;
  isLoading: boolean;
  router: InjectedRouter;
  sortHeader: string;
  sortDirection: "asc" | "desc";
  searchQuery: string;
  page: number;
  pagePath: string;
  routeTemplate?: string;
  pathPrefix: string;
  nodeSoftwareFilter: INodeSoftwareDropdownFilterVal;
  isMyDevicePage?: boolean;
  onShowSoftwareDetails: (software: INodeSoftware) => void;
}

const NodeSoftwareTable = ({
  tableConfig,
  data,
  platform,
  isLoading,
  router,
  sortHeader,
  sortDirection,
  searchQuery,
  page,
  pagePath,
  routeTemplate,
  pathPrefix,
  nodeSoftwareFilter,
  isMyDevicePage,
  onShowSoftwareDetails,
}: INodeSoftwareTableProps) => {
  const handleFilterDropdownChange = useCallback(
    (selectedFilter: SingleValue<CustomOptionType>) => {
      const newParams: QueryParams = {
        query: searchQuery,
        order_key: sortHeader,
        order_direction: sortDirection,
        page: 0,
      };

      // mutually exclusive
      if (selectedFilter?.value === "installableSoftware") {
        newParams.available_for_install = true.toString();
      } else if (selectedFilter?.value === "vulnerableSoftware") {
        newParams.vulnerable = true.toString();
      }

      const nextPath = getNextLocationPath({
        pathPrefix,
        routeTemplate,
        queryParams: newParams,
      });
      const prevYScroll = window.scrollY;
      setTimeout(() => {
        window.scroll({
          top: prevYScroll,
          behavior: "smooth",
        });
      }, 0);
      router.replace(nextPath);
    },
    [pathPrefix, routeTemplate, router, searchQuery, sortDirection, sortHeader]
  );

  const memoizedFilterDropdown = useCallback(() => {
    return (
      <DropdownWrapper
        name="node-software-filter"
        value={nodeSoftwareFilter}
        className={`${baseClass}__software-filter`}
        options={DROPDOWN_OPTIONS}
        onChange={handleFilterDropdownChange}
        tableFilter
      />
    );
  }, [handleFilterDropdownChange, nodeSoftwareFilter]);

  const determineQueryParamChange = useCallback(
    (newTableQuery: ITableQueryData) => {
      const changedEntry = Object.entries(newTableQuery).find(([key, val]) => {
        switch (key) {
          case "searchQuery":
            return val !== searchQuery;
          case "sortDirection":
            return val !== sortDirection;
          case "sortHeader":
            return val !== sortHeader;
          case "pageIndex":
            return val !== page;
          default:
            return false;
        }
      });
      return changedEntry?.[0] ?? "";
    },
    [page, searchQuery, sortDirection, sortHeader]
  );

  const generateNewQueryParams = useCallback(
    (newTableQuery: ITableQueryData, changedParam: string) => {
      const newQueryParam: QueryParams = {
        query: newTableQuery.searchQuery,
        order_direction: newTableQuery.sortDirection,
        order_key: newTableQuery.sortHeader,
        page: changedParam === "pageIndex" ? newTableQuery.pageIndex : 0,
      };

      if (nodeSoftwareFilter === "vulnerableSoftware") {
        newQueryParam.vulnerable = "true";
      } else if (nodeSoftwareFilter === "installableSoftware") {
        newQueryParam.available_for_install = "true";
      }

      return newQueryParam;
    },
    [nodeSoftwareFilter]
  );

  // TODO: Look into useDebounceCallback with dependencies
  const onQueryChange = useCallback(
    async (newTableQuery: ITableQueryData) => {
      // we want to determine which query param has changed in order to
      // reset the page index to 0 if any other param has changed.
      const changedParam = determineQueryParamChange(newTableQuery);

      // if nothing has changed, don't update the route. this can happen when
      // this handler is called on the inital render. Can also happen when
      // the filter dropdown is changed. That is handled on the onChange handler
      // for the dropdown.
      if (changedParam === "") return;

      const newRoute = getNextLocationPath({
        pathPrefix: pagePath,
        routeTemplate: "",
        queryParams: generateNewQueryParams(newTableQuery, changedParam),
      });

      router.replace(newRoute);
    },
    [determineQueryParamChange, pagePath, generateNewQueryParams, router]
  );

  const count = data?.count || data?.software?.length || 0;
  const isSoftwareNotDetected = count === 0 && searchQuery === "";

  const memoizedSoftwareCount = useCallback(() => {
    if (isSoftwareNotDetected) {
      return null;
    }

    return <TableCount name="items" count={count} />;
  }, [count, isSoftwareNotDetected]);

  const memoizedEmptyComponent = useCallback(() => {
    const vulnFilterAndNotSupported =
      isIPadOrIPhone(platform) && nodeSoftwareFilter === "vulnerableSoftware";
    return vulnFilterAndNotSupported ? (
      <VulnsNotSupported
        platformText={APPLE_PLATFORM_DISPLAY_NAMES[platform as ApplePlatform]}
      />
    ) : (
      <EmptySoftwareTable noSearchQuery={searchQuery === ""} />
    );
  }, [nodeSoftwareFilter, platform, searchQuery]);

  // Determines if a user should be able to filter or search in the table
  const hasData = data && data.software.length > 0;
  const hasQuery = searchQuery !== "";
  const hasSoftwareFilter = nodeSoftwareFilter !== "allSoftware";

  const showFilterHeaders = hasData || hasQuery || hasSoftwareFilter;

  const onClickMyDeviceRow = useCallback(
    (row: INodeSoftwareRowProps) => {
      onShowSoftwareDetails(row.original);
    },
    [onShowSoftwareDetails]
  );

  return (
    <div className={baseClass}>
      <TableContainer
        renderCount={memoizedSoftwareCount}
        columnConfigs={tableConfig}
        data={data?.software || []}
        isLoading={isLoading}
        defaultSortHeader={sortHeader}
        defaultSortDirection={sortDirection}
        defaultSearchQuery={searchQuery}
        defaultPageIndex={page}
        disableNextPage={data?.meta.has_next_results === false}
        pageSize={DEFAULT_PAGE_SIZE}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={memoizedEmptyComponent}
        customControl={
          !isMyDevicePage && showFilterHeaders
            ? memoizedFilterDropdown
            : undefined
        }
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable={showFilterHeaders}
        manualSortBy
        keyboardSelectableRows
        // my device page row clickability
        disableMultiRowSelect={isMyDevicePage}
        onClickRow={isMyDevicePage ? onClickMyDeviceRow : undefined}
      />
    </div>
  );
};

export default NodeSoftwareTable;
