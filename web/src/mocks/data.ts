import type {
  SessionSummary,
  SessionDetail,
  Turn,
  ToolCall,
  QueryCommand,
  SavedQuery,
} from "../types";

// ── helpers ──────────────────────────────────────────────────────────

function toolCall(
  overrides: Partial<ToolCall> & { tool_name: string }
): ToolCall {
  return {
    id: `call_${Math.random().toString(36).slice(2, 10)}`,
    timestamp: "2026-03-22T20:48:00Z",
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

function turn(overrides: Partial<Turn> & { idx: number; role: Turn["role"] }): Turn {
  return {
    content: "",
    source: overrides.role === "user" ? "human" : "model",
    timestamp: "2026-03-22T20:48:00Z",
    tool_calls_in_turn: [],
    ...overrides,
  };
}

// ── session summaries ────────────────────────────────────────────────

export const mockSessions: SessionSummary[] = [
  {
    id: "019d174c-fc68-7c00-8f1b-7fcc067c1fd6",
    title:
      "Read the docs in geppetto and create a new ticket to bring wesen-os/ up to the new profiles and profile registry settings",
    summary: null,
    classification: "internal",
    timing: {
      started_at: "2026-03-22T20:47:44Z",
      ended_at: "2026-03-23T21:26:00Z",
      duration_seconds: 88896,
      active_duration_seconds: 11160,
      hour_of_day: 20,
      day_of_week: 0,
    },
    metrics: { turn_count: 315, tool_call_count: 1015 },
    environment: { agent_framework: "codex", model: "gpt-5.4" },
    operational_context: {
      working_directory: "~/workspaces/2026-03-02/os-openai-app-server",
    },
  },
  {
    id: "019d376d-0103-7dc3-a96d-650c7c2e1cf7",
    title:
      "Let's work on ticket NPM-PUBLISH-001. Read the guide and create detailed tasks on how to do the whole migration towards renaming, published npm packages",
    summary: null,
    classification: "internal",
    timing: {
      started_at: "2026-03-29T02:30:51Z",
      ended_at: "2026-04-01T18:00:00Z",
      duration_seconds: 315000,
      active_duration_seconds: 44280,
      hour_of_day: 2,
      day_of_week: 6,
    },
    metrics: { turn_count: 1467, tool_call_count: 3307 },
    environment: { agent_framework: "codex", model: "gpt-5.4" },
    operational_context: {
      working_directory:
        "~/workspaces/2026-03-02/os-openai-app-server/wesen-os",
    },
  },
  {
    id: "019d4a35-9c8d-7f10-8fef-ef0650432725",
    title:
      "Work on ttmp/2026/04/01/SQLITE-FED-001--sqlite-federated-remote-release-handoff/index.md docmgr ticket",
    summary: null,
    classification: "internal",
    timing: {
      started_at: "2026-04-01T18:02:27Z",
      ended_at: "2026-04-01T19:20:00Z",
      duration_seconds: 4653,
      active_duration_seconds: 2880,
      hour_of_day: 18,
      day_of_week: 2,
    },
    metrics: { turn_count: 121, tool_call_count: 300 },
    environment: { agent_framework: "codex", model: "gpt-5.4" },
    operational_context: {
      working_directory:
        "~/workspaces/2026-03-02/os-openai-app-server/wesen-os",
    },
  },
  {
    id: "019d2f26-228e-7f23-ba6b-7946aafa514c",
    title:
      "Create a new docmgr ticket to deploy this setup to hetzner, step by step, explaining to me what the current step is",
    summary: null,
    classification: "internal",
    timing: {
      started_at: "2026-03-27T11:56:00Z",
      ended_at: "2026-03-29T22:00:00Z",
      duration_seconds: 210600,
      active_duration_seconds: 56160,
      hour_of_day: 11,
      day_of_week: 4,
    },
    metrics: { turn_count: 1807, tool_call_count: 4440 },
    environment: { agent_framework: "codex", model: "gpt-5.4" },
    operational_context: {
      working_directory: "~/code/wesen/2026-03-27--hetzner-k3s",
    },
  },
  {
    id: "019d0112-69ba-7232-9b14-875797183903",
    title:
      "Create a new ticket to add hashicorp plugin support to go-go-goja, so that we can register plugins into a runtime",
    summary: null,
    classification: "internal",
    timing: {
      started_at: "2026-03-18T13:12:11Z",
      ended_at: "2026-03-19T01:52:16Z",
      duration_seconds: 45605,
      active_duration_seconds: 19474,
      hour_of_day: 13,
      day_of_week: 2,
    },
    metrics: { turn_count: 563, tool_call_count: 1423 },
    environment: { agent_framework: "codex", model: "gpt-5.4" },
    operational_context: {
      working_directory: "~/workspaces/2026-03-18/add-goja-plugins",
    },
  },
  {
    id: "019d0295-d06b-7033-b154-a991a94672b6",
    title:
      "Let's tackle an issue in smailnail now that it's deployed on coolify",
    summary: null,
    classification: "confidential",
    timing: {
      started_at: "2026-03-18T20:15:22Z",
      ended_at: "2026-03-19T02:30:00Z",
      duration_seconds: 22723,
      active_duration_seconds: 9720,
      hour_of_day: 20,
      day_of_week: 2,
    },
    metrics: { turn_count: 243, tool_call_count: 589 },
    environment: { agent_framework: "codex", model: "gpt-5.4" },
    operational_context: {
      working_directory: "~/workspaces/2026-03-08/update-imap-mcp",
    },
  },
];

// ── session detail (blocks for 019d174c) ─────────────────────────────

const block1Turns: Turn[] = [
  turn({
    idx: 1,
    role: "user",
    timestamp: "2026-03-22T20:47:44Z",
    content:
      'Read the docs in geppetto and create a new ticket to bring wesen-os/ up to the new profiles and profile registry settings, and have it load the pinocchio config and profile files (not wesen-os ones).\n\nCreate the docmgr ticket, keep a diary.\n\nThen create a detailed analysis / design / implementation guide for the migration...',
  }),
  turn({
    idx: 2,
    role: "assistant",
    timestamp: "2026-03-22T20:47:54Z",
    content:
      'Using `ticket-research-docmgr-remarkable` with `diary` support because this is a ticketed research/doc deliverable that needs a structured workspace, diary, and reMarkable upload. I\'m starting by reading the skill instructions and locating the relevant docs in `geppetto`, `wesen-os`, and `pinocchio` so I can ground the ticket in the current system rather than guessing.',
    tool_calls_in_turn: [
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-22T20:47:54Z",
        input: { command: "pwd", arguments: { cmd: "pwd" } },
        output: {
          success: true,
          result: "/home/manuel/workspaces/2026-03-02/os-openai-app-server",
          error: null,
          duration_ms: 259,
          truncated: false,
        },
      }),
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-22T20:48:02Z",
        input: {
          command: "ls -la",
          arguments: { cmd: "ls -la geppetto/pkg/profiles/" },
        },
        output: {
          success: true,
          result:
            "total 48\ndrwxr-xr-x 3 manuel manuel 4096 profiles.go\ndrwxr-xr-x 3 manuel manuel 4096 registry.go",
          error: null,
          duration_ms: 120,
          truncated: false,
        },
      }),
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-22T20:51:32Z",
        input: {
          command: "docmgr ticket create-ticket",
          arguments: {
            cmd: 'docmgr ticket create-ticket --ticket APP-30-WESEN-OS-PINOCCHIO-PROFILE-BOOTSTRAP --title "Migrate wesen-os to Pinocchio config/bootstrap and engine-profile registries"',
          },
        },
        output: {
          success: true,
          result: "Ticket workspace created: APP-30-WESEN-OS-PINOCCHIO-PROFILE-BOOTSTRAP",
          error: null,
          duration_ms: 450,
          truncated: false,
        },
        badges: ["ticket-create"],
      }),
    ],
  }),
  turn({
    idx: 3,
    role: "assistant",
    timestamp: "2026-03-22T20:52:00Z",
    content:
      "I've confirmed the repo contains sibling `geppetto`, `pinocchio`, and `wesen-os` trees plus existing ticket workspaces. Next I'm reading the ticket/docmgr/diary skill instructions and then I'll inspect the profile-registry docs and examples in `geppetto` to identify the exact migration surface.",
  }),
];

