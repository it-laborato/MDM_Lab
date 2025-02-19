import React from "react";

import { ActivityType, INodeUpcomingActivity } from "interfaces/activity";
import { INodeUpcomingActivitiesResponse } from "services/entities/activities";

// @ts-ignore
import MdmlabIcon from "components/icons/MdmlabIcon";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import { ShowActivityDetailsHandler } from "components/ActivityItem/ActivityItem";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import { upcomingActivityComponentMap } from "../ActivityConfig";

const baseClass = "upcoming-activity-feed";

interface IUpcomingActivityFeedProps {
  activities?: INodeUpcomingActivitiesResponse;
  isError?: boolean;
  onShowDetails: ShowActivityDetailsHandler;
  onCancel: (activity: INodeUpcomingActivity) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const UpcomingActivityFeed = ({
  activities,
  isError = false,
  onShowDetails,
  onCancel,
  onNextPage,
  onPreviousPage,
}: IUpcomingActivityFeedProps) => {
  if (isError) {
    return <DataError />;
  }

  if (!activities) {
    return null;
  }

  const { activities: activitiesList, meta } = activities;

  if (activitiesList === null || activitiesList.length === 0) {
    return (
      <EmptyFeed
        title="No pending activity "
        message="Pending actions will appear here (scripts, software, lock, and wipe)."
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__feed-list`}>
        {activitiesList.map((activity: INodeUpcomingActivity) => {
          // TODO: remove this once we have a proper way of handling "Mdmlab-initiated" activities in
          // the backend. For now, if all these fields are empty, then we assume it was
          // Mdmlab-initiated.
          if (
            !activity.actor_email &&
            !activity.actor_full_name &&
            (activity.type === ActivityType.InstalledSoftware ||
              activity.type === ActivityType.InstalledAppStoreApp ||
              activity.type === ActivityType.RanScript)
          ) {
            activity.actor_full_name = "Mdmlab";
          }
          const ActivityItemComponent =
            upcomingActivityComponentMap[activity.type];
          return (
            <ActivityItemComponent
              key={activity.id}
              tab="upcoming"
              activity={activity}
              onShowDetails={onShowDetails}
              hideCancel // TODO: remove this when canceling is implemented in API
              onCancel={() => onCancel(activity)}
            />
          );
        })}
      </div>
      <div className={`${baseClass}__pagination`}>
        <Button
          disabled={!meta.has_previous_results}
          onClick={onPreviousPage}
          variant="unstyled"
          className={`${baseClass}__load-activities-button`}
        >
          <>
            <MdmlabIcon name="chevronleft" /> Previous
          </>
        </Button>
        <Button
          disabled={!meta.has_next_results}
          onClick={onNextPage}
          variant="unstyled"
          className={`${baseClass}__load-activities-button`}
        >
          <>
            Next <MdmlabIcon name="chevronright" />
          </>
        </Button>
      </div>
    </div>
  );
};

export default UpcomingActivityFeed;
