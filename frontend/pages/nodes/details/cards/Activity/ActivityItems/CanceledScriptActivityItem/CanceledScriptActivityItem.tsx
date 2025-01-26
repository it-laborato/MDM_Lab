import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import ActivityItem from "components/ActivityItem";

import { INodeActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-script-activity-item";

const CanceledScriptActivityItem = ({
  activity,
}: INodeActivityItemComponentProps) => {
  return (
    <ActivityItem className={baseClass} activity={activity}>
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>{formatScriptNameForActivityItem(activity.details?.script_name)}</b>{" "}
        script on this node.
      </>
    </ActivityItem>
  );
};

export default CanceledScriptActivityItem;
