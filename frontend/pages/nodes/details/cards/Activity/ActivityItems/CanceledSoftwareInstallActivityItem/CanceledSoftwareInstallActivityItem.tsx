import React from "react";

import ActivityItem from "components/ActivityItem";
import { INodeActivityItemComponentProps } from "../../ActivityConfig";

const baseClass = "canceled-software-install-activity-item";

const CanceledSoftwareInstallActivityItem = ({
  activity,
}: INodeActivityItemComponentProps) => {
  return (
    <ActivityItem className={baseClass} activity={activity}>
      <>
        <b>{activity.actor_full_name}</b> canceled{" "}
        <b>{activity.details?.software_title}</b> install on this node.
      </>
    </ActivityItem>
  );
};

export default CanceledSoftwareInstallActivityItem;
