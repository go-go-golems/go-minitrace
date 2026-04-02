import Chip from "@mui/material/Chip";

interface ActiveBadgeProps {
  activePct: number;
}

/** Color-coded activity badge: green >50%, amber 10-50%, red <10% */
export function ActiveBadge({ activePct }: ActiveBadgeProps) {
  const color =
    activePct > 50 ? "success" : activePct > 10 ? "warning" : "error";
  return (
    <Chip
      label={`${activePct.toFixed(0)}%`}
      size="small"
      color={color}
      variant="outlined"
      sx={{ fontFamily: "monospace", minWidth: 48, fontWeight: 600 }}
    />
  );
}
