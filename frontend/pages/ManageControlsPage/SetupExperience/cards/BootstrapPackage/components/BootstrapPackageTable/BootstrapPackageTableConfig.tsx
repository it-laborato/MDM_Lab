import React from "react";

import {
  BootstrapPackageStatus,
  IBootstrapPackageAggregate,
} from "interfaces/mdm";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import ViewAllNodesLink from "components/ViewAllNodesLink";

interface IStatusCellValue {
  displayName: string;
  statusName: "success" | "pending" | "error";
  value: BootstrapPackageStatus;
  tooltip?: string | JSX.Element;
}

interface IStatusCellProps {
  cell: {
    value: IStatusCellValue;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: {
      status: IStatusCellValue;
      teamId: number;
    };
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

type IColumnConfig = {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IStatusCellProps) => JSX.Element);
};

export const COLUMN_CONFIGS: IColumnConfig[] = [
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "status",
    Cell: ({ cell: { value } }: IStatusCellProps) => {
      const tooltipProp = value.tooltip
        ? { tooltipText: value.tooltip }
        : undefined;
      return (
        <StatusIndicatorWithIcon
          status={value.statusName}
          value={value.displayName}
          tooltip={tooltipProp}
        />
      );
    },
  },
  {
    title: "Nodes",
    Header: (cellProps: IHeaderProps) => (
      <HeaderCell
        value={cellProps.column.title}
        isSortedDesc={cellProps.column.isSortedDesc}
        disableSortBy={false}
      />
    ),
    accessor: "nodes",
    Cell: ({ cell: { value: aggregateCount } }: ICellProps) => {
      return (
        <TextCell value={aggregateCount} formatter={(val) => <>{val}</>} />
      );
    },
  },
  {
    title: "",
    Header: "",
    accessor: "linkToFilteredNodes",
    disableSortBy: true,
    Cell: (cellProps: ICellProps) => {
      return (
        <>
          {cellProps.row.original && (
            <ViewAllNodesLink
              className="view-nodes-link"
              queryParams={{
                bootstrap_package: cellProps.row.original.status.value,
                team_id: cellProps.row.original.teamId,
              }}
              rowHover
            />
          )}
        </>
      );
    },
  },
];

type StatusNames = keyof IBootstrapPackageAggregate;

type StatusEntry = [StatusNames, number];

const STATUS_CELL_VALUES: Record<StatusNames, IStatusCellValue> = {
  installed: {
    displayName: "Installed",
    statusName: "success",
    value: BootstrapPackageStatus.INSTALLED,
    tooltip: "The node installed the package.",
  },
  pending: {
    displayName: "Pending",
    statusName: "pending",
    value: BootstrapPackageStatus.PENDING,
    tooltip: "The node will install the package when it enrolls.",
  },
  failed: {
    displayName: "Failed",
    statusName: "error",
    value: BootstrapPackageStatus.FAILED,
    tooltip: "The node failed to install the package when it enrolled.",
  },
};

export const generateTableData = (
  data?: IBootstrapPackageAggregate,
  currentTeamId?: number
) => {
  if (!data) return [];
  const entries = Object.entries(data) as StatusEntry[];

  return entries.map(([status, numNodes]) => ({
    status: STATUS_CELL_VALUES[status],
    nodes: numNodes,
    teamId: currentTeamId,
  }));
};
