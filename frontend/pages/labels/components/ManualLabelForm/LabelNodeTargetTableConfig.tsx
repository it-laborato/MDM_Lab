/* eslint-disable react/prop-types */

import React from "react";
import { Column, Row } from "react-table";

import { IStringCellProps } from "interfaces/datatable_config";
import { INode } from "interfaces/node";

import TextCell from "components/TableContainer/DataTable/TextCell";
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
            <div onClick={() => handleRowRemove(cellProps.row)}>
              <Icon name="close-filled" />
            </div>
          ),
          disableHidden: true,
        },
      ]
    : [];

  return [
    {
      Header: "Node",
      accessor: "display_name",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "Nodename",
      accessor: "nodename",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    {
      Header: "Serial number",
      accessor: "hardware_serial",
      Cell: (cellProps) => <TextCell value={cellProps.cell.value} />,
    },
    ...deleteHeader,
  ];
};

export default null;
