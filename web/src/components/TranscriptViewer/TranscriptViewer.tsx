import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useSearchParams } from "react-router";
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
import { useVirtualList } from "../shared/useVirtualList";
import { ActiveBadge, FormatWallActive } from "../shared";
import { BlockCard } from "./BlockCard";
import { AnnotationPanel } from "./AnnotationPanel";
import {
  AnnotationComposer,
  type AnnotationDraftTarget,
} from "./AnnotationComposer";
import type { FocusedTranscriptTarget } from "./types";

interface TranscriptViewerProps {
  session: SessionDetail;
  onBack: () => void;
  onQuerySession: (id: string) => void;
}

export function TranscriptViewer({
  session,
  onBack,
  onQuerySession,
}: TranscriptViewerProps) {
  const [searchParams, setSearchParams] = useSearchParams();
  const [focusedTarget, setFocusedTarget] = useState<FocusedTranscriptTarget | null>(null);
  const [expandedBlocks, setExpandedBlocks] = useState<Record<number, boolean>>({});
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const firstBlockNum = session.blocks[0]?.block_num;
    setExpandedBlocks((current) => {
      if (firstBlockNum == null) {
        return Object.keys(current).length === 0 ? current : {};
      }
      return current[firstBlockNum] && Object.keys(current).length === 1
        ? current
        : { [firstBlockNum]: true };
    });
  }, [session.id, session.blocks]);

  const { data: annotationData } = useGetSessionAnnotationsQuery(session.id);
  const annotations = annotationData?.annotations ?? [];

  const view = searchParams.get("tab") === "annotations" ? "annotations" : "transcript";
  const selectedAnnotationId = searchParams.get("annotation");
  const focusTypeParam = searchParams.get("focusType");
  const focusIdParam = searchParams.get("focusId");
  const composeTypeParam = searchParams.get("composeType");
  const composeTargetParam = searchParams.get("composeTarget");

  const urlFocusedTarget = useMemo(
    () =>
      focusTypeParam && focusIdParam
        ? ({
            scopeType: focusTypeParam as "session" | "turn" | "tool_call",
            targetId: focusIdParam,
            nonce: 0,
          } satisfies FocusedTranscriptTarget)
        : null,
    [focusIdParam, focusTypeParam],
  );

  const draftTarget = useMemo(
    () =>
      composeTypeParam && composeTargetParam
        ? ({
            scopeType: composeTypeParam as "session" | "turn" | "tool_call",
            targetId: composeTargetParam,
          } satisfies AnnotationDraftTarget)
        : null,
    [composeTargetParam, composeTypeParam],
  );

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
    if (!urlFocusedTarget) {
      return null;
    }
    if (urlFocusedTarget.scopeType === "turn") {
      for (const block of session.blocks) {
        if (block.turns.some((t) => String(t.idx) === urlFocusedTarget.targetId)) {
          return block.block_num;
        }
      }
    }
    if (urlFocusedTarget.scopeType === "tool_call") {
      for (const block of session.blocks) {
        if (
          block.turns.some((t) =>
            t.tool_calls_in_turn.some((tc) => tc.id === urlFocusedTarget.targetId),
          )
        ) {
          return block.block_num;
        }
      }
    }
    return null;
  }, [urlFocusedTarget, session.blocks]);

  const firstBlockNum = session.blocks[0]?.block_num ?? 1;

  const isBlockExpanded = useCallback(
    (blockNum: number) => expandedBlocks[blockNum] ?? blockNum === firstBlockNum,
    [expandedBlocks, firstBlockNum],
  );

  const estimateBlockSize = useCallback(
    (index: number) => {
      const block = session.blocks[index];
      if (!block) {
        return 88;
      }
      if (isBlockExpanded(block.block_num) || focusedBlockNum === block.block_num) {
        return 900;
      }
      return 88;
    },
    [focusedBlockNum, isBlockExpanded, session.blocks],
  );

  const {
    virtualItems,
    topSpacerHeight,
    bottomSpacerHeight,
    measureElement,
    scrollToIndex,
  } = useVirtualList({
    count: session.blocks.length,
    scrollContainerRef,
    estimateSize: estimateBlockSize,
    overscan: 4,
    enabled: session.blocks.length > 10,
  });

  const setUrlState = useCallback((patch: Record<string, string | null>) => {
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        for (const [k, v] of Object.entries(patch)) {
          if (v == null || v === "") {
            next.delete(k);
          } else {
            next.set(k, v);
          }
        }
        return next;
      },
      { replace: false },
    );
  }, [setSearchParams]);

  useEffect(() => {
    if (view !== "transcript" || focusedBlockNum == null) {
      return;
    }

    const index = session.blocks.findIndex((block) => block.block_num === focusedBlockNum);
    if (index >= 0) {
      scrollToIndex(index, "auto", "center");
    }
  }, [focusedBlockNum, scrollToIndex, session.blocks, view]);

  useEffect(() => {
    if (view !== "transcript" || !urlFocusedTarget) {
      return;
    }

    setFocusedTarget({ ...urlFocusedTarget, nonce: Date.now() });

    const timer = window.setTimeout(() => {
      let selector = "";
      if (urlFocusedTarget.scopeType === "session") {
        selector = `[data-session-top="${session.id}"]`;
      }
      if (urlFocusedTarget.scopeType === "turn") {
        selector = `[data-turn-idx="${urlFocusedTarget.targetId}"]`;
      }
      if (urlFocusedTarget.scopeType === "tool_call") {
        selector = `[data-tool-call-id="${urlFocusedTarget.targetId}"]`;
      }
      const el = selector
        ? (document.querySelector(selector) as HTMLElement | null)
        : null;
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 120);

    return () => window.clearTimeout(timer);
  }, [view, urlFocusedTarget, session.id]);

  const handleNavigateToAnnotationTarget = useCallback((annotation: Annotation) => {
    setUrlState({
      tab: "transcript",
      focusType: annotation.scope.type,
      focusId: annotation.scope.target_id || session.id,
      annotation: null,
      composeType: null,
      composeTarget: null,
    });
  }, [session.id, setUrlState]);

  const handleOpenAnnotation = useCallback((annotation: Annotation) => {
    setUrlState({
      tab: "annotations",
      annotation: annotation.id,
      focusType: null,
      focusId: null,
      composeType: null,
      composeTarget: null,
    });
  }, [setUrlState]);

  const handleCreateScopedAnnotation = useCallback((
    scopeType: "session" | "turn" | "tool_call",
    targetId: string,
  ) => {
    setUrlState({
      tab: "transcript",
      focusType: scopeType,
      focusId: targetId,
      annotation: null,
      composeType: scopeType,
      composeTarget: targetId,
    });
  }, [setUrlState]);

  const handleToggleBlock = useCallback((blockNum: number) => {
    setExpandedBlocks((current) => ({
      ...current,
      [blockNum]: !(current[blockNum] ?? blockNum === firstBlockNum),
    }));
  }, [firstBlockNum]);

  return (
    <Box
      data-widget="transcript-viewer"
      sx={{ height: "100%", display: "flex", flexDirection: "column" }}
    >
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

      <Box sx={{ px: 2, pb: 1 }}>
        <Tabs
          value={view}
          onChange={(_, v) =>
            setUrlState({
              tab: v,
              annotation: v === "annotations" ? selectedAnnotationId : null,
              composeType: null,
              composeTarget: null,
            })}
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

      <Box ref={scrollContainerRef} sx={{ flex: 1, overflow: "auto", px: 2, pb: 2 }}>
        <Box
          sx={{ display: view === "transcript" ? "block" : "none" }}
          aria-hidden={view !== "transcript"}
        >
          {draftTarget && (
            <AnnotationComposer
              sessionId={session.id}
              target={draftTarget}
              title={`Add ${draftTarget.scopeType} annotation`}
              compact
              onCancel={() =>
                setUrlState({
                  composeType: null,
                  composeTarget: null,
                })}
              onCreated={() =>
                setUrlState({
                  composeType: null,
                  composeTarget: null,
                })}
            />
          )}
          <Typography
            variant="overline"
            color="text.secondary"
            sx={{ mb: 1, display: "block", mt: draftTarget ? 1 : 0 }}
          >
            {session.blocks.length} blocks
          </Typography>
          {topSpacerHeight > 0 && <Box sx={{ height: topSpacerHeight }} />}
          {virtualItems.map((item) => {
            const block = session.blocks[item.index];
            return (
              <Box key={block.block_num} ref={measureElement(item.index)}>
                <BlockCard
                  block={block}
                  expanded={isBlockExpanded(block.block_num)}
                  forceExpanded={focusedBlockNum === block.block_num}
                  focusedTarget={focusedTarget}
                  turnAnnotations={annotationIndex.byTurn}
                  toolCallAnnotations={annotationIndex.byToolCall}
                  onCreateScopedAnnotation={handleCreateScopedAnnotation}
                  onOpenAnnotation={handleOpenAnnotation}
                  onToggleExpanded={() => handleToggleBlock(block.block_num)}
                />
              </Box>
            );
          })}
          {bottomSpacerHeight > 0 && <Box sx={{ height: bottomSpacerHeight }} />}
        </Box>
        <Box
          sx={{ display: view === "annotations" ? "block" : "none" }}
          aria-hidden={view !== "annotations"}
        >
          <AnnotationPanel
            sessionId={session.id}
            onClose={() => setUrlState({ tab: "transcript", annotation: null })}
            onNavigateToTarget={handleNavigateToAnnotationTarget}
            selectedAnnotationId={selectedAnnotationId}
          />
        </Box>
      </Box>
    </Box>
  );
}
