import { useEffect, useRef } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView, lineNumbers } from "@codemirror/view";
import { sql } from "@codemirror/lang-sql";
import { oneDark } from "@codemirror/theme-one-dark";
import Box from "@mui/material/Box";

interface SqlCodeViewerProps {
  value: string;
  minHeight?: number;
}

export function SqlCodeViewer({ value, minHeight = 120 }: SqlCodeViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const state = EditorState.create({
      doc: "",
      extensions: [
        lineNumbers(),
        sql(),
        oneDark,
        EditorState.readOnly.of(true),
        EditorView.editable.of(false),
        EditorView.lineWrapping,
        EditorView.theme({
          "&": {
            fontSize: "13px",
            fontFamily: '"IBM Plex Mono", "Fira Code", monospace',
          },
          ".cm-content": { minHeight: `${minHeight}px` },
          ".cm-scroller": { overflow: "auto" },
          ".cm-activeLine": { backgroundColor: "transparent" },
          ".cm-gutters": {
            backgroundColor: "#282c34",
            borderRight: "1px solid rgba(255,255,255,0.08)",
          },
        }),
      ],
    });

    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    return () => {
      view.destroy();
    };
  }, [minHeight]);

  useEffect(() => {
    const view = viewRef.current;
    if (view && view.state.doc.toString() !== value) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: value },
      });
    }
  }, [value]);

  return (
    <Box
      ref={containerRef}
      sx={{
        border: "1px solid",
        borderColor: "divider",
        borderRadius: 1,
        overflow: "hidden",
        "& .cm-editor": { height: "100%" },
        "& .cm-scroller": { fontFamily: '"IBM Plex Mono", monospace' },
      }}
    />
  );
}
