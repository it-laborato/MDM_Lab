import React from "react";
import { ISoftware } from "interfaces/software";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ViewAllNodesLink from "components/ViewAllNodesLink";

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: ISoftware;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell: (props: ICellProps) => JSX.Element;
  disableHidden?: boolean;
  disableSortBy?: boolean;
}

const generateTableHeaders = (teamId?: number): IDataColumn[] => [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Version",
    Header: "Version",
    disableSortBy: true,
    accessor: "version",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Nodes",
    Header: "Nodes",
    disableSortBy: true,
    accessor: "nodes_count",
    Cell: (cellProps: ICellProps) => <TextCell value={cellProps.cell.value} />,
  },
  {
    title: "Actions",
    Header: "",
    disableSortBy: true,
    accessor: "id",
    Cell: (cellProps: ICellProps) => {
      return (
        <ViewAllNodesLink
          queryParams={{ software_id: cellProps.cell.value, team_id: teamId }} // TODO: Should redirect with the current team id?
          className="software-link"
          condensed
        />
      );
    },
  },
];

export default generateTableHeaders;
