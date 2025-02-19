/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { millisecondsToHours, millisecondsToMinutes } from "date-fns";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import Icon from "components/Icon";
import { IPolicyStats } from "interfaces/policy";
import PATHS from "router/paths";
import sortUtils from "utilities/sort";
import { PolicyResponse } from "utilities/constants";
import { createNodesByPolicyPath } from "utilities/helpers";
import InheritedBadge from "components/InheritedBadge";
import { getConditionalSelectHeaderCheckboxProps } from "components/TableContainer/utilities/config_utils";
import PassingColumnHeader from "../PassingColumnHeader";

interface IGetToggleAllRowsSelectedProps {
  checked: boolean;
  indeterminate: boolean;
  title: string;
  onChange: () => void;
  style: { cursor: string };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => IGetToggleAllRowsSelectedProps;
  toggleAllRowsSelected: () => void;
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IPolicyStats;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const getPolicyRefreshTime = (ms: number): string => {
  const seconds = ms / 1000;
  if (seconds < 60) {
    return `${seconds} seconds`;
  }
  if (seconds < 3600) {
    const minutes = millisecondsToMinutes(ms);
    return `${minutes} minute${minutes > 1 ? "s" : ""}`;
  }
  const hours = millisecondsToHours(ms);
  return `${hours} hour${hours > 1 ? "s" : ""}`;
};

const getTooltip = (osqueryPolicyMs: number): JSX.Element => {
  return (
    <>
      Mdmlab is collecting policy results. Try again
      <br />
      in about {getPolicyRefreshTime(osqueryPolicyMs)} as the system catches up.
    </>
  );
};

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  options: {
    selectedTeamId?: number | null;
    hasPermissionAndPoliciesToDelete?: boolean;
    tableType?: string;
  },
  isPremiumTier?: boolean
): IDataColumn[] => {
  const { selectedTeamId, hasPermissionAndPoliciesToDelete } = options;
  const viewingTeamPolicies = selectedTeamId !== -1;

  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <LinkCell
          className="w250 policy-name-cell"
          value={
            <>
              <div className="policy-name-text">{cellProps.cell.value}</div>
              {isPremiumTier && cellProps.row.original.critical && (
                <div className="critical-badge">
                  <span
                    className="critical-badge-icon"
                    data-tooltip-id={`critical-tooltip-${cellProps.row.original.id}`}
                  >
                    <Icon
                      className="critical-policy-icon"
                      name="policy"
                      size="small"
                      color="core-mdmlab-blue"
                    />
                  </span>
                  <ReactTooltip5
                    className="critical-tooltip"
                    disableStyleInjection
                    place="top"
                    opacity={1}
                    id={`critical-tooltip-${cellProps.row.original.id}`}
                    offset={8}
                    positionStrategy="fixed"
                  >
                    This policy has been marked as critical.
                  </ReactTooltip5>
                </div>
              )}
              {viewingTeamPolicies &&
                cellProps.row.original.team_id === null && (
                  <InheritedBadge tooltipContent="This policy runs on all nodes." />
                )}
            </>
          }
          path={PATHS.EDIT_POLICY(cellProps.row.original)}
        />
      ),
      sortType: "caseInsensitive",
    },
    {
      title: "Yes",
      Header: (cellProps) => (
        <HeaderCell
          value={<PassingColumnHeader isPassing />}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "passing_node_count",
      Cell: (cellProps: ICellProps): JSX.Element => {
        if (cellProps.row.original.has_run) {
          return (
            <LinkCell
              value={`${cellProps.cell.value} node${
                cellProps.cell.value.toString() === "1" ? "" : "s"
              }`}
              path={createNodesByPolicyPath(
                cellProps.row.original.id,
                PolicyResponse.PASSING,
                selectedTeamId
              )}
            />
          );
        }
        return (
          <div className="policy-has-not-run">
            <span
              data-tooltip-id={`passing_${cellProps.row.original.id.toString()}`}
            >
              ---
            </span>
            <ReactTooltip5
              className="policy-has-not-run-tooltip"
              disableStyleInjection
              place="top"
              opacity={1}
              id={`passing_${cellProps.row.original.id.toString()}`}
              offset={8}
              positionStrategy="fixed"
            >
              {getTooltip(cellProps.row.original.next_update_ms)}
            </ReactTooltip5>
          </div>
        );
      },
    },
    {
      title: "No",
      Header: (cellProps) => (
        <HeaderCell
          value={<PassingColumnHeader isPassing={false} />}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "failing_node_count",
      Cell: (cellProps: ICellProps): JSX.Element => {
        if (cellProps.row.original.has_run) {
          return (
            <LinkCell
              value={`${cellProps.cell.value} node${
                cellProps.cell.value.toString() === "1" ? "" : "s"
              }`}
              path={createNodesByPolicyPath(
                cellProps.row.original.id,
                PolicyResponse.FAILING,
                selectedTeamId
              )}
            />
          );
        }
        return (
          <div className="policy-has-not-run">
            <span
              data-tooltip-id={`passing_${cellProps.row.original.id.toString()}`}
            >
              ---
            </span>
            <ReactTooltip5
              className="policy-has-not-run-tooltip"
              disableStyleInjection
              place="top"
              opacity={1}
              id={`passing_${cellProps.row.original.id.toString()}`}
              offset={8}
              positionStrategy="fixed"
            >
              {getTooltip(cellProps.row.original.next_update_ms)}
            </ReactTooltip5>
          </div>
        );
      },
      sortType: "caseInsensitive",
    },
  ];

  if (hasPermissionAndPoliciesToDelete) {
    tableHeaders.unshift({
      id: "selection",
      Header: (headerProps: any) => {
        // When viewing team policies, the select all checkbox will ignore inherited policies
        const teamCheckboxProps = getConditionalSelectHeaderCheckboxProps({
          headerProps,
          checkIfRowIsSelectable: (row) => row.original.team_id !== null,
        });

        // Regular table selection logic
        const {
          getToggleAllRowsSelectedProps,
          toggleAllRowsSelected,
        } = headerProps;
        const { checked, indeterminate } = getToggleAllRowsSelectedProps();

        const regularCheckboxProps = {
          value: checked,
          indeterminate,
          onChange: () => {
            toggleAllRowsSelected();
          },
        };

        const checkboxProps = viewingTeamPolicies
          ? teamCheckboxProps
          : regularCheckboxProps;
        return <Checkbox {...checkboxProps} enableEnterToCheck />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const inheritedPolicy = cellProps.row.original.team_id === null;
        const props = cellProps.row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: props.checked,
          onChange: () => cellProps.row.toggleRowSelected(),
        };

        // When viewing team policies and a row is an inherited policy, do not render checkbox
        if (viewingTeamPolicies && inheritedPolicy) {
          return <></>;
        }

        return <Checkbox {...checkboxProps} enableEnterToCheck />;
      },
      disableHidden: true,
    });
  }

  return tableHeaders;
};

