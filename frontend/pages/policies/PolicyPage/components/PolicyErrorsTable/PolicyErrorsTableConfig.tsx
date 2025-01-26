/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { memoize } from "lodash";

import { ICampaignError } from "interfaces/campaign";
import sortUtils from "utilities/sort";

import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";

interface IHeaderProps {
  column: {
    node: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ICampaignError;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  const tableHeaders: IDataColumn[] = [
    {
      title: "Node",
      Header: "Node",
      disableSortBy: true,
      accessor: "node_display_name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Osquery version",
      Header: "Osquery version",
      disableSortBy: true,
      accessor: "osquery_version",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Error",
      Header: "Error",
      disableSortBy: true,
      accessor: "error",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
  ];
  return tableHeaders;
};

const generateDataSet = memoize(
  (policyNodesErrorsList: ICampaignError[] = []): ICampaignError[] => {
    policyNodesErrorsList = policyNodesErrorsList.sort((a, b) =>
      sortUtils.caseInsensitiveAsc(a.node_display_name, b.node_display_name)
    );
    return policyNodesErrorsList;
  }
);

export { generateTableHeaders, generateDataSet };
