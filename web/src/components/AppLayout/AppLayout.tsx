import { Outlet, useNavigate, useLocation } from "react-router";
import Box from "@mui/material/Box";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import DataObjectIcon from "@mui/icons-material/DataObject";
import StorageIcon from "@mui/icons-material/Storage";
import QueryStatsIcon from "@mui/icons-material/QueryStats";
import { useGetSessionsQuery } from "../../api/minitrace";

export function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: sessions = [] } = useGetSessionsQuery();

  const isSessionsActive =
    location.pathname === "/sessions" ||
    location.pathname.startsWith("/sessions/");
  const isQueryActive = location.pathname === "/query";

  return (
    <Box sx={{ height: "100vh", display: "flex", flexDirection: "column" }}>
      <AppBar
        position="static"
        elevation={0}
        sx={{
          bgcolor: "background.paper",
          borderBottom: "1px solid",
          borderColor: "divider",
        }}
      >
        <Toolbar variant="dense" sx={{ gap: 1 }}>
          <DataObjectIcon sx={{ color: "primary.main" }} />
          <Typography
            variant="h4"
            sx={{ mr: 3, color: "text.primary", cursor: "pointer" }}
            onClick={() => navigate("/sessions")}
          >
            minitrace
          </Typography>

          <Button
            startIcon={<StorageIcon />}
            onClick={() => navigate("/sessions")}
            variant={isSessionsActive ? "contained" : "text"}
            size="small"
          >
            Sessions
          </Button>
          <Button
            startIcon={<QueryStatsIcon />}
            onClick={() => navigate("/query")}
            variant={isQueryActive ? "contained" : "text"}
            size="small"
          >
            Query
          </Button>

          <Box sx={{ flex: 1 }} />
          <Chip
            label={`${sessions.length} sessions loaded`}
            size="small"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
        </Toolbar>
      </AppBar>

      <Box sx={{ flex: 1, overflow: "hidden" }}>
        <Outlet />
      </Box>
    </Box>
  );
}
