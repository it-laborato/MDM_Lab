/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { CellProps, Column } from "react-table";
import ReactTooltip from "react-tooltip";

import { IDeviceUser, INode } from "interfaces/node";
import Checkbox from "components/forms/fields/Checkbox";
import DiskSpaceIndicator from "pages/nodes/components/DiskSpaceIndicator";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import NodeMdmStatusCell from "components/TableContainer/DataTable/NodeMdmStatusCell/NodeMdmStatusCell";
import IssueCell from "components/TableContainer/DataTable/IssueCell/IssueCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import StatusIndicator from "components/StatusIndicator";
import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { HumanTimeDiffWithMdmlabLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import NotSupported from "components/NotSupported";

import {
  humanNodeMemory,
  humanNodeLastSeen,
  nodeTeamName,
  tooltipTextWithLineBreaks,
} from "utilities/helpers";
import { COLORS } from "styles/var/colors";
import {
  IHeaderProps,
  IStringCellProps,
  INumberCellProps,
} from "interfaces/datatable_config";
import PATHS from "router/paths";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import getNodeStatusTooltipText from "../helpers";

type INodeTableColumnConfig = Column<INode> & {
  // This is used to prevent these columns from being hidden. This will be
  // used in EditColumnsModal to prevent these columns from being hidden.
  disableHidden?: boolean;
  // We add title in the column config to be able to use it in the EditColumnsModal
  // as well
  title?: string;
};

type INodeTableHeaderProps = IHeaderProps<INode>;
type INodeTableStringCellProps = IStringCellProps<INode>;
type INodeTableNumberCellProps = INumberCellProps<INode>;
type ISelectionCellProps = CellProps<INode>;
type IIssuesCellProps = CellProps<INode, INode["issues"]>;
type IDeviceUserCellProps = CellProps<INode, INode["device_mapping"]>;

const condenseDeviceUsers = (users: IDeviceUser[]): string[] => {
  if (!users?.length) {
    return [];
  }
  const condensed =
    users.length === 4
      ? users
          .slice(-4)
          .map((u) => u.email)
          .reverse()
      : users
          .slice(-3)
          .map((u) => u.email)
          .reverse() || [];
  return users.length > 4
    ? condensed.concat(`+${users.length - 3} more`) // TODO: confirm limit
    : condensed;
};

const lastSeenTime = (status: string, seenTime: string): string => {
  if (status !== "online") {
    return `Last Seen: ${humanNodeLastSeen(seenTime)} UTC`;
  }
  return "Online";
};

const allNodeTableHeaders: INodeTableColumnConfig[] = [
  // We are using React Table useRowSelect functionality for the selection header.
  // More information on its API can be found here
  // https://react-table.tanstack.com/docs/api/useRowSelect
  {
    id: "selection",
    Header: (cellProps: INodeTableHeaderProps) => {
      const props = cellProps.getToggleAllRowsSelectedProps();
      const checkboxProps = {
        value: props.checked,
        indeterminate: props.indeterminate,
        onChange: () => cellProps.toggleAllRowsSelected(),
      };
      return <Checkbox {...checkboxProps} enableEnterToCheck />;
    },
    Cell: (cellProps: ISelectionCellProps) => {
      const props = cellProps.row.getToggleRowSelectedProps();
      const checkboxProps = {
        value: props.checked,
        onChange: () => cellProps.row.toggleRowSelected(),
      };
      return <Checkbox {...checkboxProps} enableEnterToCheck />;
    },
    disableHidden: true,
  },
  {
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell value="Node" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "display_name",
    id: "display_name",
    Cell: (cellProps: INodeTableStringCellProps) => {
      if (
        // if the node is pending, we want to disable the link to node details
        cellProps.row.original.mdm.enrollment_status === "Pending" &&
        // pending status is only supported for Apple devices
        (cellProps.row.original.platform === "darwin" ||
          cellProps.row.original.platform === "ios" ||
          cellProps.row.original.platform === "ipados") &&
        // osquery version is populated along with the rest of node details so use it
        // here to check if we already have node details and don't need to disable the link
        !cellProps.row.original.osquery_version
      ) {
        return (
          <>
            <span
              className="text-cell"
              data-tip
              data-for={`node__${cellProps.row.original.id}`}
            >
              {cellProps.cell.value}
            </span>
            <ReactTooltip
              effect="solid"
              backgroundColor={COLORS["tooltip-bg"]}
              id={`node__${cellProps.row.original.id}`}
              data-html
            >
              <span className={`tooltip__tooltip-text`}>
                This node was ordered using <br />
                Apple Business Manager <br />
                (ABM). You will see node <br />
                vitals when it is enrolled in Mdmlab <br />
              </span>
            </ReactTooltip>
          </>
        );
      }
      return (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.HOST_DETAILS(cellProps.row.original.id)}
          title={lastSeenTime(
            cellProps.row.original.status,
            cellProps.row.original.seen_time
          )}
        />
      );
    },
    disableHidden: true,
  },
  {
    title: "Nodename",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Nodename"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "nodename",
    id: "nodename",
    Cell: (cellProps: INodeTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Computer name",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Computer name"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "computer_name",
    id: "computer_name",
    Cell: (cellProps: INodeTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Team",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell value="Team" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "team_name",
    id: "team_name",
    Cell: (cellProps) => (
      <TextCell value={cellProps.cell.value} formatter={nodeTeamName} />
    ),
  },
  {
    title: "Status",
    Header: (cellProps: INodeTableHeaderProps) => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              Online nodes will respond to a live query. Offline nodes
              won&apos;t respond to a live query because they may be shut down,
              asleep, or not connected to the internet.
            </>
          }
          className="status-header"
        >
          Status
        </TooltipWrapper>
      );
      return (
        <HeaderCell
          value={cellProps.rows.length === 1 ? "Status" : titleWithToolTip}
          disableSortBy
        />
      );
    },
    disableSortBy: true,
    accessor: "status",
    id: "status",
    Cell: (cellProps: INodeTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      const value = cellProps.cell.value;
      const tooltip = {
        tooltipText: getNodeStatusTooltipText(value),
      };
      return <StatusIndicator value={value} tooltip={tooltip} />;
    },
  },
  {
    title: "Issues",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell value="Issues" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "issues",
    id: "issues",
    sortDescFirst: true,
    Cell: (cellProps: IIssuesCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <IssueCell
          issues={cellProps.row.original.issues}
          rowId={cellProps.row.original.id}
        />
      );
    },
  },
  {
    title: "Disk space available",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Disk space available"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "gigs_disk_space_available",
    id: "gigs_disk_space_available",
    Cell: (cellProps: INodeTableNumberCellProps) => {
      const {
        id,
        platform,
        percent_disk_space_available,
      } = cellProps.row.original;
      if (platform === "chrome") {
        return NotSupported;
      }
      return (
        <DiskSpaceIndicator
          baseClass="gigs_disk_space_available__cell"
          gigsDiskSpaceAvailable={cellProps.cell.value}
          percentDiskSpaceAvailable={percent_disk_space_available}
          id={`disk-space__${id}`}
          platform={platform}
        />
      );
    },
  },
  {
    title: "Operating system",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Operating system"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "os_version",
    id: "os_version",
    Cell: (cellProps: INodeTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Used by",
    Header: "Used by",
    disableSortBy: true,
    accessor: "device_mapping",
    id: "device_mapping",
    Cell: (cellProps: IDeviceUserCellProps) => {
      const numUsers = cellProps.cell.value?.length || 0;
      const users = condenseDeviceUsers(cellProps.cell.value || []);
      if (users.length > 1) {
        return (
          <TooltipWrapper
            tipContent={tooltipTextWithLineBreaks(users)}
            underline={false}
            showArrow
            position="top"
            tipOffset={10}
          >
            <TextCell italic value={`${numUsers} users`} />
          </TooltipWrapper>
        );
      }
      if (users.length === 1) {
        return <TextCell value={users[0]} />;
      }
      return <TextCell />;
    },
  },
  {
    title: "Private IP address",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Private IP address"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_ip",
    id: "primary_ip",
    Cell: (cellProps: INodeTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return <TextCell value={cellProps.cell.value} />;
    },
  },
  {
    title: "MDM status",
    Header: () => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              Settings can be updated remotely on nodes with MDM turned
              <br />
              on. To filter by MDM status, head to the Dashboard page.
            </>
          }
        >
          MDM status
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: (originalRow) => originalRow.mdm.enrollment_status,
    id: "mdm.enrollment_status",
    Cell: NodeMdmStatusCell,
  },
  {
    title: "MDM server URL",
    Header: () => {
      const titleWithToolTip = (
        <TooltipWrapper
          tipContent={
            <>
              The MDM server that updates settings on the node. To
              <br />
              filter by MDM server URL, head to the Dashboard page.
            </>
          }
        >
          MDM server URL
        </TooltipWrapper>
      );
      return <HeaderCell value={titleWithToolTip} disableSortBy />;
    },
    disableSortBy: true,
    accessor: (originalRow) => originalRow.mdm.server_url,
    id: "mdm.server_url",
    Cell: (cellProps: INodeTableStringCellProps) => {
      if (cellProps.row.original.platform === "chrome") {
        return NotSupported;
      }
      if (cellProps.cell.value) {
        return <TooltipTruncatedTextCell value={cellProps.cell.value} />;
      }
      return <span className="text-muted">{DEFAULT_EMPTY_CELL_VALUE}</span>;
    },
  },
  {
    title: "Public IP address",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value={
          <TooltipWrapper tipContent="The IP address the node uses to connect to Mdmlab.">
            Public IP address
          </TooltipWrapper>
        }
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "public_ip",
    id: "public_ip",
    Cell: (cellProps: INodeTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <TextCell value={cellProps.cell.value ?? DEFAULT_EMPTY_CELL_VALUE} />
      );
    },
  },
  {
    title: "UUID",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell value="UUID" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "uuid",
    id: "uuid",
    Cell: ({ cell: { value } }: INodeTableStringCellProps) =>
      value ? <TooltipTruncatedTextCell value={value} /> : <TextCell />,
  },
  {
    title: "CPU",
    Header: "CPU",
    disableSortBy: true,
    accessor: "cpu_type",
    id: "cpu_type",
    Cell: (cellProps: INodeTableStringCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return <TextCell value={cellProps.cell.value} />;
    },
  },
  {
    title: "RAM",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell value="RAM" isSortedDesc={cellProps.column.isSortedDesc} />
    ),
    accessor: "memory",
    id: "memory",
    Cell: (cellProps: INodeTableNumberCellProps) => {
      if (
        cellProps.row.original.platform === "ios" ||
        cellProps.row.original.platform === "ipados"
      ) {
        return NotSupported;
      }
      return (
        <TextCell value={cellProps.cell.value} formatter={humanNodeMemory} />
      );
    },
  },
  {
    title: "MAC address",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="MAC address"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "primary_mac",
    id: "primary_mac",
    Cell: (cellProps: INodeTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Serial number",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Serial number"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_serial",
    id: "hardware_serial",
    Cell: (cellProps: INodeTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Hardware model",
    Header: (cellProps: INodeTableHeaderProps) => (
      <HeaderCell
        value="Hardware model"
        isSortedDesc={cellProps.column.isSortedDesc}
      />
    ),
    accessor: "hardware_model",
    id: "hardware_model",
    Cell: (cellProps: INodeTableStringCellProps) => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
];

const defaultHiddenColumns = [
  "nodename",
  "computer_name",
  "device_mapping",
  "primary_mac",
  "public_ip",
  "cpu_type",
  // TODO: should those be mdm.<blah>?
  "mdm.server_url",
  "mdm.enrollment_status",
  "memory",
  "uptime",
  "uuid",
  "seen_time",
  "hardware_model",
  "hardware_serial",
];

/**
 * Will generate a node table column configuration based off of the current user
 * permissions and license tier of mdmlab they are on.
 */
const generateAvailableTableHeaders = ({
  isFreeTier = true,
  isOnlyObserver = true,
}: {
  isFreeTier: boolean | undefined;
  isOnlyObserver: boolean | undefined;
}): INodeTableColumnConfig[] => {
  return allNodeTableHeaders.reduce(
    (columns: Column<INode>[], currentColumn: Column<INode>) => {
      // skip over column headers that are not shown in free observer tier
      if (isFreeTier) {
        if (
          isOnlyObserver &&
          ["selection", "team_name"].includes(currentColumn.id || "")
        ) {
          return columns;
          // skip over column headers that are not shown in free admin/maintainer
        }
        if (
          currentColumn.id === "team_name" ||
          currentColumn.id === "mdm.server_url" ||
          currentColumn.id === "mdm.enrollment_status"
        ) {
          return columns;
        }
      } else if (isOnlyObserver && currentColumn.id === "selection") {
        // In premium tier, we want to check user role to enable/disable select column
        return columns;
      }

      columns.push(currentColumn);
      return columns;
    },
    []
  );
};

/**
 * Will generate a node table column configuration that a user currently sees.
 */
const generateVisibleTableColumns = ({
  hiddenColumns,
  isFreeTier = true,
  isOnlyObserver = true,
}: {
  hiddenColumns: string[];
  isFreeTier: boolean | undefined;
  isOnlyObserver: boolean | undefined;
}): INodeTableColumnConfig[] => {
  // remove columns set as hidden by the user.
  return generateAvailableTableHeaders({ isFreeTier, isOnlyObserver }).filter(
    (column) => {
      return !hiddenColumns.includes(column.id as string);
    }
  );
};

export {
  defaultHiddenColumns,
  generateAvailableTableHeaders,
  generateVisibleTableColumns,
};
