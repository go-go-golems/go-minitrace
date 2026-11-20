function good(v) {
  let out = {argc: arguments.length, vtype: typeof v, isNull: v === null};
  try { out.string = String(v); } catch(e) { out.stringErr=String(e); }
  try { out.names = Object.getOwnPropertyNames(v); } catch(e) { out.namesErr=String(e); }
  try { out.json = JSON.stringify(v); } catch(e) { out.jsonErr=String(e); }
  try { out.source = v.source; } catch(e) { out.sourceErr=String(e); }
  try { out.idx = v['source']; } catch(e) { out.idxErr=String(e); }
  return [out];
}
__verb__("good", { name: "good", short: "Good", fields: { source: { type: "string", required: true, help: "Source" } } });
