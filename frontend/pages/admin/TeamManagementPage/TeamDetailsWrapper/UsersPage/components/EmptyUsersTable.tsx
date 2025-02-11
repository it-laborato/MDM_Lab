import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import PATHS from "router/paths";

interface IEmptyUsersTableProps {
  className: string;
  searchString: string;
  isGlobalAdmin: boolean;
  isTeamAdmin: boolean;
  toggleAddUserModal: () => void;
  toggleCreateMemberModal: () => void;
}



const CreateUserButton = ({
  className,
  isGlobalAdmin,
  isTeamAdmin,
  toggleAddUserModal,
  toggleCreateMemberModal,
}: Omit<IEmptyUsersTableProps, "searchString">) => {
  if (!isGlobalAdmin && !isTeamAdmin) {
    return null;
  }

  if (isGlobalAdmin) {
    return (
      <Button
        variant="brand"
        className={`${className}__create-button`}
        onClick={toggleAddUserModal}
      >
        Add user
      </Button>
    );
  }

  return (
    <Button
      variant="brand"
      className={`${className}__create-button`}
      onClick={toggleCreateMemberModal}
    >
      Create user
    </Button>
  );
};

const EmptyMembersTable = ({
  className,
  isGlobalAdmin,
  isTeamAdmin,
  searchString,
  toggleAddUserModal,
  toggleCreateMemberModal,
}: IEmptyUsersTableProps) => {
  if (searchString !== "") {
    return (
      <EmptyTable
        header="No users match the current criteria"
        info="Expecting to see users? Try again in a few seconds as the system catches up."
      />
    );
  }

  return (
    <EmptyTable
      graphicName="empty-users"
      header="No users on this team"
      
      primaryButton={
        <CreateUserButton
          className={className}
          isGlobalAdmin={isGlobalAdmin}
          isTeamAdmin={isTeamAdmin}
          toggleAddUserModal={toggleAddUserModal}
          toggleCreateMemberModal={toggleCreateMemberModal}
        />
      }
    />
  );
};

export default EmptyMembersTable;
