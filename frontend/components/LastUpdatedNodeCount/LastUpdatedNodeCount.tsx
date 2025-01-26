import React from "react";
import LastUpdatedText from "components/LastUpdatedText";

const baseClass = "last-updated-node-count";

interface ILastUpdatedNodeCount {
  nodeCount?: string | number;
  lastUpdatedAt?: string;
}

const LastUpdatedNodeCount = ({
  nodeCount,
  lastUpdatedAt,
}: ILastUpdatedNodeCount): JSX.Element => {
  const tooltipContent = (
    <>
      The last time node data was updated. <br />
      Click <b>View all nodes</b> to see the most
      <br /> up-to-date node count.
    </>
  );

  return (
    <div className={baseClass}>
      <>{nodeCount}</>
      <LastUpdatedText
        lastUpdatedAt={lastUpdatedAt}
        customTooltipText={tooltipContent}
      />
    </div>
  );
};

export default LastUpdatedNodeCount;
