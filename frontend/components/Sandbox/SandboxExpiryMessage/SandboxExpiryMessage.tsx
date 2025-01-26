import React from "react";
import { browserHistory } from "react-router";
import PATHS from "router/paths";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "sandbox-expiry-message";

interface ISandboxExpiryMessageProps {
  expiry: string;
  noSandboxNodes?: boolean;
}

const SandboxExpiryMessage = ({
  expiry,
  noSandboxNodes,
}: ISandboxExpiryMessageProps) => {
  const openAddNodeModal = () => {
    browserHistory.push(PATHS.MANAGE_HOSTS_ADD_HOSTS);
  };

  if (noSandboxNodes) {
    return (
      <div className={baseClass}>
        <p>Your Mdmlab Sandbox expires in {expiry}.</p>
        <div className={`${baseClass}__tip`}>
          <Icon name="lightbulb" size="large" />
          <p>
            <b>Quick tip: </b> Enroll a node to get started.
          </p>
          <form>
            <Button
              onClick={openAddNodeModal}
              className={`${baseClass}__add-nodes`}
              variant="brand"
            >
              <span>Add nodes</span>
            </Button>
          </form>
        </div>
      </div>
    );
  }

  return (
    <a
      href="https://mdmlabdm.com/docs/using-mdmlab/learn-how-to-use-mdmlab#how-to-add-your-device-to-mdmlab"
      target="_blank"
      rel="noreferrer"
      className={baseClass}
    >
      <p>Your Mdmlab Sandbox expires in {expiry}.</p>
      <p>
        <b>Learn how to use Mdmlab</b>{" "}
        <Icon name="external-link" color="core-mdmlab-black" />
      </p>
    </a>
  );
};

export default SandboxExpiryMessage;
