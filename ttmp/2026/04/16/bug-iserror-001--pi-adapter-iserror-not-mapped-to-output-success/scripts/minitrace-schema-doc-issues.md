# Issues Found in go-minitrace minitrace-schema & writing-duckdb-queries Docs

## 1. Type Mismatch: `tool_name` Described as `string` but DuckDB Loads as `JSON`

**Where:** `minitrace-schema` → Tool Calls → `tool_name` says `string`
**Reality:** In DuckDB, `UNNEST(tool_calls) AS t(tc)` produces elements where
`json_extract(tc, '$.tool_name')` returns a JSON-typed value, not VARCHAR.

```sql
-- This FAILS with "Conversion Error: Malformed JSON at byte 0... Input: write"
WHERE tc.tool_name IN ('write', 'edit', 'read')

-- You must use this pattern instead:
WHERE REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') IN ('write', 'edit', 'read')
```

**Fix:** The doc should either:
- Note that DuckDB loads all unnested JSON element fields as JSON type, requiring CAST
- Or document the canonical pattern: `REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '')` for string comparisons

## 2. `writing-duckdb-queries` Shows Dot-Notation for Unnested Elements — It Doesn't Work

**Where:** `writing-duckdb-queries` → Extracting fields from array elements says:
> `json_extract(tc, '$.tool_name')` — tool name

This works, but the doc also implies you can use dot notation on the unnested element.
In practice, `tc.tool_name`, `tc.output.success`, `tc.input.file_path` all return
JSON-typed values that cannot be used directly in comparisons or WHERE clauses.

**Fix:** Explicitly show the full pattern:
```sql
-- WRONG (returns JSON type, breaks comparisons)
WHERE tc.tool_name = 'write'

-- RIGHT
WHERE REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') = 'write'
```

## 3. No Documentation of Boolean Field Handling

**Where:** `tool_calls.output.success` is documented as `bool`
**Reality:** `json_extract(tc, '$.output.success')` returns a JSON boolean.
In DuckDB, this works for `= true` / `= false` comparisons, but you CANNOT use it
directly in WHERE with `IS TRUE` or other SQL boolean operators without casting.

**Fix:** Show an example:
```sql
-- This works in DuckDB
WHERE json_extract(tc, '$.output.success') = false

-- This does NOT work
WHERE tc.output.success IS FALSE
```

## 4. Missing: How to Handle `null` JSON Values

**Where:** Many fields are nullable (`string?`, `int?`)
**Reality:** When a field is null in the JSON, `json_extract` returns DuckDB NULL.
But `CAST(NULL AS VARCHAR)` gives SQL NULL, not the string `"null"`.
The queries in the doc don't show how to handle this.

**Fix:** Show COALESCE patterns:
```sql
COALESCE(CAST(json_extract(tc, '$.input.file_path') AS VARCHAR), '(no path)')
```

## 5. Missing: Schema Discovery Commands

**Where:** Neither `minitrace-schema` nor `writing-duckdb-queries` tells you how to
discover the actual schema of your loaded data.

**Fix:** Add a section:
```sql
-- Describe the sessions_base table
DESCRIBE sessions_base;

-- Inspect structure of a single tool_call element
SELECT json_structure(tool_calls[1]) FROM sessions_base LIMIT 1;

-- Check actual types of unnested fields
SELECT DISTINCT typeof(json_extract(tc, '$.tool_name')) AS tn_type
FROM sessions_base, UNNEST(tool_calls) AS t(tc);
```

## 6. `input.arguments` Structure Not Documented

**Where:** `minitrace-schema` → Tool Call Input shows:
```
arguments: any? — Full arguments blob
```
**Reality:** For pi adapter, `arguments` has a nested structure like
`arguments.path` visible in `json_structure` output, but the doc doesn't
explain this nested structure or how to query it.

**Fix:** Show example structures for each adapter (pi, claude-code, codex).

## 7. Missing: `emitting_turn_index` Not in `writing-duckdb-queries` Examples

**Where:** The `writing-duckdb-queries` doc doesn't mention `emitting_turn_index`
anywhere, but it's the primary way to order tool calls chronologically.

**Fix:** Add to the "Extracting fields from array elements" section:
```sql
CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn
```

## 8. UNNEST Syntax Confusion

**Where:** `writing-duckdb-queries` shows:
```sql
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
```
But then shows field access like `json_extract(tc, '$.tool_name')` without
explaining that `tc` is a JSON value, not a row with typed columns.

**Fix:** Add a clear explanation:
> After `UNNEST(tool_calls) AS t(tc)`, the variable `tc` holds a single JSON element.
> You must use `json_extract(tc, '$.field')` to access fields — you cannot use
> dot notation like `tc.field` for comparisons.

## 9. Missing: How to Query `output.result` Content

**Where:** The doc doesn't explain that `output.result` is a 10KB-truncated string
that often contains multi-line output with escape sequences.

**Fix:** Show LIKE-based content search:
```sql
-- Search for specific result content
WHERE CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%File not found%'

-- Get just the first N chars
LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 200)
```

## 10. Missing: Timestamp Handling in Tool Calls

**Where:** `timestamp` in tool_calls is documented but no examples show
how to use it for time-based filtering or ordering.

**Fix:**
```sql
-- Order by tool call timestamp
ORDER BY json_extract(tc, '$.timestamp')

-- Filter by time window
WHERE json_extract(tc, '$.timestamp') >= '2026-04-16T01:48:00'
```

## Summary of Fixes Needed

| Doc | Issue | Severity |
|-----|-------|----------|
| minitrace-schema | JSON type vs string for tool_name | High — breaks all queries |
| writing-duckdb-queries | No dot-notation warning for UNNEST | High — causes confusion |
| writing-duckdb-queries | No schema discovery section | Medium — users can't debug |
| writing-duckdb-queries | Missing emitting_turn_index examples | Medium — needed for ordering |
| minitrace-schema | arguments nested structure not shown | Low — niche use case |
| both | Boolean handling in WHERE clauses | Medium — causes silent wrong results |
