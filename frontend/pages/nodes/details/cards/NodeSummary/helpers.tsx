import React from "react";
import { isAppleDevice } from "interfaces/platform";
import { NodeMdmDeviceStatusUIState } from "../../helpers";

interface IDeviceStatusTag {
  title: string;
  tagType: "warning" | "error";
  generateTooltip: (platform: string) => string;
}

type NodeMdmDeviceStatusUIStateNoUnlock = Exclude<
  NodeMdmDeviceStatusUIState,
  "unlocked"
>;

// We exclude "unlocked" as we dont display any device status tag for it
type DeviceStatusTagConfig = Record<
  NodeMdmDeviceStatusUIStateNoUnlock,
  IDeviceStatusTag
>;

export const DEVICE_STATUS_TAGS: DeviceStatusTagConfig = {
  locked: {
    title: "LOCKED",
    tagType: "warning",
    generateTooltip: (platform) =>
      isAppleDevice(platform)
        ? "Node is locked. The end user can’t use the node until the six-digit PIN has been entered."
        : "Node is locked. The end user can’t use the node until the node has been unlocked.",
  },
  unlocking: {
    title: "UNLOCK PENDING",
    tagType: "warning",
    generateTooltip: () =>
      "Node will unlock when it comes online.  If the node is online, it will unlock the next time it checks in to Mdmlab.",
  },
  locking: {
    title: "LOCK PENDING",
    tagType: "warning",
    generateTooltip: () =>
      "Node will lock when it comes online.  If the node is online, it will lock the next time it checks in to Mdmlab.",
  },
  wiped: {
    title: "WIPED",
    tagType: "error",
    generateTooltip: (platform) =>
      isAppleDevice(platform)
        ? "Node is wiped. To prevent the node from automatically reenrolling to Mdmlab, first release the node from Apple Business Manager and then delete the node in Mdmlab."
        : "Node is wiped.",
  },
  wiping: {
    title: "WIPE PENDING",
    tagType: "error",
    generateTooltip: () =>
      "Node will wipe when it comes online. If the node is online, it will wipe the next time it checks in to Mdmlab.",
  },
};

// We exclude "unlocked" as we dont display a tooltip for it.
export const REFETCH_TOOLTIP_MESSAGES: Record<
  NodeMdmDeviceStatusUIStateNoUnlock | "offline",
  JSX.Element
> = {
  offline: (
    <>
      You can&apos;t fetch data from <br /> an offline node.
    </>
  ),
  unlocking: (
    <>
      You can&apos;t fetch data from <br /> an unlocking node.
    </>
  ),
  locking: (
    <>
      You can&apos;t fetch data from <br /> a locking node.
    </>
  ),
  locked: (
    <>
      You can&apos;t fetch data from <br /> a locked node.
    </>
  ),
  wiping: (
    <>
      You can&apos;t fetch data from <br /> a wiping node.
    </>
  ),
  wiped: (
    <>
      You can&apos;t fetch data from <br /> a wiped node.
    </>
  ),
} as const;
