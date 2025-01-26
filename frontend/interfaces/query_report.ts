export interface IQueryReportResultRow {
  node_id: number;
  node_name: string;
  last_fetched: string;
  columns: any; // {col:val, ...}
}

// Query report
export interface IQueryReport {
  query_id: number;
  results: IQueryReportResultRow[];
  report_clipped: boolean;
}
