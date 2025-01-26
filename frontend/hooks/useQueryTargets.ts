import { useQuery, UseQueryResult } from "react-query";
import { filter, uniqueId } from "lodash";

import { INode } from "interfaces/node";
import { ILabel } from "interfaces/label";
import { ITeam } from "interfaces/team";
import { ISelectedTargetsForApi } from "interfaces/target";
import targetsAPI from "services/entities/targets";

export interface ITargetsLabels {
  allNodesLabels?: ILabel[];
  platformLabels?: ILabel[];
  otherLabels?: ILabel[];
  teams?: ITeam[];
  labelCount?: number;
}

export interface ITargetsQueryResponse extends ITargetsLabels {
  targetsTotalCount: number;
  targetsOnlinePercent: number;
  relatedNodes?: INode[];
}

export interface ITargetsQueryKey {
  scope: string;
  query: string;
  queryId: number | null;
  selected: ISelectedTargetsForApi;
  includeLabels: boolean;
}

const getTargets = async (
  queryKey: ITargetsQueryKey
): Promise<ITargetsQueryResponse> => {
  const { query, queryId, selected, includeLabels } = queryKey;

  try {
    const {
      targets,
      targets_count: targetsTotalCount,
      targets_online: targetsOnline,
    } = await targetsAPI.loadAll({
      query,
      queryId,
      selected,
    });
    let responseLabels: ITargetsLabels = {};

    if (includeLabels) {
      const { labels } = targets;

      const all = filter(
        labels,
        ({ display_text: text }) => text === "All Nodes"
      ).map((label) => ({ ...label, uuid: uniqueId() }));

      const platforms = filter(
        labels,
        ({ display_text: text }) =>
          text === "macOS" || text === "MS Windows" || text === "All Linux"
      ).map((label) => ({ ...label, uuid: uniqueId() }));

      const other = filter(
        labels,
        ({ label_type: type }) => type === "regular"
      ).map((label) => ({ ...label, uuid: uniqueId() }));

      const teams = targets.teams.map((team) => ({
        ...team,
        uuid: uniqueId(),
      }));

      const labelCount =
        all.length + platforms.length + other.length + teams.length;

      responseLabels = {
        allNodesLabels: all,
        platformLabels: platforms,
        otherLabels: other,
        teams,
        labelCount,
      };
    }

    const targetsOnlinePercent =
      targetsTotalCount > 0
        ? Math.round((targetsOnline / targetsTotalCount) * 100)
        : 0;

    return Promise.resolve({
      ...responseLabels,
      relatedNodes: query ? [...targets.nodes] : [],
      targetsTotalCount,
      targetsOnlinePercent,
    });
  } catch (err) {
    return Promise.reject(err);
  }
};

export const useQueryTargets = (
  targetsQueryKey: ITargetsQueryKey[],
  options: {
    onSuccess: (data: ITargetsQueryResponse) => void;
    staleTime: number;
  }
): UseQueryResult<ITargetsQueryResponse, Error> => {
  return useQuery<
    ITargetsQueryResponse,
    Error,
    ITargetsQueryResponse,
    ITargetsQueryKey[]
  >(
    targetsQueryKey,
    ({ queryKey }) => {
      return getTargets(queryKey[0]);
    },
    {
      onSuccess: options.onSuccess,
      refetchOnWindowFocus: false,
      refetchOnMount: "always",
      staleTime: options.staleTime,
    }
  );
};

export default useQueryTargets;
