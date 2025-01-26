import { MdmProfileStatus } from "interfaces/mdm";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

interface IAggregateDisplayOption {
  value: MdmProfileStatus;
  text: string;
  iconName: IndicatorStatus;
  tooltipText: string;
}

const AGGREGATE_STATUS_DISPLAY_OPTIONS: IAggregateDisplayOption[] = [
  {
    value: "verified",
    text: "Verified",
    iconName: "success",
    tooltipText: "These nodes applied all OS settings. Mdmlab verified.",
  },
  {
    value: "verifying",
    text: "Verifying",
    iconName: "successPartial",
    tooltipText:
      "These nodes acknowledged all MDM commands to apply OS settings. " +
      "Mdmlab is verifying the OS settings are applied with osquery.",
  },
  {
    value: "pending",
    text: "Pending",
    iconName: "pendingPartial",
    tooltipText:
      "These nodes will apply the latest OS settings. Click on a node to view which settings.",
  },
  {
    value: "failed",
    text: "Failed",
    iconName: "error",
    tooltipText:
      "These node failed to apply the latest OS settings. Click on a node to view error(s).",
  },
];

export default AGGREGATE_STATUS_DISPLAY_OPTIONS;
