import React from "react";

import ActivityItem from "components/ActivityItem";
import { INodeActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "unlocked-node-activity-item";

const UnlockedNodeActivityItem = ({
  activity,
}: INodeActivityItemComponentProps) => {
  let desc = "unlocked this node.";
  if (activity.details?.node_platform === "darwin") {
    desc = "viewed the six-digit unlock PIN for this node.";
  }
  return (
    <ActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name} </b> {desc}
    </ActivityItem>
  );
};

export default UnlockedNodeActivityItem;
