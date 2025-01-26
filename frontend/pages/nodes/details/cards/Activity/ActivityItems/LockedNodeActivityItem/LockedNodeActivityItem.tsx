import React from "react";

import ActivityItem from "components/ActivityItem";

import { INodeActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "locked-node-activity-item";

const LockedNodeActivityItem = ({
  activity,
}: INodeActivityItemComponentProps) => {
  return (
    <ActivityItem className={baseClass} activity={activity}>
      <b>{activity.actor_full_name}</b> locked this node.
    </ActivityItem>
  );
};

export default LockedNodeActivityItem;
