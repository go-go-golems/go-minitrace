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
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import type { QueryCommand, SavedQuery } from "../../types";

interface QuerySidebarProps {
  presets: SavedQuery[];
  savedQueries: SavedQuery[];
  commands: QueryCommand[];
  onSelect: (query: SavedQuery, kind: "preset" | "saved") => void;
  onSelectCommand?: (command: QueryCommand) => void;
}

function groupByFolder<T extends { folder: string }>(queries: T[]) {
  const groups: Record<string, T[]> = {};
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
  commands,
  onSelect,
  onSelectCommand,
}: QuerySidebarProps) {
  const commandGroups = groupByFolder(commands);
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
        Commands
      </Typography>
      {Object.entries(commandGroups).map(([folder, groupedCommands]) => (
        <List
          key={`commands-${folder}`}
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
          {groupedCommands.map((command) => (
            <Tooltip
              key={command.path}
              title={command.shortDescription || command.longDescription}
              placement="right"
              arrow
            >
              <ListItemButton
                onClick={() => onSelectCommand?.(command)}
                sx={{ py: 0.25, pl: 4 }}
              >
                <ListItemIcon sx={{ minWidth: 28 }}>
                  <PlayArrowIcon sx={{ fontSize: 14, color: "secondary.main" }} />
                </ListItemIcon>
                <ListItemText
                  primary={command.name}
                  secondary={command.kind === "alias" ? `alias for ${command.aliasFor}` : undefined}
                  primaryTypographyProps={{
                    variant: "body2",
                    sx: { fontFamily: "monospace", fontSize: "0.75rem" },
                  }}
                  secondaryTypographyProps={{
                    variant: "caption",
                    sx: { fontFamily: "monospace", fontSize: "0.65rem" },
                  }}
                />
              </ListItemButton>
            </Tooltip>
          ))}
        </List>
      ))}

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
                onClick={() => onSelect(q, "preset")}
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
                onClick={() => onSelect(q, "saved")}
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