const block2Turns: Turn[] = [
  turn({
    idx: 25,
    role: "user",
    timestamp: "2026-03-23T02:36:27Z",
    content: "how does this all impact go-go-os-chat?",
  }),
  turn({
    idx: 26,
    role: "assistant",
    timestamp: "2026-03-23T02:37:00Z",
    content:
      "The impact is significant. `go-go-os-chat` still depends on the legacy `geppetto/pkg/profiles` package for runtime types and request construction. The migration needs to update the chat handler to use the new profile-registry resolver chain instead of the old in-memory mixed profiles.",
    tool_calls_in_turn: [
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-23T02:37:10Z",
        input: {
          command: "grep -r",
          arguments: {
            cmd: 'grep -rn "geppetto/pkg/profiles" go-go-os-chat/',
          },
        },
        output: {
          success: true,
          result:
            "go-go-os-chat/pkg/chat/handler.go:12:import gprofiles \"github.com/go-go-golems/geppetto/pkg/profiles\"\ngo-go-os-chat/pkg/chat/request.go:8:import gprofiles ...",
          error: null,
          duration_ms: 180,
          truncated: false,
        },
      }),
    ],
  }),
];

const block8Turns: Turn[] = [
  turn({
    idx: 53,
    role: "user",
    timestamp: "2026-03-23T03:08:49Z",
    content: "continue",
  }),
  turn({
    idx: 54,
    role: "assistant",
    timestamp: "2026-03-23T03:09:00Z",
    content:
      "Continuing with the migration tasks. First I'm updating the launcher entrypoint to use the Pinocchio bootstrap config loader instead of the hardcoded profile construction.",
    tool_calls_in_turn: [
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-23T03:09:10Z",
        input: {
          command: "cat",
          arguments: {
            cmd: "cat cmd/wesen-os-launcher/main.go | head -50",
          },
        },
        output: {
          success: true,
          result: 'package main\n\nimport (\n\t"context"\n\t...',
          error: null,
          duration_ms: 80,
          truncated: false,
        },
      }),
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-23T03:27:00Z",
        input: {
          command: "git commit",
          arguments: {
            cmd: 'git commit -m "refactor(profilechat): resolve engine profiles with pinocchio runtime"',
          },
        },
        output: {
          success: true,
          result:
            "[main 4a7b2c1] refactor(profilechat): resolve engine profiles with pinocchio runtime\n 3 files changed, 47 insertions(+), 89 deletions(-)",
          error: null,
          duration_ms: 2100,
          truncated: false,
        },
        badges: ["commit"],
      }),
      toolCall({
        tool_name: "exec_command",
        timestamp: "2026-03-23T03:30:00Z",
        input: {
          command: "go test",
          arguments: {
            cmd: "go test ./cmd/wesen-os-launcher/... -count=1 -v",
          },
        },
        output: {
          success: false,
          result: null,
          error:
            "FAIL\tcmd/wesen-os-launcher\t0.015s\n--- FAIL: TestBootstrap (0.01s)\n    main_test.go:42: expected profile registry to contain 3 profiles, got 0",
          duration_ms: 15000,
          truncated: false,
        },
        badges: ["error"],
      }),
    ],
  }),
  turn({
    idx: 55,
    role: "assistant",
    timestamp: "2026-03-23T03:31:00Z",
    content:
      "The test is failing because the bootstrap config isn't loading the profile fixture. I need to add the test fixture YAML and point the test to it. Fixing now.",
  }),
];

