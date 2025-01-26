import React, { useMemo } from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { INodeMdmData } from "interfaces/node";

import OSSettingsTable from "./OSSettingsTable";
import { generateTableData } from "./OSSettingsTable/OSSettingsTableConfig";

interface IOSSettingsModalProps {
  nodeId: number;
  platform: string;
  nodeMDMData: INodeMdmData;
  /** controls showing the action for a user to resend a profile. Defaults to `false` */
  canResendProfiles?: boolean;
  onClose: () => void;
  /** handler that fires when a profile was reset. Requires `canResendProfiles` prop
   * to be `true`, otherwise has no effect.
   */
  onProfileResent?: () => void;
}

const baseClass = "os-settings-modal";

const OSSettingsModal = ({
  nodeId,
  platform,
  nodeMDMData,
  canResendProfiles = false,
  onClose,
  onProfileResent,
}: IOSSettingsModalProps) => {
  // the caller should ensure that nodeMDMData is not undefined and that platform is supported otherwise we will allow an empty modal will be rendered.
  // https://mdmlabdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks

  const memoizedTableData = useMemo(
    () => generateTableData(nodeMDMData, platform),
    [nodeMDMData, platform]
  );

  return (
    <Modal
      title="OS settings"
      onExit={onClose}
      className={baseClass}
      width="xlarge"
    >
      <>
        <OSSettingsTable
          canResendProfiles={canResendProfiles}
          nodeId={nodeId}
          tableData={memoizedTableData ?? []}
          onProfileResent={onProfileResent}
        />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onClose}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default OSSettingsModal;
