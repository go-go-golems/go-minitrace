function _src(paths) {
  const mt = require("minitrace");
  const s = mt.sources();
  for (const p of paths) s.File(p);
  return s.Build();
}
function _dbFrom(paths, extra) {
  const mt = require("minitrace");
  let db = mt.db().Sources(_src(paths)).SQLiteMemory();
  if (extra) db = extra(db);
  return db.Build();
}
function corpusManifest() {
  const paths = [
    "/private/tmp/gmt015-persona/inputs/archive.json",
    "/private/tmp/gmt015-persona/inputs/archive-copy.json",
    "/private/tmp/gmt015-persona/inputs/archive-conflict.json",
  ];
  const m = require("minitrace").corpusManifest(paths);
  return {
    entries: m.entries,
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    sessions: m.sessions,
  };
}
__verb__("corpusManifest", { name: "corpus-manifest", short: "corpus-manifest" });
