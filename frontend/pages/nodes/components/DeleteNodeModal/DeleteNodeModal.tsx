import React from "react";

import strUtils from "utilities/strings";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "delete-node-modal";

interface IDeleteNodeModalProps {
  onSubmit: () => void;
  onCancel: () => void;
  /** Manage node page only */
  isAllMatchingNodesSelected?: boolean;
  /** Manage node page only */
  selectedNodeIds?: number[];
  /** Manage node page only */
  nodesCount?: number;
  /** Node details page only */
  nodeName?: string;
  isUpdating: boolean;
}

const DeleteNodeModal = ({
  onSubmit,
  onCancel,
  isAllMatchingNodesSelected,
  selectedNodeIds,
  nodesCount,
  nodeName,
  isUpdating,
}: IDeleteNodeModalProps): JSX.Element => {
  const pluralizeNode = () => {
    if (!selectedNodeIds) {
      return "node";
    }
    return strUtils.pluralize(selectedNodeIds.length, "node");
  };

  const nodeText = () => {
    if (selectedNodeIds) {
      return `${selectedNodeIds.length}${
        isAllMatchingNodesSelected ? "+" : ""
      } ${pluralizeNode()}`;
    }
    return nodeName;
  };
  const largeVolumeText = (): string => {
    if (
      selectedNodeIds &&
      isAllMatchingNodesSelected &&
      nodesCount &&
      nodesCount >= 500
    ) {
      return " When deleting a large volume of nodes, it may take some time for this change to be reflected in the UI.";
    }
    return "";
  };

  return (
    <Modal title="Delete node" onExit={onCancel} className={baseClass}>
      <>
        <p>
          This will remove the record of <b>{nodeText()}</b> and associated data
          (e.g. unlock PINs).{largeVolumeText()}
        </p>
        <ul>
          <li>
            Windows nodes will re-appear unless Mdmlab&apos;s
            agent is uninstalled.{" "}
          </li>
        </ul>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdating}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteNodeModal;
