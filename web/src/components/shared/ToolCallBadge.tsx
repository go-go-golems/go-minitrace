import Chip from "@mui/material/Chip";
import CommitIcon from "@mui/icons-material/Commit";
import ConfirmationNumberIcon from "@mui/icons-material/ConfirmationNumber";
import DescriptionIcon from "@mui/icons-material/Description";
import EditNoteIcon from "@mui/icons-material/EditNote";
import ErrorOutlineIcon from "@mui/icons-material/ErrorOutline";
import type { ToolCallBadge as BadgeType } from "../../types";

const badgeConfig: Record<
  BadgeType,
  { label: string; color: "success" | "info" | "secondary" | "warning" | "error"; icon: React.ReactElement }
> = {
  commit: { label: "commit", color: "success", icon: <CommitIcon sx={{ fontSize: 14 }} /> },
  "ticket-create": { label: "ticket", color: "info", icon: <ConfirmationNumberIcon sx={{ fontSize: 14 }} /> },
  "doc-add": { label: "doc", color: "secondary", icon: <DescriptionIcon sx={{ fontSize: 14 }} /> },
  "diary-write": { label: "diary", color: "warning", icon: <EditNoteIcon sx={{ fontSize: 14 }} /> },
  error: { label: "error", color: "error", icon: <ErrorOutlineIcon sx={{ fontSize: 14 }} /> },
};

interface ToolCallBadgeProps {
  badge: BadgeType;
}

export function ToolCallBadgeChip({ badge }: ToolCallBadgeProps) {
  const config = badgeConfig[badge];
  return (
    <Chip
      icon={config.icon}
      label={config.label}
      size="small"
      color={config.color}
      variant="filled"
      sx={{ height: 20, fontSize: "0.6875rem", fontWeight: 600 }}
    />
  );
}
