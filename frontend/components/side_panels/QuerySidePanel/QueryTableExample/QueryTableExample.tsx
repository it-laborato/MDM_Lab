import React from "react";

import MdmlabMarkdown from "components/MdmlabMarkdown";

interface IQueryTableExampleProps {
  example: string;
}

const baseClass = "query-table-example";

const QueryTableExample = ({ example }: IQueryTableExampleProps) => {
  return (
    <div className={baseClass}>
      <h3>Example</h3>
      <MdmlabMarkdown markdown={example} name="query-table-example" />
    </div>
  );
};

export default QueryTableExample;
