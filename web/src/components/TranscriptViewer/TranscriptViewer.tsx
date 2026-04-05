import { useEffect, useMemo, useState } from "react";
import Box from "@mui/material/Box";
import Paper from "@mui/material/Paper";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import Tabs from "@mui/material/Tabs";
import Tab from "@mui/material/Tab";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import QueryStatsIcon from "@mui/icons-material/QueryStats";
import CommentIcon from "@mui/icons-material/Comment";
import type { Annotation, SessionDetail } from "../../types";
import { useGetSessionAnnotationsQuery } from "../../api/minitrace";
import { ActiveBadge, FormatWallActive } from "../shared";
import { BlockCard } from "./BlockCard";
import { AnnotationPanel } from "./AnnotationPanel";

interface TranscriptViewerProps {
  session: SessionDetail;
  onBack: () => void;
  onQuerySession: (id: string) => void;
}

interface FocusedTranscriptTarget {
  scopeType: "session" | "turn" | "tool_call";
  targetId: string;
  nonce: number;
}

interface DraftAnnotationTarget {
  scopeType: "session" | "turn" | "tool_call";
  targetId: string;
}

export function TranscriptViewer({
  session,
  onBack,
  onQuerySession,
}: TranscriptViewerProps) {
  const [view, setView] = useState<"transcript" | "annotations">("transcript");
  const [focusedTarget, setFocusedTarget] = useState<FocusedTranscriptTarget | null>(null);
  const [draftTarget, setDraftTarget] = useState<DraftAnnotationTarget | null>(null);

  const { data: annotationData } = useGetSessionAnnotationsQuery(session.id);
  const annotations = annotationData?.annotations ?? [];

  const activePct =
    (session.timing.active_duration_seconds /
      Math.max(session.timing.duration_seconds, 1)) *
    100;

  const annotationIndex = useMemo(() => {
    const byTurn: Record<string, Annotation[]> = {};
    const byToolCall: Record<string, Annotation[]> = {};
    const sessionScoped: Annotation[] = [];

    for (const ann of annotations) {
      if (ann.scope.type === "session") {
        sessionScoped.push(ann);
      }
      if (ann.scope.type === "turn") {
        (byTurn[ann.scope.target_id] ??= []).push(ann);
      }
      if (ann.scope.type === "tool_call") {
        (byToolCall[ann.scope.target_id] ??= []).push(ann);
      }
    }

    return { byTurn, byToolCall, sessionScoped };
  }, [annotations]);

  const focusedBlockNum = useMemo(() => {
    if (!focusedTarget) {
      return null;
    }
    if (focusedTarget.scopeType === "turn") {
      for (const block of session.blocks) {
        if (block.turns.some((t) => String(t.idx) === focusedTarget.targetId)) {
          return block.block_num;
        }
      }
    }
    if (focusedTarget.scopeType === "tool_call") {
      for (const block of session.blocks) {
        if (
          block.turns.some((t) =>
            t.tool_calls_in_turn.some((tc) => tc.id === focusedTarget.targetId),
          )
        ) {
          return block.block_num;
        }
      }
    }
    return null;
  }, [focusedTarget, session.blocks]);

  useEffect(() => {
    if (view !== "transcript" || !focusedTarget) {
      return;
    }

    const timer = window.setTimeout(() => {
      let selector = "";
      if (focusedTarget.scopeType === "session") {
        selector = `[data-session-top="${session.id}"]`;
      }
      if (focusedTarget.scopeType === "turn") {
        selector = `[data-turn-idx="${focusedTarget.targetId}"]`;
      }
      if (focusedTarget.scopeType === "tool_call") {
        selector = `[data-tool-call-id="${focusedTarget.targetId}"]`;
      }
      const el = selector
        ? (document.querySelector(selector) as HTMLElement | null)
        : null;
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 80);

    return () => window.clearTimeout(timer);
  }, [view, focusedTarget, session.id]);

  useEffect(() => {
    if (!focusedTarget) {
      return;
    }
    const timer = window.setTimeout(() => setFocusedTarget(null), 2500);
    return () => window.clearTimeout(timer);
  }, [focusedTarget]);

  const handleNavigateToAnnotationTarget = (annotation: { scope: { type: "session" | "turn" | "tool_call"; target_id: string } }) => {
    setView("transcript");
    setFocusedTarget({
      scopeType: annotation.scope.type,
      targetId: annotation.scope.target_id || session.id,
      nonce: Date.now(),
    });
  };

  const handleCreateScopedAnnotation = (
    scopeType: "session" | "turn" | "tool_call",
    targetId: string,
  ) => {
    setDraftTarget({ scopeType, targetId });
    setView("annotations");
  };

  return (
    <Box
      data-widget="transcript-viewer"
      sx={{ height: "100%", display: "flex", flexDirection: "column" }}
    >
      {/* Header */}
      <Box sx={{ p: 2, display: "flex", alignItems: "center", gap: 2 }}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={onBack}
          size="small"
          variant="text"
        >
          Sessions
        </Button>
        <Typography
          variant="caption"
          sx={{ fontFamily: "monospace", color: "text.secondary" }}
        >
          {session.id.slice(0, 8)}
        </Typography>
        <Typography variant="h3" sx={{ flex: 1 }}>
          {session.title.slice(0, 100)}
        </Typography>
        <Button
          startIcon={<QueryStatsIcon />}
          onClick={() => onQuerySession(session.id)}
          size="small"
          variant="outlined"
        >
          Query
        </Button>
      </Box>

      {/* Session info bar */}
      <Paper
        data-session-top={session.id}
        sx={{
          mx: 2,
          mb: 2,
          p: 1.5,
          border: "1px solid",
          borderColor:
            focusedTarget?.scopeType === "session" ? "warning.main" : "divider",
          bgcolor:
            focusedTarget?.scopeType === "session"
              ? "rgba(245,166,35,0.08)"
              : undefined,
          transition: "background-color 0.2s, border-color 0.2s",
        }}
      >
        <Stack direction="row" spacing={3} alignItems="center" flexWrap="wrap">
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              Started
            </Typography>
            <Typography variant="body2" sx={{ fontFamily: "monospace" }}>
              {new Date(session.timing.started_at).toLocaleString()}
            </Typography>
          </Stack>
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="caption" color="text.secondary">
              Duration
            </Typography>
            <FormatWallActive
              wallSeconds={session.timing.duration_seconds}
              activeSeconds={session.timing.active_duration_seconds}
            />
          </Stack>
          <ActiveBadge activePct={activePct} />
          <Chip
            label={`${session.metrics.turn_count} turns`}
            size="small"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
          <Chip
            label={`${session.metrics.tool_call_count} tool calls`}
            size="small"
            variant="outlined"
            sx={{ fontFamily: "monospace" }}
          />
          <Chip
            label={session.environment.model}
            size="small"
            color="secondary"
            variant="outlined"
          />
          {annotationIndex.sessionScoped.length > 0 && (
            <Chip
              label={`${annotationIndex.sessionScoped.length} session annotation${annotationIndex.sessionScoped.length === 1 ? "" : "s"}`}
              size="small"
              color="warning"
              variant="outlined"
            />
          )}
          <Typography
            variant="caption"
            sx={{ fontFamily: "monospace", color: "text.secondary" }}
          >
            {session.operational_context.working_directory}
          </Typography>
        </Stack>
      </Paper>

      {/* View toggle */}
      <Box sx={{ px: 2, pb: 1 }}>
        <Tabs
          value={view}
          onChange={(_, v) => setView(v)}
          sx={{ minHeight: 36 }}
        >
          <Tab
            value="transcript"
            label={`Transcript (${session.blocks.length} blocks)`}
            sx={{ minHeight: 36, py: 0 }}
          />
          <Tab
            value="annotations"
            icon={<CommentIcon sx={{ fontSize: 16 }} />}
            iconPosition="start"
            label="Annotations"
            sx={{ minHeight: 36, py: 0 }}
          />
        </Tabs>
      </Box>

      {/* Content */}
      <Box sx={{ flex: 1, overflow: "auto", px: 2, pb: 2 }}>
        {view === "transcript" && (
          <>
            <Typography variant="overline" color="text.secondary" sx={{ mb: 1 }}>
              {session.blocks.length} blocks
            </Typography>
            {session.blocks.map((block) => (
              <BlockCard
                key={block.block_num}
                block={block}
                defaultExpanded={block.block_num === 1}
                forceExpanded={focusedBlockNum === block.block_num}
                focusedTarget={focusedTarget}
                turnAnnotations={annotationIndex.byTurn}
                toolCallAnnotations={annotationIndex.byToolCall}
                onCreateScopedAnnotation={handleCreateScopedAnnotation}
              />
            ))}
          </>
        )}
        {view === "annotations" && (
          <AnnotationPanel
            sessionId={session.id}
            onClose={() => setView("transcript")}
            onNavigateToTarget={handleNavigateToAnnotationTarget}
            draftTarget={draftTarget}
            onDraftHandled={() => setDraftTarget(null)}
          />
        )}
      </Box>
    </Box>
  );
}
