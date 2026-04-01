import Box from "@mui/material/Box";
import List from "@mui/material/List";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import ListSubheader from "@mui/material/ListSubheader";
import Typography from "@mui/material/Typography";
import Tooltip from "@mui/material/Tooltip";
import FolderIcon from "@mui/icons-material/Folder";
import DescriptionIcon from "@mui/icons-material/Description";
import StarIcon from "@mui/icons-material/Star";
import type { SavedQuery } from "../../types";

interface QuerySidebarProps {
  presets: SavedQuery[];
  savedQueries: SavedQuery[];
  onSelect: (sql: string) => void;
}

function groupByFolder(queries: SavedQuery[]) {
  const groups: Record<string, SavedQuery[]> = {};
  for (const q of queries) {
    const folder = q.folder || "ungrouped";
    if (!groups[folder]) groups[folder] = [];
    groups[folder].push(q);
  }
  return groups;
}

export function QuerySidebar({
  presets,
  savedQueries,
  onSelect,
}: QuerySidebarProps) {
  const presetGroups = groupByFolder(presets);
  const savedGroups = groupByFolder(savedQueries);

  return (
    <Box
      data-part="query-sidebar"
      sx={{
        width: 240,
        borderRight: "1px solid",
        borderColor: "divider",
        overflow: "auto",
        bgcolor: "background.default",
      }}
    >
      <Typography variant="overline" sx={{ px: 2, pt: 2, display: "block" }}>
        Presets
      </Typography>
      {Object.entries(presetGroups).map(([folder, queries]) => (
        <List
          key={folder}
          dense
          subheader={
            <ListSubheader
              sx={{
                bgcolor: "transparent",
                lineHeight: "28px",
                display: "flex",
                alignItems: "center",
                gap: 0.5,
              }}
            >
              <FolderIcon sx={{ fontSize: 14 }} />
              {folder}
            </ListSubheader>
          }
        >
          {queries.map((q) => (
            <Tooltip key={q.path} title={q.description} placement="right" arrow>
              <ListItemButton
                onClick={() => onSelect(q.sql)}
                sx={{ py: 0.25, pl: 4 }}
              >
                <ListItemIcon sx={{ minWidth: 28 }}>
                  <DescriptionIcon sx={{ fontSize: 14, color: "text.secondary" }} />
                </ListItemIcon>
                <ListItemText
                  primary={q.name}
                  primaryTypographyProps={{
                    variant: "body2",
                    sx: { fontFamily: "monospace", fontSize: "0.75rem" },
                  }}
                />
              </ListItemButton>
            </Tooltip>
          ))}
        </List>
      ))}

      <Typography variant="overline" sx={{ px: 2, pt: 2, display: "block" }}>
        Saved
      </Typography>
      {Object.entries(savedGroups).map(([folder, queries]) => (
        <List
          key={folder}
          dense
          subheader={
            <ListSubheader
              sx={{
                bgcolor: "transparent",
                lineHeight: "28px",
                display: "flex",
                alignItems: "center",
                gap: 0.5,
              }}
            >
              <FolderIcon sx={{ fontSize: 14 }} />
              {folder}
            </ListSubheader>
          }
        >
          {queries.map((q) => (
            <Tooltip key={q.path} title={q.description} placement="right" arrow>
              <ListItemButton
                onClick={() => onSelect(q.sql)}
                sx={{ py: 0.25, pl: 4 }}
              >
                <ListItemIcon sx={{ minWidth: 28 }}>
                  <StarIcon sx={{ fontSize: 14, color: "primary.main" }} />
                </ListItemIcon>
                <ListItemText
                  primary={q.name}
                  primaryTypographyProps={{
                    variant: "body2",
                    sx: { fontFamily: "monospace", fontSize: "0.75rem" },
                  }}
                />
              </ListItemButton>
            </Tooltip>
          ))}
        </List>
      ))}
    </Box>
  );
}
