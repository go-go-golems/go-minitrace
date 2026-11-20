const PATHS = [
  "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json",
  "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-copy.json",
  "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-conflict.json",
];

function manifest() {
  const mt = require("minitrace");
  try {
    return { ok: true, manifest: mt.corpusManifest(PATHS) };
  } catch (e) {
    return { ok: false, error: String(e && e.message ? e.message : e) };
  }
}

__verb__("manifest", {
  name: "manifest",
  short: "Corpus manifest over the three synthetic archives",
  fields: {},
});
