import React, { useCallback } from "react";

import { INodeUser } from "interfaces/node_users";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";

import generateUsersTableHeaders from "./UsersTable/UsersTableConfig";

interface IUsersProps {
  users: INodeUser[];
  usersState: { username: string }[];
  isLoading: boolean;
  onUsersTableSearchChange: (queryData: ITableQueryData) => void;
  nodeUsersEnabled?: boolean;
}

const baseClass = "users-card";

const Users = ({
  users,
  usersState,
  isLoading,
  onUsersTableSearchChange,
  nodeUsersEnabled,
}: IUsersProps): JSX.Element => {
  const tableHeaders = generateUsersTableHeaders();

  const renderUsersCount = useCallback(() => {
    return <TableCount name="users" count={usersState.length} />;
  }, [usersState.length]);

  if (!nodeUsersEnabled) {
    return (
      <Card
        borderRadiusSize="xxlarge"
        includeShadow
        largePadding
        className={baseClass}
      >
        <p className="card__header">Users</p>
        <EmptyTable
          header="User collection has been disabled"
          info={
            <>
              Check out the Mdmlab documentation 
            </>
          }
        />
      </Card>
    );
  }

  return (
    <Card
      borderRadiusSize="xxlarge"
      includeShadow
      largePadding
      className={baseClass}
    >
      <>
        <p className="card__header">Users</p>
        {users?.length ? (
          <TableContainer
            columnConfigs={tableHeaders}
            data={usersState}
            isLoading={isLoading}
            defaultSortHeader="username"
            defaultSortDirection="asc"
            inputPlaceHolder="Search users by username"
            onQueryChange={onUsersTableSearchChange}
            emptyComponent={() => (
              <EmptyTable
                header="No users match your search criteria"
                info="Try a different search."
              />
            )}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            searchable
            wideSearch
            renderCount={renderUsersCount}
            isClientSidePagination
          />
        ) : (
          <EmptyTable
            header="No users detected on this node"
            info="Expecting to see users? Try again in a few seconds as the system
              catches up."
          />
        )}
      </>
    </Card>
  );
};

export default Users;