// The next update will match the next node count update, unless extra time is needed for nodes to send in their policy results.
const nextPolicyUpdateMs = (
  policyItemUpdatedAtMs: Date,
  nextNodeCountUpdateMs: number,
  nodeCountUpdateIntervalMs: number,
  osqueryPolicyMs: number
) => {
  let timeFromPolicyItemUpdateToNextNodeCountUpdateMs =
    Date.now() - policyItemUpdatedAtMs.getTime() + nextNodeCountUpdateMs;
  let additionalUpdateTimeMs = 0;
  while (timeFromPolicyItemUpdateToNextNodeCountUpdateMs <= osqueryPolicyMs) {
    additionalUpdateTimeMs += nodeCountUpdateIntervalMs;
    timeFromPolicyItemUpdateToNextNodeCountUpdateMs += nodeCountUpdateIntervalMs;
  }
  return nextNodeCountUpdateMs + additionalUpdateTimeMs;
};

const generateDataSet = (
  policiesList: IPolicyStats[] = [],
  currentAutomatedPolicies?: number[],
  osquery_policy?: number
): IPolicyStats[] => {
  policiesList = policiesList.sort((a, b) =>
    sortUtils.caseInsensitiveAsc(a.name, b.name)
  );
  // To figure out if the policy has run for all the targeted nodes, we need to do the following calculation:
  // Each node asynchronously updates its own policy result every `osquery_policy` nanoseconds.
  // Then, the node count is updated by a cron job on the server every 1 hour (this is hardcoded on the server in `cron.go`).
  // So, we need to add `osquery_policy` to the time of the cron update.
  let policiesLastRun: Date;
  let osqueryPolicyMs = 0;
  const policiesThatHaveRunNodeCountUpdatedAt =
    // node counts of all policies that have run are updated at the same time, and are therefore
    // identical, so we can use the first one. Those that haven't run will be `null`.
    policiesList.find((p) => !!p.node_count_updated_at)
      ?.node_count_updated_at || "";
  // If node_count_updated_at is not present, we assume the worst case.
  const nodeCountUpdateIntervalMs = 60 * 60 * 1000; // 1 hour (from server's `cron.go`)
  const nodeCountUpdatedAtDate = policiesThatHaveRunNodeCountUpdatedAt
    ? new Date(policiesThatHaveRunNodeCountUpdatedAt)
    : new Date(Date.now() - nodeCountUpdateIntervalMs);
  if (osquery_policy) {
    // Convert from nanosecond to milliseconds
    osqueryPolicyMs = osquery_policy / 1000000;
    policiesLastRun = new Date(
      nodeCountUpdatedAtDate.getTime() - osqueryPolicyMs
    );
  } else {
    // temporarily unused - will restore use with upcoming DB update
    policiesLastRun = nodeCountUpdatedAtDate;
  }
  // Now we figure out when the next node count update will be.
  // The % (mod) is used below in case server was restarted and previously scheduled node count update was skipped.
  const nextNodeCountUpdateMs =
    nodeCountUpdateIntervalMs -
    (policiesThatHaveRunNodeCountUpdatedAt
      ? (Date.now() - nodeCountUpdatedAtDate.getTime()) %
        nodeCountUpdateIntervalMs
      : 0);

  policiesList.forEach((policyItem) => {
    policyItem.webhook =
      currentAutomatedPolicies &&
      currentAutomatedPolicies.includes(policyItem.id)
        ? "On"
        : "Off";

    // Define policy has_run based on updated_at compared against last time policies ran.
    const policyItemUpdatedAt = new Date(policyItem.updated_at);
    // TODO: restore and update setting of policyItem.has_run based on upcoming custom
    // `policy_membership_updated_at`(ish) DB column/API response field
    // policyItem.has_run = isAfter(policiesLastRun, policyItemUpdatedAt);

    // all of the policiess `has_run` will be either true (cron has run, so node_count_updated_at
    // has a value that is the same for all such policies) or false (policy is new, wasn't included
    // in last cron run, node_count_updated_at is `null`)
    policyItem.has_run = !!policyItem.node_count_updated_at;
    if (!policyItem.has_run) {
      // Include time for next update for reference in tooltip, which is only present if policy has not run.
      policyItem.next_update_ms = nextPolicyUpdateMs(
        policyItemUpdatedAt,
        nextNodeCountUpdateMs,
        nodeCountUpdateIntervalMs,
        osqueryPolicyMs
      );
    }
  });

  return policiesList;
};

export { generateTableHeaders, generateDataSet, nextPolicyUpdateMs };
