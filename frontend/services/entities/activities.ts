import endpoints from "utilities/endpoints";
import {
  IActivity,
  INodePastActivity,
  INodeUpcomingActivity,
} from "interfaces/activity";
import sendRequest from "services";
import { buildQueryStringFromParams } from "utilities/url";

const DEFAULT_PAGE = 0;
const DEFAULT_PAGE_SIZE = 8;
const ORDER_KEY = "created_at";
const ORDER_DIRECTION = "desc";

export interface IActivitiesResponse {
  activities: IActivity[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface INodePastActivitiesResponse {
  activities: INodePastActivity[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface INodeUpcomingActivitiesResponse {
  count: number;
  activities: INodeUpcomingActivity[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export default {
  loadNext: (
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<IActivitiesResponse> => {
    const { ACTIVITIES } = endpoints;

    const queryParams = {
      page,
      per_page: perPage,
      order_key: ORDER_KEY,
      order_direction: ORDER_DIRECTION,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${ACTIVITIES}?${queryString}`;

    return sendRequest("GET", path);
  },

  getNodePastActivities: (
    id: number,
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<INodePastActivitiesResponse> => {
    const { HOST_PAST_ACTIVITIES } = endpoints;

    const queryParams = {
      page,
      per_page: perPage,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${HOST_PAST_ACTIVITIES(id)}?${queryString}`;

    return sendRequest("GET", path);
  },

  getNodeUpcomingActivities: (
    id: number,
    page = DEFAULT_PAGE,
    perPage = DEFAULT_PAGE_SIZE
  ): Promise<INodeUpcomingActivitiesResponse> => {
    const { HOST_UPCOMING_ACTIVITIES } = endpoints;

    const queryParams = {
      page,
      per_page: perPage,
    };

    const queryString = buildQueryStringFromParams(queryParams);

    const path = `${HOST_UPCOMING_ACTIVITIES(id)}?${queryString}`;

    return sendRequest("GET", path);
  },

  cancelNodeActivity: (nodeId: number, uuid: string) => {
    const { HOST_CANCEL_ACTIVITY } = endpoints;
    return sendRequest("DELETE", HOST_CANCEL_ACTIVITY(nodeId, uuid));
  },
};
