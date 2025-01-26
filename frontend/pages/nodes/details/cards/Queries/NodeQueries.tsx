import React, { useCallback, useMemo } from "react";

import { IQueryStats } from "interfaces/query_stats";
import { SUPPORT_LINK } from "utilities/constants";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import Card from "components/Card";
import PATHS from "router/paths";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import {
  generateColumnConfigs,
  generateDataSet,
} from "./NodeQueriesTableConfig";

const baseClass = "node-queries-card";

interface INodeQueriesProps {
  nodeId: number;
  schedule?: IQueryStats[];
  nodePlatform: string;
  queryReportsDisabled?: boolean;
  router: InjectedRouter;
}

interface INodeQueriesRowProps extends Row {
  original: {
    id?: number;
    should_link_to_hqr?: boolean;
    nodeId?: number;
  };
}

const NodeQueries = ({
  nodeId,
  schedule,
  nodePlatform,
  queryReportsDisabled,
  router,
}: INodeQueriesProps): JSX.Element => {
  const renderEmptyQueriesTab = () => {
    if (nodePlatform === "chrome") {
      return (
        <EmptyTable
          header="Scheduled queries are not supported for this node"
          info={
            <>
              <span>Interested in collecting data from your Chromebooks? </span>
              <CustomLink
                url="https://www.mdmlabdm.com/contact"
                text="Let us know"
                newTab
              />
            </>
          }
        />
      );
    }

    if (nodePlatform === "ios" || nodePlatform === "ipados") {
      return (
        <EmptyTable
          header="Queries are not supported for this node"
          info={
            <>
              Interested in querying{" "}
              {nodePlatform === "ios" ? "iPhones" : "iPads"}?{" "}
              <CustomLink url={SUPPORT_LINK} text="Let us know" newTab />
            </>
          }
        />
      );
    }

    return (
      <EmptyTable
        header="No queries are scheduled to run on this node"
        info={
          <>
            Expecting to see queries? Try selecting <b>Refetch</b> to ask this
            node to report fresh vitals.
          </>
        }
      />
    );
  };

  const onSelectSingleRow = useCallback(
    (row: INodeQueriesRowProps) => {
      const { id: queryId, should_link_to_hqr } = row.original;

      if (!nodeId || !queryId || !should_link_to_hqr || queryReportsDisabled) {
        return;
      }
      router.push(`${PATHS.HOST_QUERY_REPORT(nodeId, queryId)}`);
    },
    [nodeId, queryReportsDisabled, router]
  );

  const tableData = useMemo(() => generateDataSet(schedule ?? []), [schedule]);

  const columnConfigs = useMemo(
    () => generateColumnConfigs(nodeId, queryReportsDisabled),
    [nodeId, queryReportsDisabled]
  );

  const renderNodeQueries = () => {
    if (
      !schedule ||
      !schedule.length ||
      nodePlatform === "chrome" ||
      nodePlatform === "ios" ||
      nodePlatform === "ipados"
    ) {
      return renderEmptyQueriesTab();
    }

    return (
      <div>
        <TableContainer
          columnConfigs={columnConfigs}
          data={tableData}
          onQueryChange={() => null}
          resultsTitle="queries"
          defaultSortHeader="query_name"
          defaultSortDirection="asc"
          showMarkAllPages={false}
          isAllPagesSelected={false}
          emptyComponent={() => <></>}
          disablePagination
          disableCount
          disableMultiRowSelect={!queryReportsDisabled} // Removes hover/click state if reports are disabled
          isLoading={false} // loading state handled at parent level
          onSelectSingleRow={onSelectSingleRow}
        />
      </div>
    );
  };

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
      className={baseClass}
    >
      <p className="card__header">Queries</p>
      {renderNodeQueries()}
    </Card>
  );
};

export default NodeQueries;
