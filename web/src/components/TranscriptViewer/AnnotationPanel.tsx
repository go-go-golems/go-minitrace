import { useEffect, useState } from "react";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import IconButton from "@mui/material/IconButton";
import Alert from "@mui/material/Alert";
import CircularProgress from "@mui/material/CircularProgress";
import CloseIcon from "@mui/icons-material/Close";
import AddIcon from "@mui/icons-material/Add";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import SyncIcon from "@mui/icons-material/Sync";
import {
  useGetSessionAnnotationsQuery,
  useDeleteAnnotationMutation,
  useSyncAnnotationsMutation,
} from "../../api/minitrace";
import type { Annotation } from "../../types";
import { ANNOTATION_CATEGORY_COLORS as CATEGORY_COLORS } from "../../types/session";
import { AnnotationComposer } from "./AnnotationComposer";

interface AnnotationPanelProps {
  sessionId: string;
  onClose: () => void;
  onNavigateToTarget?: (annotation: Annotation) => void;
  selectedAnnotationId?: string | null;
}

export function AnnotationPanel({
  sessionId,
  onClose,
  onNavigateToTarget,
  selectedAnnotationId = null,
}: AnnotationPanelProps) {
  const { data, isLoading, isError } = useGetSessionAnnotationsQuery(sessionId);
  const [deleteAnnotation] = useDeleteAnnotationMutation();
  const [syncAnnotations, { isLoading: isSyncing }] = useSyncAnnotationsMutation();

  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const annotations = data?.annotations ?? [];

  useEffect(() => {
    if (!selectedAnnotationId) {
      return;
    }
    const timer = window.setTimeout(() => {
      const el = document.querySelector(
        `[data-annotation-id="${selectedAnnotationId}"]`,
      ) as HTMLElement | null;
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 80);
    return () => window.clearTimeout(timer);
  }, [selectedAnnotationId]);

  const handleDelete = async (annotationId: string) => {
    try {
      await deleteAnnotation({ id: annotationId, session_id: sessionId }).unwrap();
    } catch {
      setError("Failed to delete annotation");
    }
  };

  const handleSync = async () => {
    try {
      await syncAnnotations({ session_id: sessionId }).unwrap();
    } catch {
      setError("Sync failed");
    }
  };

  return (
    <Box
      data-widget="annotation-panel"
      sx={{ height: "100%", display: "flex", flexDirection: "column" }}
    >
      {/* Header */}
      <Box
        sx={{
          p: 2,
          display: "flex",
          alignItems: "center",
          gap: 1,
          borderBottom: "1px solid",
          borderColor: "divider",
        }}
      >
        <Typography variant="h6" sx={{ flex: 1 }}>
          Annotations
        </Typography>
        <Typography
          variant="caption"
          sx={{ fontFamily: "monospace", color: "text.secondary" }}
        >
          {sessionId.slice(0, 8)}
        </Typography>
        <IconButton size="small" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      {/* Annotation list */}
      <Box sx={{ flex: 1, overflow: "auto", p: 2 }}>
        {isLoading && (
          <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
            <CircularProgress size={24} />
          </Box>
        )}

        {isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            Failed to load annotations. Is the annotation store available?
          </Alert>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!isLoading && annotations.length === 0 && !showForm && (
          <Typography
            variant="body2"
            color="text.secondary"
            sx={{ textAlign: "center", py: 4 }}
          >
            No annotations yet. Add one below.
          </Typography>
        )}

        {annotations.map((ann) => (
          <AnnotationCard
            key={ann.id}
            annotation={ann}
            onDelete={() => handleDelete(ann.id)}
            onNavigateToTarget={onNavigateToTarget}
            selected={selectedAnnotationId === ann.id}
          />
        ))}

        {/* Add form */}
        {showForm && (
          <AnnotationComposer
            sessionId={sessionId}
            target={{ scopeType: "session", targetId: sessionId }}
            title="Add session annotation"
            onCancel={() => {
              setShowForm(false);
            }}
            onCreated={() => {
              setShowForm(false);
            }}
          />
        )}
      </Box>

      {/* Footer actions */}
      <Box
        sx={{
          p: 2,
          borderTop: "1px solid",
          borderColor: "divider",
          display: "flex",
          gap: 1,
        }}
      >
        {!showForm && (
          <Button
            size="small"
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setShowForm(true)}
          >
            Add
          </Button>
        )}
        <Button
          size="small"
          variant="outlined"
          startIcon={<SyncIcon />}
          onClick={handleSync}
          disabled={isSyncing || annotations.length === 0}
        >
          {isSyncing ? "Syncing..." : "Sync to JSON"}
        </Button>
      </Box>
    </Box>
  );
}

