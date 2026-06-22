__package__({
  name: "inspect",
  short: "Inspect minitrace archives and raw session exports with require(\"minitrace\")",
});

function openDB(file) {
  const mt = require("minitrace");
  return mt.db().File(file).QueryCommandDefaults().Build();
}

function summary(file) {
  const db = openDB(file);
  try {
    const row = db.queryOne(`
      SELECT
        session_id,
        title,
        agent_framework,
        working_directory,
        turn_count,
        tool_call_count,
        started_at
      FROM sessions
      ORDER BY session_id
      LIMIT 1
    `);
    return {
      file,
      session_id: row.session_id,
      title: row.title,
      framework: row.agent_framework,
      working_directory: row.working_directory,
      turns: row.turn_count,
      tools: row.tool_call_count,
      started_at: row.started_at,
    };
  } finally {
    db.close();
  }
}

__verb__("summary", {
  name: "summary",
  short: "Print high-level metadata for one minitrace archive",
  fields: {
    file: {
      argument: true,
      help: "Path to a .minitrace.json archive",
    },
  },
});

function tools(file, limit) {
  const db = openDB(file);
  try {
    return db.query(`
      SELECT
        tool_name,
        COALESCE(operation_type, '') AS operation_type,
        COUNT(*) AS calls,
        SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) AS failures
      FROM tool_calls
      GROUP BY tool_name, operation_type
      ORDER BY calls DESC, tool_name ASC, operation_type ASC
      LIMIT ?
    `, limit || 10);
  } finally {
    db.close();
  }
}

__verb__("tools", {
  name: "tools",
  short: "Summarize tool-call counts for one minitrace archive",
  fields: {
    file: {
      argument: true,
      help: "Path to a .minitrace.json archive",
    },
    limit: {
      type: "int",
      default: 10,
      help: "Maximum number of tool rows to return",
    },
  },
});

function preview(file) {
  const mt = require("minitrace");
  const importer = mt.importer()
    .File(file)
    .AutoDetect()
    .Convert();
  const converted = importer.Converted();
  const preview = importer.Preview();
  return {
    file,
    converted,
    preview: {
      sessionId: preview.sessionId,
      format: preview.format,
      adapter: preview.adapter,
      framework: preview.agentFramework,
      turns: preview.turnCount,
      tools: preview.toolCallCount,
      roles: preview.roleCounts,
      toolCounts: preview.toolCounts,
      sampleTurns: preview.sampleTurns,
      sampleTools: preview.sampleTools,
    },
  };
}

__verb__("preview", {
  name: "preview",
  short: "Auto-detect and preview a raw session export before saving it",
  fields: {
    file: {
      argument: true,
      help: "Path to a raw supported session export, for example Pi JSONL",
    },
  },
});

function autoConvert(file, cacheDir, limit) {
  const mt = require("minitrace");
  const sources = mt.sources().File(file).Build();
  const importPolicy = mt.importPolicy().AutoConvert().Strict().Build();
  const cache = mt.cache().Auto().Dir(cacheDir).Build();
  const limits = mt.limits().Rows(limit || 25).Build();
  const builder = mt.db()
    .Sources(sources)
    .Import(importPolicy)
    .Cache(cache)
    .Limits(limits);
  const cacheKey = builder.CacheKey();
  const db = builder.Build();
  try {
    const session = db.queryOne(`
      SELECT
        session_id,
        agent_framework,
        turn_count,
        tool_call_count
      FROM sessions
      LIMIT 1
    `);
    const roles = db.query(`
      SELECT role, COUNT(*) AS turns
      FROM turns
      GROUP BY role
      ORDER BY role
    `);
    return {
      file,
      cacheKey: cacheKey.key,
      cache: db.cacheInfo(),
      diagnostics: db.diagnostics(),
      sources: db.sources(),
      session,
      roles,
    };
  } finally {
    db.close();
  }
}

__verb__("autoConvert", {
  name: "auto-convert",
  short: "Build a queryable DB from a raw export using sources, import policy, cache, and limits builders",
  fields: {
    file: {
      argument: true,
      help: "Path to a raw supported session export, for example Pi JSONL",
    },
    cacheDir: {
      type: "string",
      default: "./dist/cache",
      help: "Directory used by the cache builder",
    },
    limit: {
      type: "int",
      default: 25,
      help: "Maximum rows allowed by the limits builder",
    },
  },
});

function saveConverted(file, out) {
  const mt = require("minitrace");
  const importer = mt.importer()
    .File(file)
    .AutoDetect()
    .Into(out)
    .Overwrite()
    .Convert();
  return {
    converted: importer.Converted(),
    saved: importer.Save(),
  };
}

__verb__("saveConverted", {
  name: "save-converted",
  short: "Convert a raw export and save it as a .minitrace.json archive directory",
  fields: {
    file: {
      argument: true,
      help: "Path to a raw supported session export, for example Pi JSONL",
    },
    out: {
      type: "string",
      default: "./dist/converted",
      help: "Output root directory for saved converted sessions",
    },
  },
});
