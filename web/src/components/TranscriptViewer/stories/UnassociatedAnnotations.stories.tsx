import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within, waitFor } from "storybook/test";
import { configureStore } from "@reduxjs/toolkit";
import { Provider } from "react-redux";
import { MemoryRouter } from "react-router";
import { minitraceApi } from "../../../api/minitrace";
import { uiReducer } from "../../../store";
import { mockSessionDetail } from "../../../mocks/data";
import type { Annotation } from "../../../types";
import { TranscriptViewer } from "../TranscriptViewer";

const sessionId = "review-unassociated";
const callId = "detached-call";
const annotation: Annotation = {
  id: "detached-note", timestamp: "2026-09-06T00:00:00Z", annotator: "reviewer",
  scope: { type: "tool_call", target_id: callId },
  content: { category: "observation", tags: [], title: "Unassociated record note", detail: "Retain annotation access without inventing a turn." },
  taxonomy_mappings: { minitrace: [], mast: [], toolemu: [] },
};

const meta = {
  title: "TranscriptViewer/UnassociatedAnnotations",
  component: TranscriptViewer,
  decorators: [(Story) => {
    const store = configureStore({
      reducer: { [minitraceApi.reducerPath]: minitraceApi.reducer, ui: uiReducer },
      middleware: (defaults) => defaults().concat(minitraceApi.middleware),
    });
    store.dispatch(minitraceApi.util.upsertQueryEntries([{
      endpointName: "getSessionAnnotations", arg: sessionId,
      value: { session_id: sessionId, count: 1, annotations: [annotation] },
    }]));
    return <Provider store={store}><MemoryRouter><Story /></MemoryRouter></Provider>;
  }],
  args: {
    session: {
      ...mockSessionDetail, id: sessionId, blocks: [],
      unassociated_tool_calls: [{
        id: callId, record_kind: "execution", tool_name: "exec_command",
        timestamp: "2026-09-06T00:00:00Z", operation_type: "EXECUTE",
        input: { command: "printf detached" },
        output: { success: null, status: "pending", result: null, error: null, duration_ms: 0, truncated: false },
        badges: [],
      }],
    },
    onBack: fn(), onQuerySession: fn(),
  },
} satisfies Meta<typeof TranscriptViewer>;
export default meta;
type Story = StoryObj<typeof meta>;

export const CreateAndOpenAnnotations: Story = {
  play: async ({ canvas, canvasElement }) => {
    await userEvent.click(canvas.getByRole("button", { name: "Show next unassociated records" }));
    const row = canvasElement.querySelector(`[data-tool-call-id="${callId}"]`);
    if (!(row instanceof HTMLElement)) throw new Error("Unassociated row missing");
    const tool = within(row);
    await expect(await tool.findByText("observation")).toBeVisible();
    await userEvent.click(tool.getByRole("button", { name: "Annotate" }));
    const body = within(canvasElement.ownerDocument.body);
    const dialog = await body.findByRole("dialog");
    await waitFor(() => expect(within(dialog).getByText("Add tool_call annotation")).toBeVisible());
    await expect(within(dialog).getByText(/target detached-call/)).toBeVisible();
    await userEvent.click(within(dialog).getAllByRole("button", { name: "Cancel" })[0]);
    await waitFor(() => expect(body.queryByRole("dialog")).not.toBeInTheDocument());
    await userEvent.click(tool.getByText("observation"));
    await waitFor(() => expect(canvas.getByRole("tab", { name: /Annotations/ })).toHaveAttribute("aria-selected", "true"));
    await expect((await canvas.findAllByText(annotation.content.title))[0]).toBeVisible();
  },
};
