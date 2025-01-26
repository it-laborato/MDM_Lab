import React from "react";

import Card from "components/Card";

const baseClass = "details-no-nodes";

interface IDetailsNoNodes {
  header: string;
  details: string;
}

const DetailsNoNodes = ({ header, details }: IDetailsNoNodes) => {
  return (
    <Card borderRadiusSize="xxlarge" includeShadow className={baseClass}>
      <h2>{header}</h2>
      <p>{details}</p>
    </Card>
  );
};

export default DetailsNoNodes;
