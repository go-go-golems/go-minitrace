export interface NativeEvidenceBuilder {
 file(path: string): NativeEvidenceBuilder;
 maxRecordBytes(n: number): NativeEvidenceBuilder;
 maxIndexBytes(n: number): NativeEvidenceBuilder;
 forEach(callback: (record: NativeRecord) => boolean | void): NativeScanStats;
 inspect(): NativeInspection;
}
export interface NativeCoordinate { line: number; byteOffset: number; rawBytes: number; }
export interface NativeDiagnostic { code: string; message: string; reference?: string; }
export interface NativeRecord { source: NativeCoordinate; value?: Record<string, unknown>; rawJSON?: string; diagnostic?: NativeDiagnostic; }
export interface NativeScanStats { sha256: string; hashScope: "whole_source" | "consumed_prefix"; bytes: number; lines: number; malformed: number; stopReason: "eof" | "callback" | "canceled" | "error"; }
export interface NativeBlob { source: NativeCoordinate; value: Record<string, unknown>; rawJSON: string; }
export interface NativeRequest { id: string; querySource: string; source: NativeCoordinate; shapeHash: string; refs: string[]; status: "complete" | "missing_shape" | "missing_base" | "missing_blob" | "unsupported"; diagnostics: NativeDiagnostic[]; }
export interface NativeMessageCoordinate { source: NativeCoordinate; uuid: string; parentUuid?: string; type: string; }
export interface NativeCompaction { boundary: NativeMessageCoordinate; logicalParentUuid?: string; metadata?: Record<string, unknown>; summaries: NativeMessageCoordinate[]; beforeMessage?: NativeMessageCoordinate; firstFollowingMessage?: NativeMessageCoordinate; firstFollowingAssistant?: NativeMessageCoordinate; }
export interface NativeInspection { scan: NativeScanStats; requests: NativeRequest[]; blobs: Record<string, NativeBlob>; shapes: Record<string, NativeBlob>; compactions: NativeCompaction[]; diagnostics: NativeDiagnostic[]; indexBytes: number; limitations: string[]; }

export interface ImportBuilder {
  content(content: string): ImportBuilder;
  file(path: string): ImportBuilder;
  name(name: string): ImportBuilder;
  sourcePath(path: string): ImportBuilder;
  autoDetect(): ImportBuilder;
  format(format: string): ImportBuilder;
  strict(enabled?: boolean): ImportBuilder;
  into(rootDir: string): ImportBuilder;
  sessionId(id: string): ImportBuilder;
  overwrite(enabled?: boolean): ImportBuilder;
  detect(): Record<string, unknown>;
  convert(): ImportBuilder;
  converted(): Record<string, unknown>;
  preview(): Record<string, unknown>;
  diagnostics(): Array<Record<string, unknown>>;
  save(): Record<string, unknown>;
}

export type Scalar = string | number | boolean | null;
export type ResultRow = Record<string, Scalar>;

declare const model: unique symbol;
export interface SourceSet { readonly [model]: "sources"; }
export interface Table { readonly [model]: "table"; }
export interface OrderKey { readonly [model]: "order"; }

export interface SourceSetBuilder {
  files(paths: readonly string[]): SourceSetBuilder;
  directory(path: string): SourceSetBuilder;
  glob(pattern: string): SourceSetBuilder;
  content(text: string, options?: { name?: string }): SourceSetBuilder;
  runtimeArchives(): SourceSetBuilder;
  build(): SourceSet;
}

export interface ImportOptions { autoConvert?: boolean; strict?: boolean; }
export interface CacheOptions {
  mode?: "none" | "memory" | "disk" | "auto";
  dir?: string;
  maxBytes?: number;
  forceRebuild?: boolean;
}
export interface LimitsOptions {
  maxRows?: number;
  maxColumns?: number;
  maxCellBytes?: number;
  timeoutMs?: number;
  strict?: boolean;
  requireOrderBy?: boolean;
}
export interface DBBuilder {
  sources(selection: SourceSet): DBBuilder;
  import(options: ImportOptions): DBBuilder;
  cache(options: CacheOptions): DBBuilder;
  limits(options: LimitsOptions): DBBuilder;
  cacheKey(): Record<string, unknown>;
  build(): DBHandle;
}
export interface DBHandle {
  table(name: string): Table;
  execute<Row extends ResultRow>(plan: QueryPlan<Row>, options?: { params?: Record<string, Scalar> }): QueryResult<Row>;
  schema(): Record<string, unknown>;
  tables(): Array<Record<string, unknown>>;
  stats(): Record<string, unknown>;
  sources(): Array<{ kind: string; name?: string; path?: string }>;
  diagnostics(): Array<Record<string, unknown>>;
  cacheInfo(): Record<string, unknown>;
  close(): void;
}
export interface CellDiagnostic {
  rowIndex: number;
  columnIndex: number;
  column: string;
  originalBytes: number;
  emittedBytes: number;
  truncated: boolean;
  invalidUtf8?: boolean;
}
export interface QueryResult<Row extends ResultRow = ResultRow> {
  columns: string[];
  rows: Row[];
  count: number;
  complete: boolean;
  truncated?: boolean;
  rowsTruncated?: boolean;
  cellDiagnostics?: CellDiagnostic[];
  error?: string;
}

