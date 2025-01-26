import React, { useContext } from "react";

import { MdmEnrollmentStatus } from "interfaces/mdm";
import permissions from "utilities/permissions";
import { AppContext } from "context/app";

import ActionsDropdown from "components/ActionsDropdown";
import { generateNodeActionOptions } from "./helpers";
import { NodeMdmDeviceStatusUIState } from "../../helpers";

const baseClass = "node-actions-dropdown";

interface INodeActionsDropdownProps {
  nodeTeamId: number | null;
  nodeStatus: string;
  nodeMdmEnrollmentStatus: MdmEnrollmentStatus | null;
  /** This represents the mdm managed node device status (e.g. unlocked, locked,
   * unlocking, locking, ...etc) */
  nodeMdmDeviceStatus: NodeMdmDeviceStatusUIState;
  doesStoreEncryptionKey?: boolean;
  isConnectedToMdmlabMdm?: boolean;
  nodePlatform?: string;
  onSelect: (value: string) => void;
  nodeScriptsEnabled: boolean | null;
}

const NodeActionsDropdown = ({
  nodeTeamId,
  nodeStatus,
  nodeMdmEnrollmentStatus,
  nodeMdmDeviceStatus,
  doesStoreEncryptionKey,
  isConnectedToMdmlabMdm,
  nodePlatform = "",
  nodeScriptsEnabled = false,
  onSelect,
}: INodeActionsDropdownProps) => {
  const {
    isPremiumTier = false,
    isGlobalAdmin = false,
    isGlobalMaintainer = false,
    isMacMdmEnabledAndConfigured = false,
    isWindowsMdmEnabledAndConfigured = false,
    currentUser,
  } = useContext(AppContext);

  if (!currentUser) return null;

  const isTeamAdmin = permissions.isTeamAdmin(currentUser, nodeTeamId);
  const isTeamMaintainer = permissions.isTeamMaintainer(
    currentUser,
    nodeTeamId
  );
  const isTeamObserver = permissions.isTeamObserver(currentUser, nodeTeamId);
  const isGlobalObserver = permissions.isGlobalObserver(currentUser);

  const options = generateNodeActionOptions({
    nodePlatform,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalObserver,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamObserver,
    isNodeOnline: nodeStatus === "online",
    isEnrolledInMdm: ["On (automatic)", "On (manual)"].includes(
      nodeMdmEnrollmentStatus ?? ""
    ),
    isConnectedToMdmlabMdm,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    doesStoreEncryptionKey: doesStoreEncryptionKey ?? false,
    nodeMdmDeviceStatus,
    nodeScriptsEnabled,
  });

  // No options to render. Exit early
  if (options.length === 0) return null;

  return (
    <div className={baseClass}>
      <ActionsDropdown
        className={`${baseClass}__node-actions-dropdown`}
        onChange={onSelect}
        placeholder="Actions"
        options={options}
        menuAlign="right"
      />
    </div>
  );
};

export default NodeActionsDropdown;
