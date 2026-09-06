import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect } from "storybook/test";
import Box from "@mui/material/Box";
import { ToolCallRow } from "../ToolCallRow";
import { withTheme } from "../../../test-utils/storybook-decorators";
import type { ToolCall } from "../../../types";

function tc(overrides: Partial<ToolCall>): ToolCall {
  return {
    id: `call_${Math.random().toString(36).slice(2)}`,
    tool_name: "exec_command",
    timestamp: "2026-03-23T03:27:00Z",
    operation_type: "EXECUTE",
    input: { command: "pwd", arguments: {} },
    output: {
      success: true,
      result: "/home/manuel/workspaces/wesen-os",
      error: null,
      duration_ms: 250,
      truncated: false,
    },
    badges: [],
    ...overrides,
  };
}

const meta = {
  title: "TranscriptViewer/ToolCallRow",
  component: ToolCallRow,
  decorators: [
    withTheme,
    (Story) => (
      <Box sx={{ p: 2, bgcolor: "background.paper", maxWidth: 800 }}>
        <Story />
      </Box>
    ),
  ],
} satisfies Meta<typeof ToolCallRow>;

export default meta;
type Story = StoryObj<typeof meta>;

function neutralOutcome(status: "unknown" | "pending" | "cancelled"): Story {
  return {
    args: {
      tc: tc({
        id: `outcome-${status}`,
        output: { success: null, status, result: null, error: null, duration_ms: 0, truncated: false },
      }),
    },
    play: async ({ canvas }) => {
      await expect(canvas.getByText(status)).toBeVisible();
      await expect(canvas.queryByTitle("succeeded")).not.toBeInTheDocument();
      await expect(canvas.queryByTitle("failed")).not.toBeInTheDocument();
    },
  };
}

export const UnknownOutcome: Story = neutralOutcome("unknown");
export const PendingOutcome: Story = neutralOutcome("pending");
export const CancelledOutcome: Story = neutralOutcome("cancelled");

export const OutcomeStates: Story = {
  args: { tc: tc({}) },
  render: () => (
    <Box>
      {(["succeeded", "failed", "unknown", "pending", "cancelled"] as const).map((status) => (
        <ToolCallRow key={status} tc={tc({
          id: `outcome-${status}`,
          input: { command: `example: ${status}`, arguments: {} },
          output: {
            success: status === "succeeded" ? true : status === "failed" ? false : null,
            status, result: null, error: null, duration_ms: 0, truncated: false,
          },
        })} />
      ))}
    </Box>
  ),
};

export const SimpleSuccess: Story = {
  args: {
    tc: tc({
      input: { command: "pwd", arguments: { cmd: "pwd" } },
      output: { success: true, result: "/home/manuel/workspaces/wesen-os", error: null, duration_ms: 120, truncated: false },
    }),
  },
};

export const LongCommand: Story = {
  args: {
    tc: tc({
      input: {
        command: "grep",
        arguments: {
          cmd: 'grep -rn "geppetto/pkg/profiles" go-go-os-chat/pkg/chat/ wesen-os/cmd/ pinocchio/pkg/bootstrap/ --include="*.go" | head -50',
        },
      },
      output: {
        success: true,
        result: 'go-go-os-chat/pkg/chat/handler.go:12:import gprofiles "github.com/go-go-golems/geppetto/pkg/profiles"\ngo-go-os-chat/pkg/chat/request.go:8:import gprofiles "github.com/go-go-golems/geppetto/pkg/profiles"',
        error: null,
        duration_ms: 340,
        truncated: false,
      },
    }),
  },
};

export const GitCommit: Story = {
  args: {
    tc: tc({
      input: {
        command: "git commit",
        arguments: {
          cmd: 'git commit -m "refactor(profilechat): resolve engine profiles with pinocchio runtime"',
        },
      },
      output: {
        success: true,
        result: "[main 4a7b2c1] refactor(profilechat): resolve engine profiles with pinocchio runtime\n 3 files changed, 47 insertions(+), 89 deletions(-)",
        error: null,
        duration_ms: 2100,
        truncated: false,
      },
      badges: ["commit"],
    }),
  },
};

export const TicketCreation: Story = {
  args: {
    tc: tc({
      tool_name: "exec_command",
      input: {
        command: "docmgr ticket create-ticket",
        arguments: {
          cmd: 'docmgr ticket create-ticket --ticket APP-30-WESEN-OS --title "Migrate wesen-os to Pinocchio"',
        },
      },
      output: {
        success: true,
        result: "Ticket workspace created: APP-30-WESEN-OS",
        error: null,
        duration_ms: 450,
        truncated: false,
      },
      badges: ["ticket-create"],
    }),
  },
};

export const FailedTestRun: Story = {
  args: {
    tc: tc({
      input: {
        command: "go test",
        arguments: { cmd: "go test ./cmd/wesen-os-launcher/... -count=1 -v" },
      },
      output: {
        success: false,
        result: null,
        error: "FAIL\tcmd/wesen-os-launcher\t0.015s\n--- FAIL: TestBootstrap (0.01s)\n    main_test.go:42: expected profile registry to contain 3 profiles, got 0",
        duration_ms: 15000,
        truncated: false,
      },
      badges: ["error"],
    }),
  },
};

export const DiaryWrite: Story = {
  args: {
    tc: tc({
      input: {
        command: "apply_patch",
        arguments: {
          cmd: 'apply_patch <<\'EOF\'\n--- a/ttmp/.../reference/01-diary.md\n+++ b/ttmp/.../reference/01-diary.md\n@@ -1,3 +1,15 @@\n+## Step 3: Implement profile resolver chain\n+\n+Replaced the legacy in-memory profiles...\nEOF',
        },
      },
      output: {
        success: true,
        result: "Patch applied successfully",
        error: null,
        duration_ms: 50,
        truncated: false,
      },
      badges: ["diary-write"],
    }),
  },
};

export const Expanded: Story = {
  args: {
    tc: tc({
      input: { command: "cat", arguments: { cmd: "cat geppetto/pkg/profiles/registry.go | head -80" } },
      output: {
        success: true,
        result: 'package profiles\n\nimport (\n\t"fmt"\n\t"sync"\n)\n\ntype Registry struct {\n\tmu       sync.RWMutex\n\tprofiles map[string]*Profile\n}\n\nfunc NewRegistry() *Registry {\n\treturn &Registry{profiles: make(map[string]*Profile)}\n}',
        error: null,
        duration_ms: 80,
        truncated: false,
      },
    }),
    defaultExpanded: true,
  },
};

export const MultipleBadges: Story = {
  args: {
    tc: tc({
      input: { command: "git commit", arguments: { cmd: 'git commit -m "fix: diary + commit"' } },
      output: { success: true, result: "[main abc123] fix", error: null, duration_ms: 500, truncated: false },
      badges: ["commit", "diary-write"],
    }),
  },
};
