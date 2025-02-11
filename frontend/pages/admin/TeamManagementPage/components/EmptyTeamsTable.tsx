import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";

const EmptyTeamsTable = ({
  className,
  onActionButtonClick,
}: {
  className: string;
  onActionButtonClick: () => void;
}) => {
  return (
    <EmptyTable
      graphicName="empty-teams"
      header="Set up team permissions"
      info="Keep your organization organized and efficient by ensuring every user has the correct access to the right nodes."
      additionalInfo={
        <>
          {" "}
          </>
      }
      primaryButton={
        <Button
          variant="brand"
          className={`${className}__create-button`}
          onClick={onActionButtonClick}
        >
          Create team
        </Button>
      }
    />
  );
};

export default EmptyTeamsTable;
