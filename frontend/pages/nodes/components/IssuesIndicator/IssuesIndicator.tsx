import React from "react";

import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import Icon from "components/Icon";

interface IIssuesIndicatorProps {
  totalIssuesCount?: number;
  failingPoliciesCount?: number;
  /** Premium only */
  criticalVulnerabilitiesCount?: number;
  rowId?: number;
  tooltipPosition?: "top" | "bottom";
}

const IssuesIndicator = ({
  totalIssuesCount,
  failingPoliciesCount,
  criticalVulnerabilitiesCount,
  rowId,
  tooltipPosition = "top",
}: IIssuesIndicatorProps): JSX.Element => {
  return (
    <>
      <span
        className="node-issue tooltip tooltip__tooltip-icon"
        data-tip
        data-for={`node-issue-count-${rowId}`}
        data-tip-disable={false}
      >
        <Icon name="error-outline" color="ui-mdmlab-black-50" />{" "}
        {totalIssuesCount}
      </span>
      <ReactTooltip
        place={tooltipPosition}
        effect="solid"
        backgroundColor={COLORS["tooltip-bg"]}
        id={`node-issue-count-${rowId}`}
        data-html
      >
        <span className={`tooltip__tooltip-text`}>
          {!!criticalVulnerabilitiesCount &&
            `Critical vulnerabilities (${criticalVulnerabilitiesCount})`}
          {!!criticalVulnerabilitiesCount && !!failingPoliciesCount && <br />}
          {!!failingPoliciesCount &&
            `Failing policies (${failingPoliciesCount})`}
        </span>
      </ReactTooltip>
    </>
  );
};

export default IssuesIndicator;