export const mockSessionDetail: SessionDetail = {
  id: "019d174c-fc68-7c00-8f1b-7fcc067c1fd6",
  title:
    "Read the docs in geppetto and create a new ticket to bring wesen-os/ up to the new profiles",
  summary: null,
  classification: "internal",
  timing: {
    started_at: "2026-03-22T20:47:44Z",
    ended_at: "2026-03-23T21:26:00Z",
    duration_seconds: 88896,
    active_duration_seconds: 11160,
    hour_of_day: 20,
    day_of_week: 0,
  },
  metrics: { turn_count: 315, tool_call_count: 1015 },
  environment: { agent_framework: "codex", model: "gpt-5.4" },
  operational_context: {
    working_directory: "~/workspaces/2026-03-02/os-openai-app-server",
  },
  provenance: {
    source_format: "codex-session-jsonl-v1",
    source_path: "/home/manuel/.codex/sessions/2026/03/22/rollout-2026-03-22.jsonl",
    original_session_id: "019d174c-fc68-7c00-8f1b-7fcc067c1fd6",
    converted_at: "2026-04-01T19:23:05Z",
  },
  blocks: [
    {
      block_num: 1,
      user_turn_idx: 1,
      user_ts: "2026-03-22T20:47:44Z",
      user_content:
        "Read the docs in geppetto and create a new ticket to bring wesen-os/ up to the new profiles...",
      agent_turns: 23,
      tool_calls: 130,
      gap_minutes: null,
      turns: block1Turns,
      artifacts: {
        commits: [],
        tickets_created: ["APP-30-WESEN-OS-PINOCCHIO-PROFILE-BOOTSTRAP"],
        docs_added: [
          "Investigation diary (reference)",
          "Intern guide to migrating... (design-doc)",
        ],
        diary_writes: 1,
      },
    },
    {
      block_num: 2,
      user_turn_idx: 25,
      user_ts: "2026-03-23T02:36:27Z",
      user_content: "how does this all impact go-go-os-chat?",
      agent_turns: 2,
      tool_calls: 4,
      gap_minutes: 349,
      turns: block2Turns,
      artifacts: {
        commits: [],
        tickets_created: [],
        docs_added: [],
        diary_writes: 0,
      },
    },
    {
      block_num: 8,
      user_turn_idx: 53,
      user_ts: "2026-03-23T03:08:49Z",
      user_content: "continue",
      agent_turns: 47,
      tool_calls: 257,
      gap_minutes: 5,
      turns: block8Turns,
      artifacts: {
        commits: [
          "refactor(profilechat): resolve engine profiles with pinocchio runtime",
        ],
        tickets_created: [],
        docs_added: [],
        diary_writes: 2,
      },
    },
  ],
};

