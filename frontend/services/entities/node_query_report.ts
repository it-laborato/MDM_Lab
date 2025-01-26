import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IHQRResult {
  columns: Record<string, string>;
}
export interface IGetHQRResponse {
  query_id: number;
  node_id: number;
  node_name: string;
  last_fetched: string | null; // timestamp
  report_clipped: boolean;
  results: IHQRResult[];
}

export default {
  load: (nodeId: number, queryId: number): Promise<IGetHQRResponse> => {
    return sendRequest("GET", endpoints.HOST_QUERY_REPORT(nodeId, queryId));
  },
};
