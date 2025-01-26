/* eslint-disable react/prop-types */

import React from "react";
import { Column, Row } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { INode } from "interfaces/node";

import TextCell from "components/TableContainer/DataTable/TextCell";
import LiveQueryIssueCell from "components/TableContainer/DataTable/LiveQueryIssueCell/LiveQueryIssueCell";
import StatusIndicator from "components/StatusIndicator";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

export type ITargestInputNodeTableConfig = Column<INode>;
type ITableStringCellProps = IStringCellProps<INode>;

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
export const generateTableHeaders = (
  handleRowRemove?: (value: Row<INode>) => void
): ITargestInputNodeTableConfig[] => {
  const deleteHeader = handleRowRemove
    ? [
        {
          id: "delete",
          Header: "",
          Cell: (cellProps: ITableStringCellProps) => (
            <Button
              onClick={() => handleRowRemove(cellProps.row)}
              variant="icon"
            >
              <Icon name="close-filled" />
            </Button>
          ),
          disableHidden: true,
        },
      ]
    : [];

  return [
    {
      Header: "Node",
      accessor: "display_name",
      Cell: (cellProps: ITableStringCellProps) => {
        return (
          <LiveQueryIssueCell
            displayName={cellProps.cell.value}
            distributedInterval={cellProps.row.original.distributed_interval}
            status={cellProps.row.original.status}
            rowId={cellProps.row.original.id}
          />
        );
      },
    },
    // TODO: Consider removing status column from selected nodes table because
    // status info is not refreshed once a target has been selected
    {
      Header: "Status",
      disableSortBy: true,
      accessor: "status",
      Cell: (cellProps) => <StatusIndicator value={cellProps.cell.value} />,
    },
    {
      Header: "Private IP address",
      accessor: "primary_ip",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "MAC address",
      accessor: "primary_mac",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "OS",
      accessor: "os_version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "Osquery",
      accessor: "osquery_version",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    ...deleteHeader,
  ];
};

export default null;
