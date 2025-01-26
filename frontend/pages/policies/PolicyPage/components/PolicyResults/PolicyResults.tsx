import React, { useState, useContext } from "react";
import { Tab, Tabs, TabList, TabPanel } from "react-tabs";
import classnames from "classnames";
import FileSaver from "file-saver";
import { get } from "lodash";
import { PolicyContext } from "context/policy";

import {
  generateCSVFilename,
  generateCSVPolicyResults,
  generateCSVPolicyErrors,
} from "utilities/generate_csv";
import { ICampaign } from "interfaces/campaign";
import { ITarget } from "interfaces/target";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import TabsWrapper from "components/TabsWrapper";
import InfoBanner from "components/InfoBanner";
import ShowQueryModal from "components/modals/ShowQueryModal";
import TooltipWrapper from "components/TooltipWrapper";

import ResultsHeading from "components/queries/queryResults/QueryResultsHeading";
import AwaitingResults from "components/queries/queryResults/AwaitingResults";

import PolicyResultsTable from "../PolicyResultsTable/PolicyResultsTable";
import PolicyQueriesErrorsTable from "../PolicyErrorsTable/PolicyErrorsTable";
import { getYesNoCounts } from "./helpers";

interface IPolicyResultsProps {
  campaign: ICampaign;
  isQueryFinished: boolean;
  policyName?: string;
  onRunQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onStopQuery: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  setSelectedTargets: (value: ITarget[]) => void;
  goToQueryEditor: () => void;
  targetsTotalCount: number;
}

const baseClass = "query-results";
const CSV_TITLE = "New Policy";
const NAV_TITLES = {
  RESULTS: "Results",
  ERRORS: "Errors",
};

