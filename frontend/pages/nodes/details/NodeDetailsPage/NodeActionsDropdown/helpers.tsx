import React from "react";
import { cloneDeep } from "lodash";

import { IDropdownOption } from "interfaces/dropdownOption";
import { isLinuxLike, isAppleDevice } from "interfaces/platform";
import { isScriptSupportedPlatform } from "interfaces/script";

import {
  NodeMdmDeviceStatusUIState,
  isDeviceStatusUpdating,
} from "../../helpers";

const DEFAULT_OPTIONS = [
  {
    label: "Transfer",
    value: "transfer",
    disabled: false,
    premiumOnly: true,
  },
  {
    label: "Query",
    value: "query",
    disabled: false,
  },
  {
    label: "Run script",
    value: "runScript",
    disabled: false,
  },
  {
    label: "Show disk encryption key",
    value: "diskEncryption",
    disabled: false,
  },
  {
    label: "Turn off MDM",
    value: "mdmOff",
    disabled: false,
  },
  {
    label: "Lock",
    value: "lock",
    disabled: false,
  },
  {
    label: "Wipe",
    value: "wipe",
    disabled: false,
  },
  {
    label: "Unlock",
    value: "unlock",
    disabled: false,
  },
  {
    label: "Delete",
    disabled: false,
    value: "delete",
  },
] as const;

// eslint-disable-next-line import/prefer-default-export
interface INodeActionConfigOptions {
  nodePlatform: string;
  isPremiumTier: boolean;
  isGlobalAdmin: boolean;
  isGlobalMaintainer: boolean;
  isGlobalObserver: boolean;
  isTeamAdmin: boolean;
  isTeamMaintainer: boolean;
  isTeamObserver: boolean;
  isNodeOnline: boolean;
  isEnrolledInMdm: boolean;
  isConnectedToMdmlabMdm?: boolean;
  isMacMdmEnabledAndConfigured: boolean;
  isWindowsMdmEnabledAndConfigured: boolean;
  doesStoreEncryptionKey: boolean;
  nodeMdmDeviceStatus: NodeMdmDeviceStatusUIState;
  nodeScriptsEnabled: boolean | null;
}

const canTransferTeam = (config: INodeActionConfigOptions) => {
  const { isPremiumTier, isGlobalAdmin, isGlobalMaintainer } = config;
  return isPremiumTier && (isGlobalAdmin || isGlobalMaintainer);
};

