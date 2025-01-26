import BackLink from "components/BackLink";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { browserHistory, InjectedRouter, Link } from "react-router";
import { Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import hqrAPI, { IGetHQRResponse } from "services/entities/node_query_report";
import queryAPI from "services/entities/queries";
import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import HQRTable from "./HQRTable";

const baseClass = "node-query-report";

interface INodeQueryReportProps {
  router: InjectedRouter;
  params: Params;
}

const NodeQueryReport = ({
  router,
  params: { node_id, query_id },
}: INodeQueryReportProps) => {
  const { config } = useContext(AppContext);
  const globalReportsDisabled = config?.server_settings.query_reports_disabled;
  const nodeId = Number(node_id);
  const queryId = Number(query_id);

  if (globalReportsDisabled) {
    router.push(PATHS.HOST_QUERIES(nodeId));
  }

  const [showQuery, setShowQuery] = useState(false);

  const {
    data: hqrResponse,
    isLoading: hqrLoading,
    error: hqrError,
  } = useQuery<IGetHQRResponse, Error>(
    [nodeId, queryId],
    () => hqrAPI.load(nodeId, queryId),
    {
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  const {
    isLoading: queryLoading,
    data: queryResponse,
    error: queryError,
  } = useQuery<IGetQueryResponse, Error, ISchedulableQuery>(
    ["query", queryId],
    () => queryAPI.load(queryId),

    {
      select: (data) => data.query,
      enabled: !!queryId,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  const isLoading = queryLoading || hqrLoading;

  const {
    node_name: nodeName,
    report_clipped: reportClipped,
    last_fetched: lastFetched,
    results,
  } = hqrResponse || {};

  // API response is nested this way to mirror that of the full Query Reports response (IQueryReport)
  const rows = results?.map((row) => row.columns) ?? [];

  const {
    name: queryName,
    description: queryDescription,
    query: querySQL,
    discard_data: queryDiscardData,
  } = queryResponse || {};

  // previous reroute can be done before API call, not this one, hence 2
  if (queryDiscardData) {
    router.push(PATHS.HOST_QUERIES(nodeId));
  }

  // Updates title that shows up on browser tabs
  if (queryName && nodeName) {
    // e.g., Discover TLS certificates (Rachel's MacBook Pro) | Nodes | Mdmlab
    document.title = `${queryName} (${nodeName}) |
   Nodes | ${DOCUMENT_TITLE_SUFFIX}`;
  } else {
    document.title = `Nodes | ${DOCUMENT_TITLE_SUFFIX}`;
  }

  const HQRHeader = useCallback(() => {
    const fullReportPath = PATHS.QUERY_DETAILS(queryId);
    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__header__row1`}>
          <BackLink
            text="Back to node details"
            path={PATHS.HOST_QUERIES(nodeId)}
          />
        </div>
        <div className={`${baseClass}__header__row2`}>
          {!hqrError && <h1 className="node-name">{nodeName}</h1>}
          <Link
            // to and onClick seem redundant
            to={fullReportPath}
            onClick={() => {
              browserHistory.push(fullReportPath);
            }}
            className={`${baseClass}__direction-link`}
          >
            <>
              <span>View full query report</span>
              <Icon name="chevron-right" color="core-mdmlab-blue" />
            </>
          </Link>
        </div>
      </div>
    );
  }, [queryId, nodeId, hqrError, nodeName]);

  return (
    <MainContent className={baseClass}>
      {isLoading ? (
        <Spinner />
      ) : (
        <>
          <HQRHeader />
          <HQRTable
            queryName={queryName}
            queryDescription={queryDescription}
            nodeName={nodeName}
            rows={rows}
            reportClipped={reportClipped}
            lastFetched={lastFetched}
            onShowQuery={() => setShowQuery(true)}
            isLoading={false}
          />
          {showQuery && (
            <ShowQueryModal
              query={querySQL}
              onCancel={() => setShowQuery(false)}
            />
          )}
        </>
      )}
    </MainContent>
  );
};

export default NodeQueryReport;
