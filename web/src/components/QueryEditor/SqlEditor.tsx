import { useRef, useEffect } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine } from "@codemirror/view";
import { sql } from "@codemirror/lang-sql";
import { oneDark } from "@codemirror/theme-one-dark";
import { defaultKeymap } from "@codemirror/commands";
import Box from "@mui/material/Box";

interface SqlEditorProps {
  value: string;
  onChange: (value: string) => void;
  onExecute?: () => void;
}

export function SqlEditor({ value, onChange, onExecute }: SqlEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const runKey = keymap.of([
      {
        key: "Ctrl-Enter",
        mac: "Cmd-Enter",
        run: () => {
          onExecute?.();
          return true;
        },
      },
    ]);

    const state = EditorState.create({
      doc: value,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        sql(),
        oneDark,
        runKey,
        keymap.of(defaultKeymap),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            onChange(update.state.doc.toString());
          }
        }),
        EditorView.theme({
          "&": {
            fontSize: "13px",
            fontFamily: '"IBM Plex Mono", "Fira Code", monospace',
          },
          ".cm-content": { minHeight: "120px" },
          ".cm-scroller": { overflow: "auto" },
        }),
      ],
    });

    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    return () => {
      view.destroy();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync external value changes
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
      data-part="sql-editor"
      sx={{
        flex: 1,
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