// ── preset / saved queries ───────────────────────────────────────────

export const mockPresets: SavedQuery[] = [
  {
    name: "session-list",
    folder: "core",
    path: "core/session-list.sql",
    description: "List all sessions sorted by start time with key metrics",
    sql: "SELECT id, timing->>'started_at' AS started_at, title,\n  CAST(metrics->>'turn_count' AS INT) AS turns,\n  CAST(metrics->>'tool_call_count' AS INT) AS tools\nFROM sessions_base\nORDER BY timing->>'started_at';",
    readonly: true,
  },
  {
    name: "human-blocks",
    folder: "analysis",
    path: "analysis/human-blocks.sql",
    description:
      "Decompose a session into human-input blocks with agent turn and tool counts",
    sql: "-- Replace SESSION_ID\nWITH numbered AS (\n  SELECT t.idx,\n    CAST(t.turn->>'role' AS VARCHAR) AS role,\n    CAST(t.turn->>'content' AS VARCHAR) AS content,\n    CAST(t.turn->>'timestamp' AS VARCHAR) AS ts,\n    json_array_length(COALESCE(t.turn->'tool_calls_in_turn', '[]'::JSON)) AS tc_count\n  FROM sessions_base\n  CROSS JOIN UNNEST(turns) WITH ORDINALITY AS t(turn, idx)\n  WHERE id = 'SESSION_ID'\n)\nSELECT * FROM numbered LIMIT 20;",
    readonly: true,
  },
  {
    name: "git-commits",
    folder: "analysis",
    path: "analysis/git-commits.sql",
    description: "All successful git commits with messages",
    sql: "SELECT s.id, CAST(tc->>'timestamp' AS VARCHAR) AS ts,\n  LEFT(CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR), 200) AS cmd\nFROM sessions_base s\nCROSS JOIN UNNEST(tool_calls) AS t(tc)\nWHERE CAST(tc->>'tool_name' AS VARCHAR) = 'exec_command'\n  AND CAST(tc->'input'->'arguments'->>'cmd' AS VARCHAR) LIKE '%git commit%'\n  AND CAST(tc->'output'->>'success' AS BOOLEAN) = true;",
    readonly: true,
  },
  {
    name: "tool-breakdown",
    folder: "analysis",
    path: "analysis/tool-breakdown.sql",
    description: "Tool call breakdown by tool name",
    sql: "SELECT\n  REPLACE(CAST(tc->>'tool_name' AS VARCHAR), '\"', '') AS tool,\n  COUNT(*) AS calls\nFROM sessions_base\nCROSS JOIN UNNEST(tool_calls) AS t(tc)\nGROUP BY tool\nORDER BY calls DESC;",
    readonly: true,
  },
];

