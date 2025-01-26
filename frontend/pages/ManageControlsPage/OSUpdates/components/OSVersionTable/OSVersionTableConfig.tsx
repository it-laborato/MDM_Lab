import React from "react";

import { IOperatingSystemVersion } from "interfaces/operating_system";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllNodesLink from "components/ViewAllNodesLink";
import TextCell from "components/TableContainer/DataTable/TextCell";

import OSTypeCell from "../OSTypeCell";
import { IFilteredOperatingSystemVersion } from "../CurrentVersionSection/CurrentVersionSection";

interface IOSTypeCellProps {
  row: {
    original: IFilteredOperatingSystemVersion;
  };
}

interface INodeCellProps {
  row: {
    original: IOperatingSystemVersion;
  };
}

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

// eslint-disable-next-line import/prefer-default-export
export const generateTableHeaders = (teamId: number) => {
  return [
    {
      title: "OS type",
      Header: "OS type",
      disableSortBy: true,
      accessor: "platform",
      Cell: ({ row }: IOSTypeCellProps) => (
        <OSTypeCell
          platform={row.original.platform}
          versionName={row.original.name_only}
        />
      ),
    },
    {
      title: "Version",
      Header: "Version",
      disableSortBy: true,
      accessor: "version",
    },
    {
      title: "Nodes",
      accessor: "nodes_count",
      disableSortBy: false,
      Header: (cellProps: IHeaderProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      Cell: ({ row }: INodeCellProps): JSX.Element => {
        const { nodes_count } = row.original;
        return <TextCell value={nodes_count} />;
      },
    },
    {
      title: "",
      Header: "",
      accessor: "linkToFilteredNodes",
      disableSortBy: true,
      Cell: (cellProps: IOSTypeCellProps) => {
        return (
          <>
            {cellProps.row.original && (
              <ViewAllNodesLink
                queryParams={{
                  os_name: cellProps.row.original.name_only,
                  os_version: cellProps.row.original.version,
                  team_id: teamId,
                }}
                condensed
                className="os-nodes-link"
                rowHover
              />
            )}
          </>
        );
      },
    },
  ];
};
