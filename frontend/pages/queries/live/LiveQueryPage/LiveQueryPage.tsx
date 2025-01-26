import React, { useState, useEffect, useContext, useCallback } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter, Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import useTeamIdParam from "hooks/useTeamIdParam";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { LIVE_QUERY_STEPS, DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import queryAPI from "services/entities/queries";
import nodeAPI from "services/entities/nodes";
import { INode, INodeResponse } from "interfaces/node";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import MainContent from "components/MainContent";
import SelectTargets from "components/LiveQuery/SelectTargets";

import RunQuery from "pages/queries/live/screens/RunQuery";

interface IRunQueryPageProps {
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
    query: { node_id: string; team_id?: string };
    search: string;
  };
}

const baseClass = "run-query-page";

const RunQueryPage = ({
  router,
  params: { id: paramsQueryId },
  location,
}: IRunQueryPageProps): JSX.Element => {
  const queryId = paramsQueryId ? parseInt(paramsQueryId, 10) : null;

  const { currentTeamId } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const handlePageError = useErrorHandler();
  const { config } = useContext(AppContext);
  const {
    editingExistingQuery,
    selectedQueryTargets,
    setSelectedQueryTargets,
    selectedQueryTargetsByType,
    setSelectedQueryTargetsByType,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
    setLastEditedQueryFrequency,
    setLastEditedQueryLoggingType,
    setLastEditedQueryMinOsqueryVersion,
    setLastEditedQueryPlatforms,
  } = useContext(QueryContext);

  const [queryParamNodesAdded, setQueryParamNodesAdded] = useState(false);
  const [step, setStep] = useState(LIVE_QUERY_STEPS[1]);
  const [targetedNodes, setTargetedNodes] = useState<INode[]>(
    selectedQueryTargetsByType.nodes
  );
  const [targetedLabels, setTargetedLabels] = useState<ILabel[]>(
    selectedQueryTargetsByType.labels
  );
  const [targetedTeams, setTargetedTeams] = useState<ITeam[]>(
    selectedQueryTargetsByType.teams
  );
  const [targetsTotalCount, setTargetsTotalCount] = useState(0);

  const disabledLiveQuery = config?.server_settings.live_query_disabled;

  // Reroute users out of live flow when live queries are globally disabled
  if (disabledLiveQuery) {
    queryId
      ? router.push(PATHS.QUERY_DETAILS(queryId, currentTeamId))
      : router.push(PATHS.NEW_QUERY(currentTeamId));
  }

  // disabled on page load so we can control the number of renders
  // else it will re-populate the context on occasion
  const { data: storedQuery } = useQuery<
    IGetQueryResponse,
    Error,
    ISchedulableQuery
  >(["query", queryId], () => queryAPI.load(queryId as number), {
    enabled: !!queryId && !editingExistingQuery,
    refetchOnWindowFocus: false,
    select: (data) => data.query,
    onSuccess: (returnedQuery) => {
      setLastEditedQueryId(returnedQuery.id);
      setLastEditedQueryName(returnedQuery.name);
      setLastEditedQueryDescription(returnedQuery.description);
      setLastEditedQueryBody(returnedQuery.query);
      setLastEditedQueryObserverCanRun(returnedQuery.observer_can_run);
      setLastEditedQueryFrequency(returnedQuery.interval);
      setLastEditedQueryPlatforms(returnedQuery.platform);
      setLastEditedQueryLoggingType(returnedQuery.logging);
      setLastEditedQueryMinOsqueryVersion(returnedQuery.min_osquery_version);
    },
    onError: (error) => handlePageError(error),
  });

  useQuery<INodeResponse, Error, INode>(
    "nodeFromURL",
    () =>
      nodeAPI.loadNodeDetails(parseInt(location.query.node_id as string, 10)),
    {
      enabled: !!location.query.node_id && !queryParamNodesAdded,
      select: (data: INodeResponse) => data.node,
      onSuccess: (node) => {
        setTargetedNodes((prevNodes) =>
          prevNodes.filter((h) => h.id !== node.id).concat(node)
        );
        const targets = selectedQueryTargets;
        node.target_type = "nodes";
        targets.push(node);
        setSelectedQueryTargets([...targets]);
        if (!queryParamNodesAdded) {
          setQueryParamNodesAdded(true);
        }
        router.replace(location.pathname);
      },
    }
  );

  useEffect(() => {
    setSelectedQueryTargetsByType({
      nodes: targetedNodes,
      labels: targetedLabels,
      teams: targetedTeams,
    });
  }, [targetedLabels, targetedNodes, targetedTeams]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Run Discover TLS certificates | Queries | Mdmlab
    if (storedQuery?.name) {
      document.title = `Run ${storedQuery.name} | Queries | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Queries | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedQuery?.name]);

  const goToQueryEditor = useCallback(
    () =>
      queryId
        ? router.push(PATHS.EDIT_QUERY(queryId, currentTeamId))
        : router.push(PATHS.NEW_QUERY(currentTeamId)),
    []
  );

  const renderScreen = () => {
    const step1Props = {
      baseClass,
      queryId,
      selectedTargets: selectedQueryTargets,
      targetedNodes,
      targetedLabels,
      targetedTeams,
      targetsTotalCount,
      goToQueryEditor,
      goToRunQuery: () => setStep(LIVE_QUERY_STEPS[2]),
      setSelectedTargets: setSelectedQueryTargets,
      setTargetedNodes,
      setTargetedLabels,
      setTargetedTeams,
      setTargetsTotalCount,
      isObserverCanRunQuery: storedQuery?.observer_can_run,
    };

    const step2Props = {
      queryId,
      selectedTargets: selectedQueryTargets,
      storedQuery,
      setSelectedTargets: setSelectedQueryTargets,
      goToQueryEditor,
      targetsTotalCount,
    };

    switch (step) {
      case LIVE_QUERY_STEPS[2]:
        return <RunQuery {...step2Props} />;
      default:
        return <SelectTargets {...step1Props} />;
    }
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}_wrapper`}>{renderScreen()}</div>
    </MainContent>
  );
};

export default RunQueryPage;