export const mockQueryCommands: QueryCommand[] = [
  {
    name: "session-list",
    folder: "core",
    path: "session-list.sql",
    shortDescription: "List minitrace sessions",
    longDescription: "List sessions with optional framework and title filters.",
    flags: [
      {
        name: "framework",
        type: "stringList",
        help: "Filter by agent framework",
        required: false,
        defaultJson: "[]",
        choices: [],
        positional: false,
        shortFlag: "",
      },
      {
        name: "title_like",
        type: "string",
        help: "Filter titles with LIKE",
        required: false,
        defaultJson: "",
        choices: [],
        positional: false,
        shortFlag: "",
      },
      {
        name: "limit",
        type: "int",
        help: "Limit the number of rows returned",
        required: false,
        defaultJson: "100",
        choices: [],
        positional: false,
        shortFlag: "",
      },
    ],
    arguments: [],
    tags: ["analysis"],
    readonly: true,
    kind: "verb",
    aliasFor: "",
  },
  {
    name: "codex-framework-summary",
    folder: "aliases",
    path: "aliases/codex-framework-summary.alias.yaml",
    shortDescription: "Summarize only codex sessions",
    longDescription: "Alias for framework-summary with framework preset to codex.",
    flags: [
      {
        name: "framework",
        type: "stringList",
        help: "Restrict the summary to selected frameworks",
        required: false,
        defaultJson: '["codex"]',
        choices: [],
        positional: false,
        shortFlag: "",
      },
    ],
    arguments: [],
    tags: ["analysis"],
    readonly: true,
    kind: "alias",
    aliasFor: "framework-summary",
  },
];

export const mockSavedQueries: SavedQuery[] = [
  {
    name: "wesen-os-filter",
    folder: "my-queries",
    path: "my-queries/wesen-os-filter.sql",
    description: "Find sessions referencing wesen-os in title, workdir, or first-turn content",
    sql: "SELECT id, timing->>'started_at' AS started_at, title,\n  operational_context->>'working_directory' AS workdir\nFROM sessions_base\nWHERE LOWER(title) LIKE '%wesen-os%'\n  OR LOWER(operational_context->>'working_directory') LIKE '%wesen-os%'\nORDER BY timing->>'started_at';",
    readonly: false,
  },
];

// ── query result ─────────────────────────────────────────────────────

export const mockQueryResult = {
  columns: ["id", "started_at", "title", "turns", "tools"],
  rows: mockSessions.map((s) => ({
    id: s.id,
    started_at: s.timing.started_at,
    title: s.title.slice(0, 80),
    turns: s.metrics.turn_count,
    tools: s.metrics.tool_call_count,
  })),
  duration_ms: 12,
  row_count: mockSessions.length,
};
