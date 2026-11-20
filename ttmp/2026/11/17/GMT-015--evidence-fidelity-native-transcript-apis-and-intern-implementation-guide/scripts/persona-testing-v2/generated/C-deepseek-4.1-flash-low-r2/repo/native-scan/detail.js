function nativeDetail(filters) {
  const mt = require("minitrace");
  const path = "/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl";
  const insp = mt.native().File(path)
    .MaxRecordBytes(16 * 1024 * 1024)
    .MaxIndexBytes(256 * 1024 * 1024)
    .Inspect();
  const scan = mt.native().File(path).ForEach(function(r) {});
  const reqs = (insp.requests || []).map(function(r) {
    return { id: r.id, status: r.status, source: r.source, shapeHash: r.shapeHash, refs: r.refs, diagnostics: r.diagnostics };
  });
  const comps = (insp.compactions || []).map(function(c) {
    return {
      boundarySource: c.boundary.source,
      boundaryUuid: c.boundary.uuid,
      boundaryParentUuid: c.boundary.parentUuid === undefined ? null : c.boundary.parentUuid,
      boundaryType: c.boundary.type,
      logicalParentUuid: c.logicalParentUuid === undefined ? null : c.logicalParentUuid,
      metadata: c.metadata === undefined ? null : c.metadata,
      summaries: (c.summaries || []).map(function(s) { return { uuid: s.uuid, source: s.source }; }),
      beforeMessage: c.beforeMessage === undefined ? null : { uuid: c.beforeMessage.uuid, source: c.beforeMessage.source },
      firstFollowingMessage: c.firstFollowingMessage === undefined ? null : { uuid: c.firstFollowingMessage.uuid, source: c.firstFollowingMessage.source },
      firstFollowingAssistant: c.firstFollowingAssistant === undefined ? null : { uuid: c.firstFollowingAssistant.uuid, source: c.firstFollowingAssistant.source }
    };
  });
  return [{
    path: path,
    scan: { sha256: scan.sha256, hashScope: scan.hashScope, bytes: scan.bytes, lines: scan.lines, malformed: scan.malformed, stopReason: scan.stopReason },
    requests: reqs,
    compactions: comps,
    diagnostics: insp.diagnostics,
    limitations: insp.limitations
  }];
}
__verb__("nativeDetail", {
  name: "native-detail",
  short: "Synthetic fixture native request/compaction/diagnostic detail"
});
