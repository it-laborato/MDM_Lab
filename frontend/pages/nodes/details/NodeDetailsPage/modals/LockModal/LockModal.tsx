import React, { useContext } from "react";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import nodeAPI from "services/entities/nodes";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "lock-modal";

interface ILockModalProps {
  id: number;
  platform: string;
  nodeName: string;
  onSuccess: () => void;
  onClose: () => void;
}

const LockModal = ({
  id,
  platform,
  nodeName,
  onSuccess,
  onClose,
}: ILockModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [lockChecked, setLockChecked] = React.useState(false);
  const [isLocking, setIsLocking] = React.useState(false);

  const onLock = async () => {
    setIsLocking(true);
    try {
      await nodeAPI.lockNode(id);
      onSuccess();
      renderFlash("success", "Locking node or will lock when it comes online.");
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    onClose();
    setIsLocking(false);
  };

  return (
    <Modal className={baseClass} title="Lock node" onExit={onClose}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <p>Lock a node when it needs to be returned to your organization.</p>
          {platform === "darwin" && (
            <p>Mdmlab will generate a six-digit unlock PIN.</p>
          )}
          <div className={`${baseClass}__confirm-message`}>
            <span>
              <b>Please check to confirm:</b>
            </span>
            <Checkbox
              wrapperClassName={`${baseClass}__lock-checkbox`}
              value={lockChecked}
              onChange={(value: boolean) => setLockChecked(value)}
            >
              I wish to lock <b>{nodeName}</b>
            </Checkbox>
          </div>
        </div>

        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onLock}
            variant="brand"
            className="delete-loading"
            disabled={!lockChecked}
            isLoading={isLocking}
          >
            Done
          </Button>
          <Button onClick={onClose} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default LockModal;
