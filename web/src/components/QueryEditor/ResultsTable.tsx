import Box from "@mui/material/Box";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Chip from "@mui/material/Chip";
import Link from "@mui/material/Link";
import type { QueryResult } from "../../types";

interface ResultsTableProps {
  result: QueryResult;
  onClickSessionId?: (id: string) => void;
}

function isSessionId(value: unknown): value is string {
  return (
    typeof value === "string" &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(
      value
    )
  );
}

export function ResultsTable({ result, onClickSessionId }: ResultsTableProps) {
  return (
    <Box data-part="results-table">
      <Box sx={{ display: "flex", alignItems: "center", gap: 1.5, px: 1, py: 0.5 }}>
        <Chip
          label={`${result.row_count} rows`}
          size="small"
          variant="outlined"
          sx={{ fontFamily: "monospace" }}
        />
        <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
          {result.duration_ms}ms
        </Typography>
      </Box>
      <TableContainer component={Paper} sx={{ maxHeight: 500 }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              {result.columns.map((col) => (
                <TableCell key={col}>{col}</TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {result.rows.map((row, i) => (
              <TableRow key={i} hover>
                {result.columns.map((col) => {
                  const val = row[col];
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
                  const display =
                    typeof val === "string" && val.length > 120
                      ? val.slice(0, 120) + "…"
                      : String(val ?? "");
                  return (
                    <TableCell key={col}>
                      <Typography
                        variant="body2"
                        sx={{ fontFamily: "monospace", fontSize: "0.75rem" }}
                      >
                        {display}
                      </Typography>
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
