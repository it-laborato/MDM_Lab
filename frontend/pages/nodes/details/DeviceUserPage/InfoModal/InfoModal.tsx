import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

export interface IInfoModalProps {
  onCancel: () => void;
}

const baseClass = "device-user-info";

const InfoModal = ({ onCancel }: IInfoModalProps): JSX.Element => {
  return (
    <Modal
      title="Welcome to Mdmlab"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <div>
        <p>
          Your organization uses Mdmlab to check if all devices meet its security
          policies.
        </p>
        <p>With Mdmlab, you and your team can secure your device, together.</p>
        <p>
         
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            OK
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default InfoModal;
