import React, { useContext } from "react";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";
import { InjectedRouter } from "react-router";

const baseClass = "turn-on-mdm-message";

interface ITurnOnMdmMessageProps {
  router: InjectedRouter;
  /** Default: Manage your nodes */
  header?: string;
  /** Default: MDM must be turned on to change settings on your nodes. */
  info?: string;
  buttonText?: string;
}

const TurnOnMdmMessage = ({
  router,
  header,
  info,
  buttonText = "Turn on",
}: ITurnOnMdmMessageProps) => {
  const { isGlobalAdmin } = useContext(AppContext);

  const onConnectClick = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  const renderConnectButton = () => {
    return isGlobalAdmin ? (
      <Button
        variant="brand"
        onClick={onConnectClick}
        className={`${baseClass}__connectAPC-button`}
      >
        {buttonText}
      </Button>
    ) : (
      <></>
    );
  };

  return (
    <EmptyTable
      className={baseClass}
      header={header || "Manage your nodes"}
      info={info || "MDM must be turned on to change settings on your nodes."}
      primaryButton={renderConnectButton()}
    />
  );
};

export default TurnOnMdmMessage;
