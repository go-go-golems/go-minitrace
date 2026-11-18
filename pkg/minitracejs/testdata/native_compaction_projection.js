// `inspection` is the result of mt.native().file(path).inspect().
// Preserve absence explicitly; physical coordinates do not certify replay.
inspection.compactions.map(c => ({
  boundary: c.boundary.source,
  boundaryUuid: c.boundary.uuid,
  logicalParentUuid: c.logicalParentUuid ?? null,
  metadata: c.metadata ?? null,
  summaries: c.summaries.map(s => ({uuid: s.uuid, source: s.source})),
  precedingUuid: c.beforeMessage?.uuid ?? null,
  firstFollowingUuid: c.firstFollowingMessage?.uuid ?? null,
  firstFollowingAssistantUuid: c.firstFollowingAssistant?.uuid ?? null,
}));
