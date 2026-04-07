import { useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TableSortLabel from "@mui/material/TableSortLabel";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Button from "@mui/material/Button";
import Popover from "@mui/material/Popover";
import DownloadIcon from "@mui/icons-material/Download";
import type { QueryResult } from "../../types";

interface ResultsTableProps {
  result: QueryResult;
  onClickSessionId?: (id: string) => void;
}

function isSessionId(value: unknown): value is string {
  return (
    typeof value === "string" &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(value)
  );
}

type SortDir = "asc" | "desc";

function exportCsv(result: QueryResult) {
  const header = result.columns.join(",");
  const rows = result.rows.map((row) =>
    result.columns
      .map((col) => {
        const v = String(row[col] ?? "");
        return v.includes(",") || v.includes('"') || v.includes("\n")
          ? `"${v.replace(/"/g, '""')}"`
          : v;
      })
      .join(",")
  );
  const csv = [header, ...rows].join("\n");
  const blob = new Blob([csv], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "query-results.csv";
  a.click();
  URL.revokeObjectURL(url);
}

function exportJson(result: QueryResult) {
  const blob = new Blob([JSON.stringify(result.rows, null, 2)], {
    type: "application/json",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "query-results.json";
  a.click();
  URL.revokeObjectURL(url);
}

export function ResultsTable({ result, onClickSessionId }: ResultsTableProps) {
  const [sortCol, setSortCol] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [expandedCell, setExpandedCell] = useState<{
    anchor: HTMLElement;
    content: string;
  } | null>(null);

  const handleSort = (col: string) => {
    if (sortCol === col) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortCol(col);
      setSortDir("asc");
    }
  };

  const sortedRows = useMemo(() => [...result.rows].sort((a, b) => {
    if (!sortCol) return 0;
    const va = a[sortCol];
    const vb = b[sortCol];
    if (va == null && vb == null) return 0;
    if (va == null) return 1;
    if (vb == null) return -1;
    const cmp =
      typeof va === "number" && typeof vb === "number"
        ? va - vb
        : String(va).localeCompare(String(vb));
    return sortDir === "asc" ? cmp : -cmp;
  }), [result.rows, sortCol, sortDir]);

  return (
    <Box data-part="results-table">
      {/* Header bar */}
      <Stack direction="row" spacing={1.5} alignItems="center" sx={{ px: 1, py: 0.75 }}>
        <Chip
          label={`${result.row_count} rows`}
          size="small"
          variant="outlined"
          sx={{ fontFamily: "monospace" }}
        />
        <Chip
          label={`${result.columns.length} cols`}
          size="small"
          variant="outlined"
          sx={{ fontFamily: "monospace" }}
        />
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ fontFamily: "monospace" }}
        >
          {result.duration_ms}ms
        </Typography>
        <Box sx={{ flex: 1 }} />
        <Button
          size="small"
          startIcon={<DownloadIcon />}
          onClick={() => exportCsv(result)}
          sx={{ fontSize: "0.75rem" }}
        >
          CSV
        </Button>
        <Button
          size="small"
          startIcon={<DownloadIcon />}
          onClick={() => exportJson(result)}
          sx={{ fontSize: "0.75rem" }}
        >
          JSON
        </Button>
      </Stack>

      {/* Table */}
      <TableContainer
        component={Paper}
        sx={{ maxHeight: "calc(100vh - 360px)", overflow: "auto" }}
      >
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell
                sx={{
                  width: 40,
                  bgcolor: "background.paper",
                  color: "text.secondary",
                  fontFamily: "monospace",
                  fontSize: "0.625rem",
                }}
              >
                #
              </TableCell>
              {result.columns.map((col) => (
                <TableCell key={col} sx={{ bgcolor: "background.paper" }}>
                  <TableSortLabel
                    active={sortCol === col}
                    direction={sortCol === col ? sortDir : "asc"}
                    onClick={() => handleSort(col)}
                  >
                    {col}
                  </TableSortLabel>
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {sortedRows.map((row, i) => (
              <TableRow key={i} hover>
                <TableCell
                  sx={{
                    fontFamily: "monospace",
                    fontSize: "0.625rem",
                    color: "text.secondary",
                    width: 40,
                  }}
                >
                  {i + 1}
                </TableCell>
                {result.columns.map((col) => {
                  const val = row[col];

                  // Session ID → clickable link
                  if (isSessionId(val) && onClickSessionId) {
                    return (
                      <TableCell key={col}>
                        <Link
                          component="button"
                          onClick={() => onClickSessionId(val)}
                          sx={{
                            fontFamily: "monospace",
                            fontSize: "0.75rem",
                            color: "primary.light",
                            textDecoration: "none",
                            "&:hover": { textDecoration: "underline" },
                          }}
                        >
                          {val.slice(0, 8)}…
                        </Link>
                      </TableCell>
                    );
                  }

                  // Long text → truncated with click-to-expand
                  const display = String(val ?? "");
                  const isLong = display.length > 100;

                  return (
                    <TableCell key={col}>
                      <Typography
                        variant="body2"
                        onClick={
                          isLong
                            ? (e) =>
                                setExpandedCell({
                                  anchor: e.currentTarget as HTMLElement,
                                  content: display,
                                })
                            : undefined
                        }
                        sx={{
                          fontFamily: "monospace",
                          fontSize: "0.75rem",
                          maxWidth: 400,
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                          cursor: isLong ? "pointer" : "default",
                          "&:hover": isLong
                            ? { color: "primary.light", textDecoration: "underline" }
                            : {},
                        }}
                      >
                        {isLong ? display.slice(0, 100) + "…" : display}
                      </Typography>
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Cell expand popover */}
      <Popover
        open={!!expandedCell}
        anchorEl={expandedCell?.anchor}
        onClose={() => setExpandedCell(null)}
        anchorOrigin={{ vertical: "bottom", horizontal: "left" }}
        transformOrigin={{ vertical: "top", horizontal: "left" }}
      >
        <Box
          sx={{
            p: 2,
            maxWidth: 600,
            maxHeight: 400,
            overflow: "auto",
            bgcolor: "background.default",
          }}
        >
          <Typography
            component="pre"
            variant="body2"
            sx={{
              fontFamily: "monospace",
              fontSize: "0.75rem",
              whiteSpace: "pre-wrap",
              wordBreak: "break-all",
              m: 0,
            }}
          >
            {expandedCell?.content}
          </Typography>
        </Box>
      </Popover>
    </Box>
  );
}
