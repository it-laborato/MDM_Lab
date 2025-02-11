import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";

const EmptyIntegrationsTable = ({
  className,
  onActionButtonClick,
}: {
  className: string;
  onActionButtonClick: () => void;
}) => {
  return (
    <EmptyTable
      graphicName="empty-integrations"
      header="Set up integrations"
      info="Create tickets automatically when Mdmlab detects new software vulnerabilities or nodes failing policies."
      additionalInfo={
        <>
          Want to learn more?&nbsp;
         
        </>
      }
      primaryButton={
        <Button
          variant="brand"
          className={`${className}__add-button`}
          onClick={onActionButtonClick}
        >
          Add integration
        </Button>
      }
    />
  );
};

export default EmptyIntegrationsTable;