const canEditMdm = (config: INodeActionConfigOptions) => {
  const {
    nodePlatform,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isEnrolledInMdm,
    isConnectedToMdmlabMdm,
    isMacMdmEnabledAndConfigured,
  } = config;
  return (
    nodePlatform === "darwin" &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm &&
    isConnectedToMdmlabMdm &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canQueryNode = ({ nodePlatform }: INodeActionConfigOptions) => {
  // Currently we cannot query iOS or iPadOS
  const isIosOrIpadosNode = nodePlatform === "ios" || nodePlatform === "ipados";

  return !isIosOrIpadosNode;
};

const canLockNode = ({
  isPremiumTier,
  nodePlatform,
  isMacMdmEnabledAndConfigured,
  isEnrolledInMdm,
  isConnectedToMdmlabMdm,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  nodeMdmDeviceStatus,
}: INodeActionConfigOptions) => {
  // macOS nodes can be locked if they are enrolled in MDM and the MDM is enabled
  const canLockDarwin =
    nodePlatform === "darwin" &&
    isConnectedToMdmlabMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  return (
    isPremiumTier &&
    nodeMdmDeviceStatus === "unlocked" &&
    (nodePlatform === "windows" ||
      isLinuxLike(nodePlatform) ||
      canLockDarwin) &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canWipeNode = ({
  isPremiumTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  isConnectedToMdmlabMdm,
  isEnrolledInMdm,
  isMacMdmEnabledAndConfigured,
  isWindowsMdmEnabledAndConfigured,
  nodePlatform,
  nodeMdmDeviceStatus,
}: INodeActionConfigOptions) => {
  const nodeMdmEnabled =
    (isAppleDevice(nodePlatform) && isMacMdmEnabledAndConfigured) ||
    (nodePlatform === "windows" && isWindowsMdmEnabledAndConfigured);

  // Windows and Apple devices (i.e. macOS, iOS, iPadOS) have the same conditions and can be wiped if they
  // are enrolled in MDM and the MDM is enabled.
  const canWipeWindowsOrAppleOS =
    nodeMdmEnabled && isConnectedToMdmlabMdm && isEnrolledInMdm;

  return (
    isPremiumTier &&
    nodeMdmDeviceStatus === "unlocked" &&
    (isLinuxLike(nodePlatform) || canWipeWindowsOrAppleOS) &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer)
  );
};

const canUnlock = ({
  isPremiumTier,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
  isConnectedToMdmlabMdm,
  isEnrolledInMdm,
  isMacMdmEnabledAndConfigured,
  nodePlatform,
  nodeMdmDeviceStatus,
}: INodeActionConfigOptions) => {
  const canUnlockDarwin =
    nodePlatform === "darwin" &&
    isConnectedToMdmlabMdm &&
    isMacMdmEnabledAndConfigured &&
    isEnrolledInMdm;

  // "unlocking" for a macOS node means that somebody saw the unlock pin, but
  // shouldn't prevent users from trying to see the pin again, which is
  // considered an "unlock"
  const isValidState =
    (nodeMdmDeviceStatus === "unlocking" && nodePlatform === "darwin") ||
    nodeMdmDeviceStatus === "locked";

  return (
    isPremiumTier &&
    isValidState &&
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer) &&
    (canUnlockDarwin || nodePlatform === "windows" || isLinuxLike(nodePlatform))
  );
};

const canDeleteNode = (config: INodeActionConfigOptions) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = config;
  return isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;
};

const canShowDiskEncryption = (config: INodeActionConfigOptions) => {
  const { isPremiumTier, doesStoreEncryptionKey, nodePlatform } = config;

  // Currently we cannot show disk encryption key for iOS or iPadOS
  const isIosOrIpadosNode = nodePlatform === "ios" || nodePlatform === "ipados";

  return isPremiumTier && doesStoreEncryptionKey && !isIosOrIpadosNode;
};

const canRunScript = ({
  nodePlatform,
  isGlobalAdmin,
  isGlobalMaintainer,
  isTeamAdmin,
  isTeamMaintainer,
}: INodeActionConfigOptions) => {
  return (
    (isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer) &&
    isScriptSupportedPlatform(nodePlatform)
  );
};

const removeUnavailableOptions = (
  options: IDropdownOption[],
  config: INodeActionConfigOptions
) => {
  if (!canTransferTeam(config)) {
    options = options.filter((option) => option.value !== "transfer");
  }

  if (!canQueryNode(config)) {
    options = options.filter((option) => option.value !== "query");
  }

  if (!canShowDiskEncryption(config)) {
    options = options.filter((option) => option.value !== "diskEncryption");
  }

  if (!canEditMdm(config)) {
    options = options.filter((option) => option.value !== "mdmOff");
  }

  if (!canDeleteNode(config)) {
    options = options.filter((option) => option.value !== "delete");
  }

  if (!canRunScript(config)) {
    options = options.filter((option) => option.value !== "runScript");
  }

  if (!canLockNode(config)) {
    options = options.filter((option) => option.value !== "lock");
  }

  if (!canWipeNode(config)) {
    options = options.filter((option) => option.value !== "wipe");
  }

  if (!canUnlock(config)) {
    options = options.filter((option) => option.value !== "unlock");
  }

  // TODO: refactor to filter in one pass using predefined filters specified for each of the
  // DEFAULT_OPTIONS. Note that as currently, structured the default is to include all options.
  // This is a bit confusing since we remove options instead of add options

  return options;
};

// Available tooltips for disabled options
export const getDropdownOptionTooltipContent = (
  value: string | number,
  isNodeOnline?: boolean
) => {
  const tooltipAction: Record<string, string> = {
    runScript: "run scripts on",
    wipe: "wipe",
    lock: "lock",
    unlock: "unlock",
    installSoftware: "install software on", // Node software dropdown option
    uninstallSoftware: "uninstall software on", // Node software dropdown option
  };
  if (tooltipAction[value]) {
    return (
      <>
        To {tooltipAction[value]} this node, deploy the
        <br />
        mdmlabd agent with --enable-scripts and
        <br />
        refetch node vitals
      </>
    );
  }
  if (!isNodeOnline && value === "query") {
    return <>You can&apos;t query an offline node.</>;
  }
  return undefined;
};

const modifyOptions = (
  options: IDropdownOption[],
  {
    isNodeOnline,
    nodeMdmDeviceStatus,
    nodeScriptsEnabled,
    nodePlatform,
  }: INodeActionConfigOptions
) => {
  const disableOptions = (optionsToDisable: IDropdownOption[]) => {
    optionsToDisable.forEach((option) => {
      option.disabled = true;
      option.tooltipContent = getDropdownOptionTooltipContent(
        option.value,
        isNodeOnline
      );
    });
  };

  let optionsToDisable: IDropdownOption[] = [];
  if (
    !isNodeOnline ||
    isDeviceStatusUpdating(nodeMdmDeviceStatus) ||
    nodeMdmDeviceStatus === "locked" ||
    nodeMdmDeviceStatus === "wiped"
  ) {
    optionsToDisable = optionsToDisable.concat(
      options.filter(
        (option) => option.value === "query" || option.value === "mdmOff"
      )
    );
  }

  // null intentionally excluded from this condition:
  // scripts_enabled === null means this agent is not an orbit agent, or this agent is version
  // <=1.23.0 which is not collecting the scripts enabled info
  // in each of these cases, we maintain these options
  if (nodeScriptsEnabled === false) {
    optionsToDisable = optionsToDisable.concat(
      options.filter((option) => option.value === "runScript")
    );
    if (isLinuxLike(nodePlatform)) {
      optionsToDisable = optionsToDisable.concat(
        options.filter(
          (option) =>
            option.value === "lock" ||
            option.value === "unlock" ||
            option.value === "wipe"
        )
      );
    }
    if (nodePlatform === "windows") {
      optionsToDisable = optionsToDisable.concat(
        options.filter(
          (option) => option.value === "lock" || option.value === "unlock"
        )
      );
    }
  }
  disableOptions(optionsToDisable);
  return options;
};

/**
 * Generates the node actions options depending on the configuration. There are
 * many variations of the options that are shown/not shown or disabled/enabled
 * which are all controlled by the configurations options argument.
 */
// eslint-disable-next-line import/prefer-default-export
export const generateNodeActionOptions = (config: INodeActionConfigOptions) => {
  // deep clone to always start with a fresh copy of the default options.
  let options: IDropdownOption[] = cloneDeep([...DEFAULT_OPTIONS]);
  options = removeUnavailableOptions(options, config);

  if (options.length === 0) return options;

  options = modifyOptions(options, config);

  return options;
};
