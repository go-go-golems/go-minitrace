// corpusManifest preflight on the three corpus-identity fixtures:
//   archive.json          (baseline)
//   archive-copy.json     (byte-identical copy -> duplicateOf: 0)
//   archive-conflict.json (same normalized session id, different bytes -> conflict)
// Note duplicateOf is a ZERO-BASED index, so 0 is a real duplicate marker and
// a falsy check (`if (entry.duplicateOf)`) would silently keep the copy.

__section__("opts", {
  title: "Options",
  fields: {
    root: { type: "string", default: "/private/tmp/gmt015-persona/inputs", help: "Fixture directory" },
  },
});

function manifest(opts) {
  const mt = require("minitrace");
  const paths = [
    opts.root + "/archive.json",
    opts.root + "/archive-copy.json",
    opts.root + "/archive-conflict.json",
  ];
  const m = mt.corpusManifest(paths);
  return {
    probe: "corpus-manifest",
    inputOrder: paths,
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    sessions: m.sessions,
    entries: (m.entries || []).map((e) => ({
      path: e.path,
      sessionId: e.sessionId,
      originalSessionId: e.originalSessionId,
      sourceFormat: e.sourceFormat,
      sha256: e.sha256,
      sizeBytes: e.sizeBytes,
      duplicateOf: e.duplicateOf,
      duplicateOfIsZero: e.duplicateOf === 0,
      hasOwnDuplicateKey: Object.prototype.hasOwnProperty.call(e, "duplicateOf"),
      conflict: e.conflict,
    })),
    groups: m.groups,
    // Demonstrate the zero-index trap explicitly.
    falsyFilterWouldKeep: (m.entries || []).filter((e) => !!e.duplicateOf).map((e) => e.path),
    strictFilterKeeps: (m.entries || [])
      .filter((e) => e.duplicateOf === null)
      .map((e) => e.path),
  };
}

__verb__("manifest", {
  name: "manifest",
  short: "Corpus identity manifest for the three fixtures",
  fields: { opts: { bind: "opts" } },
});
