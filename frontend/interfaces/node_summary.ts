import { ILabelSummary } from "./label";

export interface INodeSummaryPlatforms {
  platform: string;
  hosts_count: number;
}

export interface INodeSummary {
  all_linux_count: number;
  totals_hosts_count: number;
  platforms: INodeSummaryPlatforms[] | null;
  online_count: number;
  offline_count: number;
  mia_count: number; // DEPRECATED: to be removed in Mdmlab 5.0
  new_count: number;
  missing_30_days_count?: number; // premium feature
  low_disk_space_count?: number; // premium feature
  builtin_labels: ILabelSummary[];
}
