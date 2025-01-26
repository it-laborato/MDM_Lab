import React from "react";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import Icon from "components/Icon";
import NotSupported from "components/NotSupported";
import { INode } from "interfaces/node";

const baseClass = "node-mdm-status-cell";

const NodeMdmStatusCell = ({
  row: {
    original: { id, mdm, platform },
  },
  cell: { value },
}: {
  row: { original: INode };
  cell: { value: string };
}): JSX.Element => {
  if (platform === "chrome") {
    return NotSupported;
  }

  if (!value) {
    return <span className={`${baseClass}`}>{DEFAULT_EMPTY_CELL_VALUE}</span>;
  }

  return (
    <span className={`${baseClass}`}>
      {value}
      {mdm?.dep_profile_error && (
        <>
          <span
            className={`tooltip tooltip__tooltip-icon`}
            data-tip
            data-for={`node-mdm-status__${id}`}
            data-tip-disable={false}
          >
            <Icon name="error-outline" color="status-error" size="medium" />
          </span>
          <ReactTooltip
            place="top"
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id={`node-mdm-status__${id}`}
            data-html
          >
            <span className={`tooltip__tooltip-text`}>
              Mdmlab hit Apple’s API rate limit when preparing the macOS Setup
              Assistant for this node. Mdmlab will try again every hour.
            </span>
          </ReactTooltip>
        </>
      )}
    </span>
  );
};

export default NodeMdmStatusCell;
