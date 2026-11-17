// native-scan.js — scanner-first JS query command using mt.native()
//
// Purpose: forensic scan of one native JSONL source (real or synthetic).
// Reports: whole-source scan facts (sha256, hashScope, bytes, lines,
// malformed, stopReason), record-type histogram (decoded value.type or
// record kind fallback), api-request record count, Inspect() request
// status breakdown (complete vs not-complete), compaction count, and
// diagnostics. Uses only read-only mt.native() streaming; no history is
// executed and no normalized tables are written.

__section__("params", {
  title: "Scan target",
  fields: {
    path: {
      type: "string",
      help: "Absolute path of the native JSONL file to scan",
    },
  },
});

function nativeScan(params) {
  const mt = require("minitrace");
  const path = params.path;
  const maxRecord = 16 * 1024 * 1024;

  const typeCounts = {};
  let apiRequestRecords = 0;
  let diagnosticCount = 0;
  const diagnosticCodes = {};

  const scan = mt.native().File(path)
    .MaxRecordBytes(maxRecord)
    .ForEach(record => {
      if (record.diagnostic) {
        diagnosticCount += 1;
        const code = record.diagnostic.code || "unknown";
        diagnosticCodes[code] = (diagnosticCodes[code] || 0) + 1;
        return; // continue
      }
      let kind = "no-type";
      if (record.value && typeof record.value.type === "string") {
        kind = record.value.type;
      }
      typeCounts[kind] = (typeCounts[kind] || 0) + 1;
      if (record.value && record.value.type === "api-request") {
        apiRequestRecords += 1;
      }
      return; // continue to EOF
    });

  // Second pass with Inspect() to get request-graph status facts.
  const inspection = mt.native().File(path)
    .MaxRecordBytes(maxRecord)
    .Inspect();

  const statusCounts = {};
  for (const r of inspection.requests) {
    statusCounts[r.status] = (statusCounts[r.status] || 0) + 1;
  }

  return [{
    path: path,
    sha256: scan.sha256,
    hashScope: scan.hashScope,
    bytes: scan.bytes,
    lines: scan.lines,
    malformed: scan.malformed,
    stopReason: scan.stopReason,
    typeHistogram: typeCounts,
    apiRequestRecords: apiRequestRecords,
    diagnostics: diagnosticCount,
    diagnosticCodes: diagnosticCodes,
    inspectRequestsTotal: inspection.requests.length,
    requestStatus: statusCounts,
    inspectCompactions: inspection.compactions.length,
    inspectBlobs: Object.keys(inspection.blobs).length,
    inspectShapes: Object.keys(inspection.shapes).length,
    inspectIndexBytes: inspection.indexBytes,
    limitations: inspection.limitations,
  }];
}

__verb__("nativeScan", {
  name: "native-scan",
  short: "Scan one native JSONL with mt.native(): scan facts and request counts",
  fields: {
    params: { bind: "params" },
  },
});

// Optional exploration: compaction projection with explicit absence -> null.
function nativeCompactions(params) {
  const mt = require("minitrace");
  const path = params.path;
  const inspection = mt.native().File(path)
    .MaxRecordBytes(16 * 1024 * 1024)
    .Inspect();
  return inspection.compactions.map(c => ({
    path: path,
    boundaryLine: c.boundary.source && c.boundary.source.line,
    boundaryByteOffset: c.boundary.source && c.boundary.source.byteOffset,
    boundaryUuid: c.boundary.uuid,
    boundaryType: c.boundary.type,
    logicalParentUuid: c.logicalParentUuid ?? null,
    boundaryParentUuid: c.boundary.parentUuid ?? null,
    metadataKind: c.metadata ? (typeof c.metadata) : null,
    summaryCount: c.summaries.length,
    summaryUuids: c.summaries.map(s => s.uuid),
    summaryLines: c.summaries.map(s => s.source && s.source.line),
    precedingUuid: c.beforeMessage ? c.beforeMessage.uuid : null,
    precedingLine: c.beforeMessage && c.beforeMessage.source ? c.beforeMessage.source.line : null,
    firstFollowingUuid: c.firstFollowingMessage ? c.firstFollowingMessage.uuid : null,
    firstFollowingAssistantUuid: c.firstFollowingAssistant ? c.firstFollowingAssistant.uuid : null,
  }));
}

__verb__("nativeCompactions", {
  name: "native-compactions",
  short: "Project mt.native() compaction records with explicit absence",
  fields: {
    params: { bind: "params" },
  },
});
