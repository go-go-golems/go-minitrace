import { useMemo } from "react";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import InputAdornment from "@mui/material/InputAdornment";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import SearchIcon from "@mui/icons-material/Search";
import QueryStatsIcon from "@mui/icons-material/QueryStats";
import type { SessionSummary } from "../../types";
import { ActiveBadge, FormatWallActive } from "../shared";

interface SessionBrowserProps {
  sessions: SessionSummary[];
  filterText: string;
  onFilterChange: (text: string) => void;
  onSelectSession: (id: string) => void;
  onQuerySession: (id: string) => void;
}

export function SessionBrowser({
  sessions,
  filterText,
  onFilterChange,
  onSelectSession,
  onQuerySession,
}: SessionBrowserProps) {
  const filtered = useMemo(() => {
    if (!filterText.trim()) return sessions;
    const q = filterText.toLowerCase();
    return sessions.filter(
      (s) =>
        s.title.toLowerCase().includes(q) ||
        s.id.toLowerCase().includes(q) ||
        s.operational_context.working_directory.toLowerCase().includes(q) ||
        s.environment.model.toLowerCase().includes(q)
    );
  }, [sessions, filterText]);

  const totalWall = filtered.reduce(
    (a, s) => a + s.timing.duration_seconds,
    0
  );
  const totalActive = filtered.reduce(
    (a, s) => a + s.timing.active_duration_seconds,
    0
  );
  const totalTurns = filtered.reduce(
    (a, s) => a + s.metrics.turn_count,
    0
  );

  return (
    <Box data-widget="session-browser" sx={{ height: "100%", display: "flex", flexDirection: "column" }}>
      {/* Filter bar */}
      <Box sx={{ p: 2, display: "flex", gap: 2, alignItems: "center" }}>
        <TextField
          size="small"
          placeholder="Filter sessions… (title, workdir, model, id)"
          value={filterText}
          onChange={(e) => onFilterChange(e.target.value)}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon sx={{ fontSize: 18, color: "text.secondary" }} />
                </InputAdornment>
              ),
            },
          }}
          sx={{ flex: 1, maxWidth: 500 }}
        />
        <Typography variant="caption" color="text.secondary">
          {filtered.length} sessions · {(totalWall / 3600).toFixed(0)}h wall /{" "}
          {(totalActive / 3600).toFixed(0)}h active · {totalTurns.toLocaleString()} turns
        </Typography>
      </Box>

      {/* Table */}
      <TableContainer component={Paper} sx={{ flex: 1, mx: 2, mb: 2 }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell sx={{ width: 130 }}>Date</TableCell>
              <TableCell sx={{ width: 100 }}>Duration</TableCell>
              <TableCell sx={{ width: 60 }}>Active</TableCell>
              <TableCell>Title</TableCell>
              <TableCell sx={{ width: 80 }} align="right">Turns</TableCell>
              <TableCell sx={{ width: 80 }} align="right">Tools</TableCell>
              <TableCell sx={{ width: 100 }}>Model</TableCell>
              <TableCell sx={{ width: 50 }} />
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((session) => {
              const activePct =
                (session.timing.active_duration_seconds /
                  Math.max(session.timing.duration_seconds, 1)) *
                100;
              const date = new Date(session.timing.started_at);
              return (
                <TableRow
                  key={session.id}
                  hover
                  onClick={() => onSelectSession(session.id)}
                  sx={{ cursor: "pointer", "&:hover": { bgcolor: "action.hover" } }}
                >
                  <TableCell>
                    <Typography variant="body2" sx={{ fontFamily: "monospace", fontSize: "0.75rem" }}>
                      {date.toLocaleDateString("en-CA")}{" "}
                      {date.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" })}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <FormatWallActive
                      wallSeconds={session.timing.duration_seconds}
                      activeSeconds={session.timing.active_duration_seconds}
                    />
                  </TableCell>
                  <TableCell>
                    <ActiveBadge activePct={activePct} />
                  </TableCell>
                  <TableCell>
                    <Typography
                      variant="body2"
                      sx={{
                        maxWidth: 500,
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                      }}
                    >
                      {session.title}
                    </Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
                      {session.operational_context.working_directory}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                      {session.metrics.turn_count.toLocaleString()}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
                      {session.metrics.tool_call_count.toLocaleString()}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption" color="text.secondary">
                      {session.environment.model}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Tooltip title="Open in Query Editor">
                      <IconButton
                        size="small"
                        onClick={(e) => {
                          e.stopPropagation();
                          onQuerySession(session.id);
                        }}
                      >
                        <QueryStatsIcon sx={{ fontSize: 16 }} />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
}
