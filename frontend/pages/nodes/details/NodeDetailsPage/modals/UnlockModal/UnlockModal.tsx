import React, { useContext } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import nodeAPI, { IUnlockNodeResponse } from "services/entities/nodes";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

const baseClass = "unlock-modal";

interface IUnlockModalProps {
  id: number;
  platform: string;
  nodeName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const UnlockModal = ({
  id,
  platform,
  nodeName,
  onSuccess,
  onClose,
}: IUnlockModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUnlocking, setIsUnlocking] = React.useState(false);

  const {
    data: macUnlockData,
    isLoading: macIsLoading,
    isError: macIsError,
  } = useQuery<IUnlockNodeResponse, AxiosError>(
    ["mac-unlock-pin", id],
    () => nodeAPI.unlockNode(id),
    {
      enabled: platform === "darwin",
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
      retry: false,
    }
  );

  const onUnlock = async () => {
    setIsUnlocking(true);
    try {
      await nodeAPI.unlockNode(id);
      onSuccess();
      renderFlash(
        "success",
        "Unlocking node or will unlock when it comes online."
      );
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    onClose();
    setIsUnlocking(false);
  };

  const renderModalContent = () => {
    if (platform === "darwin") {
      if (macIsLoading) return <Spinner />;
      if (macIsError) return <DataError />;

      if (!macUnlockData) return null;

      return (
        <>
          {/* TODO: replace with DataSet component */}
          <p>
            When the node is returned, use the 6-digit PIN to unlock the node.
          </p>
          <div className={`${baseClass}__pin`}>
            <b>PIN</b>
            <span>{macUnlockData.unlock_pin}</span>
          </div>
        </>
      );
    }

    return (
      <>
        <p>
          Are you sure you&apos;re ready to unlock <b>{nodeName}</b>?
        </p>
      </>
    );
  };

  const renderModalButtons = () => {
    if (platform === "darwin") {
      return (
        <>
          <Button type="button" onClick={onClose} variant="brand">
            Done
          </Button>
        </>
      );
    }

    return (
      <>
        <Button
          type="button"
          onClick={onUnlock}
          variant="brand"
          className="delete-loading"
          isLoading={isUnlocking}
        >
          Unlock
        </Button>
        <Button onClick={onClose} variant="inverse">
          Cancel
        </Button>
      </>
    );
  };

  return (
    <Modal className={baseClass} title="Unlock node" onExit={onClose}>
      <>
        <div className={`${baseClass}__modal-content`}>
          {renderModalContent()}
        </div>

        <div className="modal-cta-wrap">{renderModalButtons()}</div>
      </>
    </Modal>
  );
};

export default UnlockModal;
