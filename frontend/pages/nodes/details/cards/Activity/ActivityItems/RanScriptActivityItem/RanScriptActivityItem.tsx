import React from "react";

import { formatScriptNameForActivityItem } from "utilities/helpers";

import ActivityItem from "components/ActivityItem";
import { INodeActivityItemComponentPropsWithShowDetails } from "../../ActivityConfig";

const baseClass = "ran-script-activity-item";

const RanScriptActivityItem = ({
  tab,
  activity,
  onShowDetails,
  onCancel,
  isSoloActivity,
  hideCancel,
}: INodeActivityItemComponentPropsWithShowDetails) => {
  const ranScriptPrefix = tab === "past" ? "ran" : "told Mdmlab to run";

  return (
    <ActivityItem
      className={baseClass}
      activity={activity}
      onShowDetails={onShowDetails}
      onCancel={onCancel}
      isSoloActivity={isSoloActivity}
      hideCancel={hideCancel}
    >
      <b>{activity.actor_full_name}</b>
      <>
        {" "}
        {ranScriptPrefix}{" "}
        {formatScriptNameForActivityItem(activity.details?.script_name)} on this
        node.{" "}
      </>
    </ActivityItem>
  );
};

export default RanScriptActivityItem;
