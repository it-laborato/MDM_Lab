import { HOST_LINUX_PLATFORMS } from "./platform";

export interface IScript {
  id: number;
  team_id: number | null;
  name: string;
  created_at: string;
  updated_at: string;
}

export const isScriptSupportedPlatform = (nodePlatform: string) =>
  ["darwin", "windows", ...HOST_LINUX_PLATFORMS].includes(nodePlatform); // excludes chrome, ios, ipados see also https://github.com/mdmlabdm/mdmlab/blob/5a21e2cfb029053ddad0508869eb9f1f23997bf2/server/mdmlab/nodes.go#L775

export type IScriptExecutionStatus = "ran" | "pending" | "error";

export interface ILastExecution {
  execution_id: string;
  executed_at: string;
  status: IScriptExecutionStatus;
}

export interface INodeScript {
  script_id: number;
  name: string;
  last_execution: ILastExecution | null;
}
