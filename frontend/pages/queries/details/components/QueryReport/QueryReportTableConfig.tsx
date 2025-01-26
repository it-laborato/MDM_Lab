/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

import { CellProps, Column } from "react-table";

import DefaultColumnFilter from "components/TableContainer/DataTable/DefaultColumnFilter";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";

import {
  getUniqueColsAreNumTypeFromRows,
  humanNodeLastSeen,
  internallyTruncateText,
} from "utilities/helpers";
import { IHeaderProps, IWebSocketData } from "interfaces/datatable_config";

type IQueryReportTableColumnConfig = Column<IWebSocketData>;
type ITableHeaderProps = IHeaderProps<IWebSocketData>;
type ITableCellProps = CellProps<IWebSocketData, string | unknown>;

const _unshiftNodename = (headers: IQueryReportTableColumnConfig[]) => {
  const newHeaders = [...headers];
  const displayNameIndex = headers.findIndex(
    (h) => h.id === "node_display_name"
  );
  if (displayNameIndex >= 0) {
    // remove nodename header from headers
    const [displayNameHeader] = newHeaders.splice(displayNameIndex, 1);
    // reformat title and insert at start of headers array
    newHeaders.unshift({ ...displayNameHeader, id: "Node" });
  }
  // TODO: Remove after v5 when node_nodename is removed rom API response.
  const nodeNameIndex = headers.findIndex((h) => h.id === "node_nodename");
  if (nodeNameIndex >= 0) {
    newHeaders.splice(nodeNameIndex, 1);
  }
  // end remove
  return newHeaders;
};

const generateReportColumnConfigsFromResults = (
  results: IWebSocketData[]
): IQueryReportTableColumnConfig[] => {
  const colsAreNumTypes = getUniqueColsAreNumTypeFromRows(results) as Map<
    string,
    boolean
  >;
  const columnConfigs = Array.from(colsAreNumTypes.keys()).map<
    Column<IWebSocketData>
  >((colName) => {
    return {
      id: colName,
      Header: (headerProps: ITableHeaderProps) => (
        <HeaderCell
          value={
            headerProps.column.id === "last_fetched"
              ? "Last fetched"
              : headerProps.column.id
          }
          isSortedDesc={headerProps.column.isSortedDesc}
        />
      ),
      accessor: (data) => data[colName],
      Cell: (cellProps: ITableCellProps) => {
        if (typeof cellProps.cell.value !== "string") return null;

        // Sorts chronologically by date, but UI displays readable last fetched
        if (cellProps.column.id === "last_fetched") {
          return <>{humanNodeLastSeen(cellProps?.cell?.value)}</>;
        }
        // truncate columns longer than 300 characters
        const val = cellProps?.cell?.value;
        return !!val?.length && val.length > 300 ? (
          internallyTruncateText(val)
        ) : (
          <>{val}</>
        );
      },
      Filter: DefaultColumnFilter, // Component hides filter for last_fetched
      filterType: "text",
      disableSortBy: false,
      sortType: colsAreNumTypes.get(colName)
        ? "alphanumeric"
        : "caseInsensitive",
    };
  });
  return _unshiftNodename(columnConfigs);
};

export default generateReportColumnConfigsFromResults;
