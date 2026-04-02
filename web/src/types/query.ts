/** Types for the query editor and saved queries */

export interface SavedQuery {
  name: string;
  folder: string;
  path: string;
  description: string;
  sql: string;
  readonly: boolean;
}

export interface QueryResult {
  columns: string[];
  rows: Record<string, unknown>[];
  duration_ms: number;
  row_count: number;
}

export interface QueryError {
  message: string;
  line?: number;
  column?: number;
}

export interface QueryExecution {
  sql: string;
  result: QueryResult | null;
  error: QueryError | null;
  executed_at: string;
}
