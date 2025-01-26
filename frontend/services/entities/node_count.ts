/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  DiskEncryptionStatus,
  BootstrapPackageStatus,
  MdmProfileStatus,
} from "interfaces/mdm";
import { NodeStatus } from "interfaces/node";
import {
  buildQueryStringFromParams,
  getLabelParam,
  reconcileMutuallyExclusiveNodeParams,
  reconcileMutuallyInclusiveNodeParams,
} from "utilities/url";

import { MacSettingsStatusQueryParam } from "./nodes";

export interface ISortOption {
  key: string;
  direction: string;
}

export interface INodesCountResponse {
  count: number;
}

export interface INodesCountQueryKey extends INodeCountLoadOptions {
  scope: "nodes_count";
}

export interface INodeCountLoadOptions {
  page?: number;
  perPage?: number;
  selectedLabels?: string[];
  globalFilter?: string;
  status?: NodeStatus;
  teamId?: number;
  policyId?: number;
  policyResponse?: string;
  macSettingsStatus?: MacSettingsStatusQueryParam;
  softwareId?: number;
  softwareTitleId?: number;
  softwareVersionId?: number;
  softwareStatus?: string;
  lowDiskSpaceNodes?: number;
  mdmId?: number;
  mdmEnrollmentStatus?: string;
  munkiIssueId?: number;
  osVersionId?: number;
  osName?: string;
  osVersion?: string;
  osSettings?: MdmProfileStatus;
  vulnerability?: string;
  diskEncryptionStatus?: DiskEncryptionStatus;
  bootstrapPackageStatus?: BootstrapPackageStatus;
}

export default {
  load: (
    options: INodeCountLoadOptions | undefined
  ): Promise<INodesCountResponse> => {
    const selectedLabels = options?.selectedLabels || [];
    const policyId = options?.policyId;
    const policyResponse = options?.policyResponse;
    const globalFilter = options?.globalFilter || "";
    const teamId = options?.teamId;
    const softwareId = options?.softwareId;
    const softwareTitleId = options?.softwareTitleId;
    const softwareVersionId = options?.softwareVersionId;
    const softwareStatus = options?.softwareStatus;
    const macSettingsStatus = options?.macSettingsStatus;
    const status = options?.status;
    const mdmId = options?.mdmId;
    const mdmEnrollmentStatus = options?.mdmEnrollmentStatus;
    const munkiIssueId = options?.munkiIssueId;
    const lowDiskSpaceNodes = options?.lowDiskSpaceNodes;
    const label = getLabelParam(selectedLabels);
    const osVersionId = options?.osVersionId;
    const osName = options?.osName;
    const osVersion = options?.osVersion;
    const osSettings = options?.osSettings;
    const vulnerability = options?.vulnerability;
    const diskEncryptionStatus = options?.diskEncryptionStatus;
    const bootstrapPackageStatus = options?.bootstrapPackageStatus;

    const queryParams = {
      query: globalFilter,
      ...reconcileMutuallyInclusiveNodeParams({
        teamId,
        macSettingsStatus,
        osSettings,
      }),
      // TODO: shouldn't macSettingsStatus be included in the mutually exclusive query params too?
      // If so, this todo applies in other places.
      ...reconcileMutuallyExclusiveNodeParams({
        teamId,
        label,
        policyId,
        policyResponse,
        mdmId,
        mdmEnrollmentStatus,
        munkiIssueId,
        softwareId,
        softwareTitleId,
        softwareStatus,
        softwareVersionId,
        lowDiskSpaceNodes,
        osName,
        osVersionId,
        osVersion,
        osSettings,
        vulnerability,
        diskEncryptionStatus,
        bootstrapPackageStatus,
      }),
      label_id: label,
      status,
    };

    const queryString = buildQueryStringFromParams(queryParams);
    const endpoint = endpoints.HOSTS_COUNT;
    const path = `${endpoint}?${queryString}`;
    return sendRequest("GET", path);
  },
};
