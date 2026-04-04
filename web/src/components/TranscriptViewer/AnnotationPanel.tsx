import { useState } from "react";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import IconButton from "@mui/material/IconButton";
import TextField from "@mui/material/TextField";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import Alert from "@mui/material/Alert";
import CircularProgress from "@mui/material/CircularProgress";
import CloseIcon from "@mui/icons-material/Close";
import AddIcon from "@mui/icons-material/Add";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import SyncIcon from "@mui/icons-material/Sync";
import {
  useGetSessionAnnotationsQuery,
  useCreateAnnotationMutation,
  useDeleteAnnotationMutation,
  useSyncAnnotationsMutation,
} from "../../api/minitrace";
import type {
  Annotation,
  AnnotationCategory,
} from "../../types";
import { ANNOTATION_CATEGORY_COLORS as CATEGORY_COLORS } from "../../types/session";

const CATEGORIES: AnnotationCategory[] = [
  "observation",
  "ai-failure",
  "user-error",
  "environment-issue",
  "success",
  "question",
  "to-discuss",
  "to-improve",
];

interface NewAnnotation {
  category: AnnotationCategory;
  title: string;
  detail: string;
  tags: string;
}

interface AnnotationPanelProps {
  sessionId: string;
  onClose: () => void;
}

export function AnnotationPanel({
  sessionId,
  onClose,
}: AnnotationPanelProps) {
  const { data, isLoading, isError } = useGetSessionAnnotationsQuery(sessionId);
  const [createAnnotation, { isLoading: isCreating }] = useCreateAnnotationMutation();
  const [deleteAnnotation] = useDeleteAnnotationMutation();
  const [syncAnnotations, { isLoading: isSyncing }] = useSyncAnnotationsMutation();

  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<NewAnnotation>({
    category: "observation",
    title: "",
    detail: "",
    tags: "",
  });
  const [error, setError] = useState<string | null>(null);

  const annotations = data?.annotations ?? [];

  const handleCreate = async () => {
    if (!form.title.trim()) {
      setError("Title is required");
      return;
    }
    setError(null);
    try {
      await createAnnotation({
        session_id: sessionId,
        category: form.category,
        title: form.title.trim(),
        detail: form.detail.trim(),
        tags: form.tags
          ? form.tags
              .split(",")
              .map((t) => t.trim())
              .filter(Boolean)
          : [],
      }).unwrap();
      setForm({ category: "observation", title: "", detail: "", tags: "" });
      setShowForm(false);
    } catch {
      setError("Failed to create annotation");
    }
  };

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
          />
        ))}

        {/* Add form */}
        {showForm && (
          <Paper sx={{ p: 2, mt: 2 }} variant="outlined">
            <Stack spacing={1.5}>
              <FormControl size="small" fullWidth>
                <InputLabel>Category</InputLabel>
                <Select
                  value={form.category}
                  label="Category"
                  onChange={(e) =>
                    setForm((f) => ({ ...f, category: e.target.value as AnnotationCategory }))
                  }
                >
                  {CATEGORIES.map((cat) => (
                    <MenuItem key={cat} value={cat}>
                      <Chip
                        label={cat}
                        size="small"
                        color={CATEGORY_COLORS[cat] ?? "default"}
                        sx={{ mr: 1 }}
                      />
                      {cat}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              <TextField
                label="Title"
                size="small"
                fullWidth
                value={form.title}
                onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                placeholder="Brief description"
                autoFocus
              />

              <TextField
                label="Detail"
                size="small"
                fullWidth
                multiline
                minRows={2}
                value={form.detail}
                onChange={(e) => setForm((f) => ({ ...f, detail: e.target.value }))}
                placeholder="Optional detail..."
              />

              <TextField
                label="Tags"
                size="small"
                fullWidth
                value={form.tags}
                onChange={(e) => setForm((f) => ({ ...f, tags: e.target.value }))}
                placeholder="comma-separated tags"
                helperText="e.g. auth, regression, slow"
              />

              {error && <Alert severity="error">{error}</Alert>}

              <Stack direction="row" spacing={1} justifyContent="flex-end">
                <Button
                  size="small"
                  variant="text"
                  onClick={() => {
                    setShowForm(false);
                    setError(null);
                  }}
                >
                  Cancel
                </Button>
                <Button
                  size="small"
                  variant="contained"
                  onClick={handleCreate}
                  disabled={isCreating || !form.title.trim()}
                >
                  {isCreating ? "Saving..." : "Save"}
                </Button>
              </Stack>
            </Stack>
          </Paper>
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
}

function AnnotationCard({ annotation, onDelete }: AnnotationCardProps) {
  const color = CATEGORY_COLORS[annotation.content.category] ?? "default";
  return (
    <Paper
      sx={{ p: 1.5, mb: 1, borderLeft: 3, borderColor: `${color}.main` }}
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
          <Typography variant="caption" color="text.disabled" sx={{ mt: 0.5, display: "block" }}>
            {annotation.annotator} · {new Date(annotation.timestamp).toLocaleString()}
          </Typography>
        </Box>
        <IconButton size="small" onClick={onDelete} sx={{ mt: -0.5 }}>
          <DeleteOutlineIcon fontSize="small" />
        </IconButton>
      </Box>
    </Paper>
  );
}
