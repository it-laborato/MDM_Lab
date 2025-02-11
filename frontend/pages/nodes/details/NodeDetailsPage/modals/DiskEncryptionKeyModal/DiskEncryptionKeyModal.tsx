import React from "react";
import { useQuery } from "react-query";

import { INodeEncrpytionKeyResponse } from "interfaces/node";
import nodeAPI from "services/entities/nodes";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import DataError from "components/DataError";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { NodePlatform } from "interfaces/platform";

const baseClass = "disk-encryption-key-modal";

interface IDiskEncryptionKeyModal {
  platform: NodePlatform;
  nodeId: number;
  onCancel: () => void;
}

const DiskEncryptionKeyModal = ({
  platform,
  nodeId,
  onCancel,
}: IDiskEncryptionKeyModal) => {
  const { data: encryptionKey, error: encryptionKeyError } = useQuery<
    INodeEncrpytionKeyResponse,
    unknown,
    string
  >("nodeEncrpytionKey", () => nodeAPI.getEncryptionKey(nodeId), {
    refetchOnMount: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    retry: false,
    select: (data) => data.encryption_key.key,
  });

  const recoveryText =
    platform === "darwin"
      ? "Use this key to log in to the node if you forgot the password."
      : "Use this key to unlock the encrypted drive.";

  return (
    <Modal
      title="Disk encryption key"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {encryptionKeyError ? (
        <DataError />
      ) : (
        <>
          <InputFieldHiddenContent value={encryptionKey ?? ""} />
          <p>
            {recoveryText}{" "}
           
          </p>
          <div className="modal-cta-wrap">
            <Button onClick={onCancel}>Done</Button>
          </div>
        </>
      )}
    </Modal>
  );
};

export default DiskEncryptionKeyModal;
