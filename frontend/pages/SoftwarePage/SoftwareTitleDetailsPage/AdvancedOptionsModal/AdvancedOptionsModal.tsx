import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import MdmlabAce from "components/MdmlabAce";
import Editor from "components/Editor";

const baseClass = "advanced-options-modal";

interface IAdvancedOptionsModalProps {
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  onExit: () => void;
}

const AdvancedOptionsModal = ({
  installScript,
  preInstallQuery,
  postInstallScript,
  onExit,
}: IAdvancedOptionsModalProps) => {
  return (
    <Modal className={baseClass} title="Advanced options" onExit={onExit}>
      <>
        <p>
          Advanced options are read-only. To change options, delete software and
          add again.
        </p>
        <div className={`${baseClass}__form-inputs`}>
          <Editor
            readOnly
            wrapEnabled
            maxLines={10}
            name="install-script"
            value={installScript}
            helpText="Mdmlab will run this command on nodes to install software."
            label="Install script"
            labelTooltip="For security agents, add the script provided by the vendor."
            isFormField
          />
          {preInstallQuery && (
            <div className={`${baseClass}__input-field`}>
              <span>Pre-install condition:</span>
              <MdmlabAce
                readOnly
                value={preInstallQuery}
                label="Query"
                name="preInstallQuery"
                maxLines={10}
                helpText={
                  <>
                    Software will be installed
                   
                  </>
                }
              />
            </div>
          )}
          {postInstallScript && (
            <div className={`${baseClass}__input-field`}>
              <span>Post-install script:</span>
              <Editor
                readOnly
                wrapEnabled
                name="post-install-script-editor"
                maxLines={10}
                value={postInstallScript}
                helpText="Shell (macOS and Linux) or PowerShell (Windows)."
                isFormField
              />
            </div>
          )}
        </div>
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onExit}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AdvancedOptionsModal;
