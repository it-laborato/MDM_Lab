import React from "react";

import Icon from "components/Icon/Icon";

const baseClass = "learn-mdmlab";

const LearnMdmlab = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <p>
        Want to explore Mdmlab&apos;s features? Learn how to ask questions about
        your device using queries.
      </p>
      <a
        className="dashboard-info-card__action-button"
        href="https://mdmlabdm.com/docs/using-mdmlab/learn-how-to-use-mdmlab"
        target="_blank"
        rel="noopener noreferrer"
      >
        Learn how to use Mdmlab
        <Icon name="arrow-internal-link" color="core-mdmlab-blue" />
      </a>
    </div>
  );
};

export default LearnMdmlab;
