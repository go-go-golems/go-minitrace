import { useState } from "react";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import FormControl from "@mui/material/FormControl";
import InputLabel from "@mui/material/InputLabel";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import Chip from "@mui/material/Chip";
import TextField from "@mui/material/TextField";
import Alert from "@mui/material/Alert";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import type { AnnotationCategory } from "../../types";
import { useCreateAnnotationMutation } from "../../api/minitrace";
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

export interface AnnotationDraftTarget {
  scopeType: "session" | "turn" | "tool_call";
  targetId: string;
}

interface AnnotationComposerProps {
  sessionId: string;
  target: AnnotationDraftTarget;
  onCancel: () => void;
  onCreated?: () => void;
  title?: string;
  compact?: boolean;
}

export function AnnotationComposer({
  sessionId,
  target,
  onCancel,
  onCreated,
  title = "Add annotation",
  compact = false,
}: AnnotationComposerProps) {
  const [createAnnotation, { isLoading: isCreating }] = useCreateAnnotationMutation();
  const [category, setCategory] = useState<AnnotationCategory>("observation");
  const [annotationTitle, setAnnotationTitle] = useState("");
  const [detail, setDetail] = useState("");
  const [tags, setTags] = useState("");
  const [error, setError] = useState<string | null>(null);

  const handleCreate = async () => {
    if (!annotationTitle.trim()) {
      setError("Title is required");
      return;
    }
    setError(null);
    try {
      await createAnnotation({
        session_id: sessionId,
        category,
        title: annotationTitle.trim(),
        detail: detail.trim(),
        scope_type: target.scopeType,
        target_id: target.targetId,
        tags: tags
          ? tags.split(",").map((t) => t.trim()).filter(Boolean)
          : [],
      }).unwrap();
      setCategory("observation");
      setAnnotationTitle("");
      setDetail("");
      setTags("");
      onCreated?.();
    } catch {
      setError("Failed to create annotation");
    }
  };

  return (
    <Paper sx={{ p: 2, mt: compact ? 0 : 2 }} variant="outlined">
      <Stack spacing={1.5}>
        <Typography variant="subtitle2">{title}</Typography>

        <FormControl size="small" fullWidth>
          <InputLabel>Category</InputLabel>
          <Select
            value={category}
            label="Category"
            onChange={(e) => setCategory(e.target.value as AnnotationCategory)}
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
          value={annotationTitle}
          onChange={(e) => setAnnotationTitle(e.target.value)}
          placeholder="Brief description"
          autoFocus
        />

        <TextField
          label="Detail"
          size="small"
          fullWidth
          multiline
          minRows={compact ? 1 : 2}
          value={detail}
          onChange={(e) => setDetail(e.target.value)}
          placeholder="Optional detail..."
        />

        <Alert severity="info" variant="outlined">
          Scope: <strong>{target.scopeType}</strong>
          {target.scopeType !== "session" ? ` · target ${target.targetId}` : ""}
        </Alert>

        <TextField
          label="Tags"
          size="small"
          fullWidth
          value={tags}
          onChange={(e) => setTags(e.target.value)}
          placeholder="comma-separated tags"
          helperText="e.g. auth, regression, slow"
        />

        {error && <Alert severity="error">{error}</Alert>}

        <Stack direction="row" spacing={1} justifyContent="flex-end">
          <Button size="small" variant="text" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            size="small"
            variant="contained"
            onClick={handleCreate}
            disabled={isCreating || !annotationTitle.trim()}
          >
            {isCreating ? "Saving..." : "Save"}
          </Button>
        </Stack>
      </Stack>
    </Paper>
  );
}
