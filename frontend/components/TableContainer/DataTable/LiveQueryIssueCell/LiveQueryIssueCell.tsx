import React from "react";
import ReactTooltip from "react-tooltip";

import Icon from "components/Icon";
import { COLORS } from "styles/var/colors";

interface ILiveQueryIssueCellProps<T> {
  displayName: string;
  distributedInterval: number;
  status: string;
  rowId: number;
}

const LiveQueryIssueCell = ({
  displayName,
  distributedInterval,
  status,
  rowId,
}: ILiveQueryIssueCellProps<any>): JSX.Element => {
  if (distributedInterval < 60 && status === "online") {
    return <>{displayName}</>;
  }

  return (
    <>
      {displayName}{" "}
      <span
        className={`node-issue tooltip tooltip__tooltip-icon`}
        data-tip
        data-for={`node-issue__${rowId.toString()}`}
        data-tip-disable={false}
      >
        <Icon
          name="error-outline"
          size="small"
          color={status === "offline" ? "status-error" : "status-warning"}
        />
      </span>
      <ReactTooltip
        place="top"
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
        id={`node-issue__${rowId.toString()}`}
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          {status === "offline" ? (
            <>
              Offline nodes will not <br />
              respond to a live query.
            </>
          ) : (
            <>
              This node might take up to
              <br /> {distributedInterval} seconds to respond.
            </>
          )}
        </span>
      </ReactTooltip>
    </>
  );
};

export default LiveQueryIssueCell;
