const transforms = require("./lib/transforms");

__section__("options", {
  title: "Options",
  fields: {
    prefix: {
      type: "string",
      default: "demo",
      help: "Prefix used when generating synthetic rows",
    },
    tags: {
      type: "stringList",
      help: "Synthetic tags used to generate preview cards",
    },
    include_runtime: {
      type: "bool",
      default: false,
      help: "Whether to include DB/runtime details in the emitted row",
    },
  },
});

function showContext(options) {
  const mt = require("minitrace");
  const runtime = mt.runtime || {};
  return {
    command_name: runtime.commandName || "",
    table_name: runtime.tableName || "",
    archive_glob_count: Array.isArray(runtime.archiveGlob) ? runtime.archiveGlob.length : 0,
    prefix: options.prefix,
    include_runtime: !!options.include_runtime,
    db_path: options.include_runtime ? runtime.dbPath || "" : "hidden",
  };
}

function buildSyntheticRows(options) {
  const tags = Array.isArray(options.tags) && options.tags.length > 0
    ? options.tags
    : ["alpha", "beta", "gamma"];
  return transforms.makePreviewCards(options.prefix || "demo", tags);
}

__verb__("showContext", {
  name: "show-context",
  short: "Return runtime metadata without querying DuckDB",
  fields: {
    options: { bind: "options" },
  },
});

__verb__("buildSyntheticRows", {
  name: "build-synthetic-rows",
  short: "Generate rows entirely in JS from typed inputs",
  fields: {
    options: { bind: "options" },
  },
});
