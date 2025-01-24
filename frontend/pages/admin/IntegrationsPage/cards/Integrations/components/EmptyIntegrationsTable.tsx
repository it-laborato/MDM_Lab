import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

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
      info="Create tickets automatically when Mdmlab detects new software vulnerabilities or hosts failing policies."
      additionalInfo={
        <>
          Want to learn more?&nbsp;
          <CustomLink
            url="https://mdmlabdm.com/docs/using-mdmlab/automations"
            text="Read about automations"
            newTab
          />
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
