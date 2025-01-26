import React from "react";

import {
  ActivityType,
  INodePastActivityType,
  INodePastActivity,
  INodeUpcomingActivityType,
  INodeUpcomingActivity,
} from "interfaces/activity";

import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import RanScriptActivityItem from "./ActivityItems/RanScriptActivityItem";
import LockedNodeActivityItem from "./ActivityItems/LockedNodeActivityItem";
import UnlockedNodeActivityItem from "./ActivityItems/UnlockedNodeActivityItem";
import InstalledSoftwareActivityItem from "./ActivityItems/InstalledSoftwareActivityItem";
import CanceledScriptActivityItem from "./ActivityItems/CanceledScriptActivityItem";
import CanceledSoftwareInstallActivityItem from "./ActivityItems/CanceledSoftwareInstallActivityItem";

/** The component props that all node activity items must adhere to */
export interface INodeActivityItemComponentProps {
  activity: INodePastActivity | INodeUpcomingActivity;
  tab: "past" | "upcoming";
  /** Set this to `true` when rendering only this activity by itself. This will
   * change the styles for the activity item for solo rendering.
   * @default false */
  isSoloActivity?: boolean;
  /** Set this to `true` to hide the close button and prevent from rendering
   * @default false
   */
  hideCancel?: boolean;
}

/** Used for activity items component that need a show details handler */
export interface INodeActivityItemComponentPropsWithShowDetails
  extends INodeActivityItemComponentProps {
  onShowDetails: ShowActivityDetailsHandler;
  onCancel?: () => void;
}

export const pastActivityComponentMap: Record<
  INodePastActivityType,
  | React.FC<INodeActivityItemComponentProps>
  | React.FC<INodeActivityItemComponentPropsWithShowDetails>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.LockedNode]: LockedNodeActivityItem,
  [ActivityType.UnlockedNode]: UnlockedNodeActivityItem,
  [ActivityType.InstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.UninstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAppStoreApp]: InstalledSoftwareActivityItem,
  [ActivityType.CanceledScript]: CanceledScriptActivityItem,
  [ActivityType.CanceledSoftwareInstall]: CanceledSoftwareInstallActivityItem,
};

export const upcomingActivityComponentMap: Record<
  INodeUpcomingActivityType,
  | React.FC<INodeActivityItemComponentProps>
  | React.FC<INodeActivityItemComponentPropsWithShowDetails>
> = {
  [ActivityType.RanScript]: RanScriptActivityItem,
  [ActivityType.InstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.UninstalledSoftware]: InstalledSoftwareActivityItem,
  [ActivityType.InstalledAppStoreApp]: InstalledSoftwareActivityItem,
};
