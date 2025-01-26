import React from "react";
import { noop } from "lodash";

import { IPolicyNodeResponse } from "interfaces/node";
import TableContainer from "components/TableContainer";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PolicyResultsTableConfig";

// TODO - this class is duplicated and styles are overlapping with PolicyErrorsTable. Differentiate
// them clearly and encapsulate common styles.
const baseClass = "policy-results-table";

interface IPolicyResultsTableProps {
  nodeResponses: IPolicyNodeResponse[];
  isLoading: boolean;
  resultsTitle?: string;
  canAddOrDeletePolicy?: boolean;
}

const PolicyResultsTable = ({
  nodeResponses,
  isLoading,
  resultsTitle,
  canAddOrDeletePolicy,
}: IPolicyResultsTableProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle={resultsTitle || "policies"}
        columnConfigs={generateTableHeaders()}
        data={generateDataSet(nodeResponses)}
        isLoading={isLoading}
        defaultSortHeader="query_results"
        defaultSortDirection="asc"
        showMarkAllPages={false}
        isAllPagesSelected={false}
        isClientSidePagination
        primarySelectAction={{
          name: "delete policy",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "text-icon",
        }}
        emptyComponent={() => (
          <div className="no-nodes__inner">
            <p>No nodes are online.</p>
          </div>
        )}
        onQueryChange={noop}
        disableCount
      />
    </div>
  );
};

export default PolicyResultsTable;
