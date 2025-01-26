import React from "react";

import { syntaxHighlight } from "utilities/helpers";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "node-status-webhook-preview-modal";

const getNodeStatusPreview = (teamScope?: boolean) => {
  const data = {
    unseen_nodes: 1,
    total_nodes: 2,
    days_unseen: 3,
    team_id: 123,
  } as Record<string, number>;

  if (!teamScope) {
    delete data.team_id;
  }

  return {
    text:
      "More than X% of your nodes have not checked into Mdmlab for more than Y days. You’ve been sent this message because the Node status webhook is enabled in your Mdmlab instance.",
    data,
  };
};

interface INodeStatusWebhookPreviewModal {
  isTeamScope?: boolean;
  toggleModal: () => void;
}

const NodeStatusWebhookPreviewModal = ({
  isTeamScope = false,
  toggleModal,
}: INodeStatusWebhookPreviewModal) => {
  return (
    <Modal
      title="Node status webhook"
      onExit={toggleModal}
      onEnter={toggleModal}
      className={baseClass}
    >
      <>
        <p>
          An example request sent to your configured <b>Destination URL</b>.
        </p>
        <div className={baseClass}>
          <pre
            dangerouslySetInnerHTML={{
              __html: syntaxHighlight(getNodeStatusPreview(isTeamScope)),
            }}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={toggleModal} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default NodeStatusWebhookPreviewModal;
