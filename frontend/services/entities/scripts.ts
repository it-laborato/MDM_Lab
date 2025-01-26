import { IScript, INodeScript } from "interfaces/script";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

/** Single script response from GET /script/:id */
export type IScriptResponse = IScript;

/** All scripts response from GET /scripts */
export interface IScriptsResponse {
  scripts: IScript[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IListScriptsApiParams {
  page?: number;
  per_page?: number;
  team_id?: number;
}

export interface IListScriptsQueryKey extends IListScriptsApiParams {
  scope: "scripts";
}

/**
 * Script Result response from GET /scripts/results/:id
 */
export interface IScriptResultResponse {
  nodename: string;
  node_id: number;
  execution_id: string;
  script_contents: string;
  script_id: number;
  exit_code: number | null;
  output: string;
  message: string;
  runtime: number;
  node_timeout: boolean;
  created_at: string;
}

/**
 * Request params for for GET /nodes/:id/scripts
 */
export interface INodeScriptsRequestParams {
  node_id: number;
  page?: number;
  per_page?: number;
}

export interface INodeScriptsQueryKey extends INodeScriptsRequestParams {
  scope: "node_scripts";
}

/**
 * Script response from GET /nodes/:id/scripts
 */
export interface INodeScriptsResponse {
  scripts: INodeScript[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

/**
 * Request body for POST /scripts/run
 *
 * https://github.com/mdmlabdm/mdmlab/blob/main/docs/Contributing/API-for-contributors.md#run-script-asynchronously
 */
export interface IScriptRunRequest {
  node_id: number;
  script_id: number; // script_id is not required by the API currently, but we require it here to ensure it is always provided
  // script_contents: string; // script_contents is only supported for the CLI currently
}

/**
 * Response body for POST /scripts/run
 *
 * https://github.com/mdmlabdm/mdmlab/blob/main/docs/Contributing/API-for-contributors.md#run-script-asynchronously
 */
export interface IScriptRunResponse {
  node_id: number;
  execution_id: string;
}

export default {
  getNodeScripts({ node_id, page, per_page }: INodeScriptsRequestParams) {
    const { HOST_SCRIPTS } = endpoints;
    const path = `${HOST_SCRIPTS(node_id)}?${buildQueryStringFromParams({
      page,
      per_page,
    })}`;

    return sendRequest("GET", path);
  },

  getScripts(params: IListScriptsApiParams): Promise<IScriptsResponse> {
    const { SCRIPTS } = endpoints;
    const path = `${SCRIPTS}?${buildQueryStringFromParams({ ...params })}`;

    return sendRequest("GET", path);
  },

  getScript(id: number) {
    const { SCRIPT } = endpoints;
    return sendRequest("GET", SCRIPT(id));
  },

  uploadScript(file: File, teamId?: number) {
    const { SCRIPTS } = endpoints;

    const formData = new FormData();
    formData.append("script", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", SCRIPTS, formData);
  },

  downloadScript(id: number) {
    const { SCRIPT } = endpoints;
    const path = `${SCRIPT(id)}?${buildQueryStringFromParams({
      alt: "media",
    })}`;
    return sendRequest("GET", path);
  },

  deleteScript(id: number) {
    const { SCRIPT } = endpoints;
    return sendRequest("DELETE", SCRIPT(id));
  },

  getScriptResult(executionId: string) {
    const { SCRIPT_RESULT } = endpoints;
    return sendRequest("GET", SCRIPT_RESULT(executionId));
  },

  runScript(request: IScriptRunRequest): Promise<IScriptRunResponse> {
    const { SCRIPT_RUN } = endpoints;
    return sendRequest("POST", SCRIPT_RUN, request);
  },
};
