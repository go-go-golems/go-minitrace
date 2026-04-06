import { memo } from "react";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import PersonIcon from "@mui/icons-material/Person";
import SmartToyIcon from "@mui/icons-material/SmartToy";
import CommitIcon from "@mui/icons-material/Commit";
import ConfirmationNumberIcon from "@mui/icons-material/ConfirmationNumber";
import EditNoteIcon from "@mui/icons-material/EditNote";
import type { SessionBlock } from "../../types";

interface BlockHeaderProps {
  block: SessionBlock;
  isExpanded: boolean;
  onToggle: () => void;
}

function BlockHeaderImpl({ block, isExpanded, onToggle }: BlockHeaderProps) {
  const date = new Date(block.user_ts);
  const timeStr = date.toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
  });
  const dateStr = date.toLocaleDateString("en-CA");

  return (
    <Box
      onClick={onToggle}
      sx={{
        display: "flex",
        alignItems: "center",
        gap: 1.5,
        px: 2,
        py: 1,
        cursor: "pointer",
        bgcolor: isExpanded ? "rgba(245,166,35,0.04)" : "transparent",
        "&:hover": { bgcolor: "action.hover" },
      }}
    >
      <IconButton size="small" sx={{ p: 0 }}>
        <ExpandMoreIcon
          sx={{
            fontSize: 18,
            transform: isExpanded ? "rotate(0deg)" : "rotate(-90deg)",
            transition: "transform 0.15s",
          }}
        />
      </IconButton>

      <Chip
        label={`#${block.block_num}`}
        size="small"
        variant="outlined"
        sx={{ fontFamily: "monospace", fontWeight: 700, minWidth: 40 }}
      />

      <Typography variant="caption" sx={{ fontFamily: "monospace", color: "text.secondary" }}>
        {dateStr} {timeStr}
      </Typography>

      {block.gap_minutes != null && block.gap_minutes > 30 && (
        <Chip
          label={`${block.gap_minutes >= 60 ? (block.gap_minutes / 60).toFixed(1) + "h" : Math.round(block.gap_minutes) + "m"} gap`}
          size="small"
          color="warning"
          variant="outlined"
          sx={{ fontFamily: "monospace", fontSize: "0.6875rem" }}
        />
      )}

      <Typography
        variant="body2"
        sx={{
          flex: 1,
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
          fontWeight: isExpanded ? 600 : 400,
        }}
      >
        {block.user_content}
      </Typography>

      <Stack direction="row" spacing={1} alignItems="center">
        {block.artifacts.commits.length > 0 && (
          <Chip
            icon={<CommitIcon sx={{ fontSize: 14 }} />}
            label={block.artifacts.commits.length}
            size="small"
            color="success"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
        )}
        {block.artifacts.tickets_created.length > 0 && (
          <Chip
            icon={<ConfirmationNumberIcon sx={{ fontSize: 14 }} />}
            label={block.artifacts.tickets_created.length}
            size="small"
            color="info"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
        )}
        {block.artifacts.diary_writes > 0 && (
          <Chip
            icon={<EditNoteIcon sx={{ fontSize: 14 }} />}
            label={block.artifacts.diary_writes}
            size="small"
            color="warning"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
        )}
        <Typography variant="caption" color="text.secondary" sx={{ fontFamily: "monospace" }}>
          {block.agent_turns}t / {block.tool_calls}tc
        </Typography>
      </Stack>
    </Box>
  );
}

export const BlockHeader = memo(BlockHeaderImpl);
export { PersonIcon, SmartToyIcon };
