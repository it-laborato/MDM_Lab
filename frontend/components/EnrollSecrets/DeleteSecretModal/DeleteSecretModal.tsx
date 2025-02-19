import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";

interface IDeleteSecretModal {
  selectedTeam: number;
  teams: ITeam[];
  onDeleteSecret: () => void;
  toggleDeleteSecretModal: () => void;
  isUpdatingSecret: boolean;
}

const baseClass = "delete-secret-modal";

const DeleteSecretModal = ({
  selectedTeam,
  teams,
  onDeleteSecret,
  toggleDeleteSecretModal,
  isUpdatingSecret,
}: IDeleteSecretModal): JSX.Element => {
  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team" };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  return (
    <Modal
      onExit={toggleDeleteSecretModal}
      onEnter={onDeleteSecret}
      title="Delete secret"
      className={baseClass}
    >
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          <p>Nodes can no longer enroll using this secret.</p>
          <p>
           
          </p>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onDeleteSecret}
            className="delete-loading"
            isLoading={isUpdatingSecret}
          >
            Delete
          </Button>
          <Button onClick={toggleDeleteSecretModal} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteSecretModal;