// ── AnnotationCard ────────────────────────────────────────────────────────────

interface AnnotationCardProps {
  annotation: Annotation;
  onDelete: () => void;
  onNavigateToTarget?: (annotation: Annotation) => void;
  selected?: boolean;
}

function formatScopeLabel(annotation: Annotation) {
  if (annotation.scope.type === "session") {
    return "Session";
  }
  if (annotation.scope.type === "turn") {
    return `Turn #${annotation.scope.target_id}`;
  }
  return `Tool call ${annotation.scope.target_id.slice(0, 12)}…`;
}

function AnnotationCard({
  annotation,
  onDelete,
  onNavigateToTarget,
  selected = false,
}: AnnotationCardProps) {
  const color = CATEGORY_COLORS[annotation.content.category] ?? "default";
  return (
    <Paper
      data-annotation-id={annotation.id}
      onClick={() => onNavigateToTarget?.(annotation)}
      sx={{
        p: 1.5,
        mb: 1,
        borderLeft: 3,
        borderColor: selected ? "warning.main" : `${color}.main`,
        bgcolor: selected ? "rgba(245,166,35,0.08)" : undefined,
        cursor: onNavigateToTarget ? "pointer" : "default",
        transition: "background-color 0.15s, border-color 0.15s",
        '&:hover': onNavigateToTarget ? { bgcolor: 'action.hover' } : undefined,
      }}
      variant="outlined"
    >
      <Box sx={{ display: "flex", alignItems: "flex-start", gap: 1 }}>
        <Box sx={{ flex: 1 }}>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 0.5 }}>
            <Chip
              label={annotation.content.category}
              size="small"
              color={color}
              sx={{ fontSize: "0.7rem" }}
            />
            {annotation.content.tags.map((tag) => (
              <Chip key={tag} label={tag} size="small" variant="outlined" sx={{ fontSize: "0.65rem" }} />
            ))}
          </Stack>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>
            {annotation.content.title}
          </Typography>
          {annotation.content.detail && (
            <Typography variant="caption" color="text.secondary" sx={{ display: "block", mt: 0.5 }}>
              {annotation.content.detail}
            </Typography>
          )}
          {annotation.taxonomy_mappings.minitrace.length > 0 && (
            <Typography
              variant="caption"
              sx={{ fontFamily: "monospace", color: "text.secondary", mt: 0.5, display: "block" }}
            >
              {annotation.taxonomy_mappings.minitrace.join(", ")}
            </Typography>
          )}
          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, display: "block" }}>
            {formatScopeLabel(annotation)}
            {onNavigateToTarget ? ' · click to jump to transcript' : ''}
          </Typography>
          <Typography variant="caption" color="text.disabled" sx={{ mt: 0.25, display: "block" }}>
            {annotation.annotator} · {new Date(annotation.timestamp).toLocaleString()}
          </Typography>
        </Box>
        <IconButton
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            onDelete();
          }}
          sx={{ mt: -0.5 }}
        >
          <DeleteOutlineIcon fontSize="small" />
        </IconButton>
      </Box>
    </Paper>
  );
}