const PolicyResults = ({
  campaign,
  isQueryFinished,
  policyName,
  onRunQuery,
  onStopQuery,
  setSelectedTargets,
  goToQueryEditor,
  targetsTotalCount,
}: IPolicyResultsProps): JSX.Element => {
  const { lastEditedQueryBody } = useContext(PolicyContext);

  const { nodes: nodeResponses, nodes_count: nodesCount, errors } =
    campaign || {};

  const totalRowsCount = get(campaign, ["nodes_count", "successful"], 0);

  const [navTabIndex, setNavTabIndex] = useState(0);
  const [showQueryModal, setShowQueryModal] = useState(false);

  const onExportResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (nodeResponses) {
      const nodesExport = nodeResponses.map((node) => {
        return {
          node: node.display_name,
          status:
            node.query_results && node.query_results.length ? "yes" : "no",
        };
      });

      FileSaver.saveAs(
        generateCSVPolicyResults(
          nodesExport,
          generateCSVFilename(`${policyName || CSV_TITLE} - Results`)
        )
      );
    }
  };

  const onExportErrorsResults = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();

    if (errors) {
      FileSaver.saveAs(
        generateCSVPolicyErrors(
          errors,
          generateCSVFilename(`${policyName || CSV_TITLE} - Errors`)
        )
      );
    }
  };

  const onShowQueryModal = () => {
    setShowQueryModal(!showQueryModal);
  };

  const onQueryDone = () => {
    setSelectedTargets([]);
    goToQueryEditor();
  };

  const renderTableButtons = (tableType: "results" | "errors") => {
    return (
      <div className={`${baseClass}__results-cta`}>
        <Button
          className={`${baseClass}__show-query-btn`}
          onClick={onShowQueryModal}
          variant="text-icon"
        >
          <>
            Show query <Icon name="eye" />
          </>
        </Button>
        <Button
          className={`${baseClass}__export-btn`}
          onClick={
            tableType === "errors" ? onExportErrorsResults : onExportResults
          }
          variant="text-icon"
        >
          <>
            Export {tableType}
            <Icon name="download" color="core-mdmlab-blue" />
          </>
        </Button>
      </div>
    );
  };

  const renderPassFailPcts = () => {
    const { yes: yesCt, no: noCt } = getYesNoCounts(nodeResponses);
    return (
      <span className={`${baseClass}__results-pass-fail-pct`}>
        {" "}
        (Yes:{" "}
        <TooltipWrapper tipContent={`${yesCt} node${yesCt !== 1 ? "s" : ""}`}>
          {Math.ceil((yesCt / nodesCount.successful) * 100)}%
        </TooltipWrapper>
        , No:{" "}
        <TooltipWrapper tipContent={`${noCt} node${noCt !== 1 ? "s" : ""}`}>
          {Math.floor((noCt / nodesCount.successful) * 100)}%
        </TooltipWrapper>
        )
      </span>
    );
  };

  const renderResultsTable = () => {
    const emptyResults =
      !nodeResponses || !nodeResponses.length || !nodesCount.successful;
    const hasNoResultsYet = !isQueryFinished && emptyResults;
    const finishedWithNoResults =
      isQueryFinished && (!nodesCount.successful || emptyResults);

    if (hasNoResultsYet) {
      return <AwaitingResults />;
    }

    if (finishedWithNoResults) {
      return (
        <p className="no-results-message">
          Your live query returned no results.
          <span>
            Expecting to see results? Check to see if the node
            {`${targetsTotalCount > 1 ? "s" : ""}`} you targeted reported
            &ldquo;Online&rdquo; or check out the &ldquo;Errors&rdquo; table.
          </span>
        </p>
      );
    }

    return (
      <div className={`${baseClass}__results-table-container`}>
        <InfoBanner>
          Nodes that responded with results are marked <strong>Yes</strong>.
          Nodes that responded with no results are marked <strong>No</strong>.
        </InfoBanner>
        <div className={`${baseClass}__results-table-header`}>
          <span className={`${baseClass}__results-meta`}>
            <span className={`${baseClass}__results-count`}>
              {totalRowsCount} result{totalRowsCount !== 1 && "s"}
            </span>
            {isQueryFinished && renderPassFailPcts()}
          </span>
          <div className={`${baseClass}__results-cta`}>
            {renderTableButtons("results")}
          </div>
        </div>
        <PolicyResultsTable
          isLoading={false}
          nodeResponses={nodeResponses}
          resultsTitle="nodes"
        />
      </div>
    );
  };

  const renderErrorsTable = () => {
    return (
      <div className={`${baseClass}__error-table-container`}>
        <div className={`${baseClass}__errors-table-header`}>
          {errors && (
            <span className={`${baseClass}__error-count`}>
              {errors.length} error{errors.length !== 1 && "s"}
            </span>
          )}
          <div className={`${baseClass}__errors-cta`}>
            {renderTableButtons("errors")}
          </div>
        </div>
        <PolicyQueriesErrorsTable
          isLoading={false}
          errorsList={errors}
          resultsTitle="errors"
        />
      </div>
    );
  };

  const firstTabClass = classnames("react-tabs__tab", "no-count", {
    "errors-empty": !errors || errors?.length === 0,
  });

  return (
    <div className={baseClass}>
      <ResultsHeading
        respondedNodes={nodesCount.total}
        targetsTotalCount={targetsTotalCount}
        isQueryFinished={isQueryFinished}
        onClickDone={onQueryDone}
        onClickRunAgain={onRunQuery}
        onClickStop={onStopQuery}
      />
      <TabsWrapper>
        <Tabs selectedIndex={navTabIndex} onSelect={(i) => setNavTabIndex(i)}>
          <TabList>
            <Tab className={firstTabClass}>{NAV_TITLES.RESULTS}</Tab>
            <Tab disabled={!errors?.length}>
              <span>
                {errors?.length > 0 && (
                  <span className="count">{errors.length}</span>
                )}
                {NAV_TITLES.ERRORS}
              </span>
            </Tab>
          </TabList>
          <TabPanel>{renderResultsTable()}</TabPanel>
          <TabPanel>{renderErrorsTable()}</TabPanel>
        </Tabs>
      </TabsWrapper>
      {showQueryModal && (
        <ShowQueryModal
          query={lastEditedQueryBody}
          onCancel={onShowQueryModal}
        />
      )}
    </div>
  );
};

export default PolicyResults;