export interface Expression<Value extends Scalar = Scalar> {
  readonly [model]: Value;
  eq(value: Scalar | Expression): Expression<boolean>;
  ne(value: Scalar | Expression): Expression<boolean>;
  gt(value: Scalar | Expression): Expression<boolean>;
  gte(value: Scalar | Expression): Expression<boolean>;
  lt(value: Scalar | Expression): Expression<boolean>;
  lte(value: Scalar | Expression): Expression<boolean>;
  and(value: boolean | Expression<boolean>): Expression<boolean>;
  or(value: boolean | Expression<boolean>): Expression<boolean>;
  not(): Expression<boolean>;
  contains(value: string | Expression<string>): Expression<boolean>;
  isNull(): Expression<boolean>;
  isNotNull(): Expression<boolean>;
  asc(): OrderKey;
  desc(): OrderKey;
}
export interface ParameterUI { widget?: string; label?: string; help?: string; }
export type ScalarKind = "string" | "integer" | "number" | "boolean";
export interface ParameterDescriptor {
  name: string;
  type: ScalarKind;
  hasDefault: boolean;
  default?: Scalar;
  hasRange: boolean;
  min: number;
  max: number;
  ui: ParameterUI;
}
export interface Parameter<Value extends Scalar> extends Expression<Value> {
  default(value: Value): this;
  ui(options: ParameterUI): this;
}
export interface IntegerParameter extends Parameter<number> {
  range(min: number, max: number): this;
}
export interface ColumnType<Value extends Scalar> {
  readonly [model]: Value;
  nullable(): ColumnType<Value | null>;
}
type ColumnTypes = Record<string, ColumnType<Scalar>>;
type DeclaredRow<Columns> = { [Name in keyof Columns]: Columns[Name] extends ColumnType<infer V> ? V : never };
type FieldMap = Record<string, Scalar | Expression>;
type ProjectedRow<Fields> = { [Name in keyof Fields]: Fields[Name] extends Expression<infer V> ? V : Fields[Name] };
type SymbolicRow<Row> = { [Name in keyof Row]: Row[Name] extends Scalar ? Expression<Row[Name]> : never };

export interface QueryPlan<Row extends ResultRow = ResultRow> {
  readonly [model]: Row;
  explain(): { sql: string; mapRows: boolean };
  parameters(): ParameterDescriptor[];
}
export interface TerminalQuery<Row extends ResultRow = ResultRow> { build(): QueryPlan<Row>; }
export interface SQLSource extends TerminalQuery {
  columns<Columns extends ColumnTypes>(columns: Columns): Query<DeclaredRow<Columns>>;
}
type FragmentMethods = Record<string, (query: Query, ...args: any[]) => Query>;
type BoundMethods<Methods> = { [Name in keyof Methods]:
  Methods[Name] extends (query: Query, ...args: infer Args) => infer Result ? (...args: Args) => Result : never };
export interface Fragment<Methods extends FragmentMethods> { readonly [model]: Methods; }
export interface Query<Row extends ResultRow = ResultRow> extends TerminalQuery<Row> {
  where(predicate: (row: SymbolicRow<Row>) => Expression<boolean>): this;
  select<Fields extends FieldMap>(projection: (row: SymbolicRow<Row>) => Fields): Query<ProjectedRow<Fields> & ResultRow>;
  orderBy(order: (row: SymbolicRow<Row>) => OrderKey | readonly OrderKey[]): this;
  use<Q extends Query>(transform: (query: this) => Q): Q;
  count(): Expression<number>;
  extend<Methods extends FragmentMethods>(fragment: Fragment<Methods>): this & BoundMethods<Methods>;
  mapRows<Columns extends ColumnTypes>(callback: (row: Row) => DeclaredRow<Columns>,
    options: { columns: Columns }): TerminalQuery<DeclaredRow<Columns>>;
}

export const param: {
  string(name: string): Parameter<string>;
  boolean(name: string): Parameter<boolean>;
  integer(name: string): IntegerParameter;
  number(name: string): Parameter<number>;
};
export const type: {
  string(): ColumnType<string>;
  boolean(): ColumnType<boolean>;
  integer(): ColumnType<number>;
  number(): ColumnType<number>;
};
export const runtime: { tableName: string; dbPath: string; archiveGlob: string[]; persistLoaded: boolean; commandName: string };
export function db(): DBBuilder;
export function sources(): SourceSetBuilder;
export function importer(): ImportBuilder;
export function native(): NativeEvidenceBuilder;
export function from(table: Table): Query;
export function sql(strings: TemplateStringsArray, ...values: Array<Scalar | Parameter<Scalar>>): SQLSource;
export function fragment<Methods extends FragmentMethods>(config: {
  name: string; target: "query"; requires?: readonly string[]; methods: Methods;
}): Fragment<Methods>;
export function utf8ByteLength(text: string): number;
export function corpusManifest(paths: readonly string[]): Record<string, unknown>;
