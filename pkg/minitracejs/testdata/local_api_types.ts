import {db, sources, from, sql, type, param, fragment} from "../minitrace";

const handle = db().sources(sources().runtimeArchives().build()).build();
const sessions = from(handle.table("sessions"));
const tools = from(handle.table("tool_calls"));
const minimum = param.integer("minimum").range(0, 100).default(0).ui({widget: "slider"});
const projection = sessions.select(s => ({
  id: s.session_id,
  count: tools.where(t => t.session_id.eq(s.session_id)).count(),
})).where(r => r.count.gte(minimum));
const mapped = projection.mapRows(row => ({label: String(row.count)}), {
  columns: {label: type.string()},
});
const result = handle.execute(mapped.build(), {params: {minimum: 1}});
const label: string = result.rows[0].label;
const within = fragment({name: "within", target: "query", methods: {
  min(q, limit: number) { return q.where(r => r.count.gte(limit)); },
}});
projection.extend(within).min(5).build();
sql`SELECT ${minimum} AS n`.columns({n: type.integer()}).where(r => r.n.gt(1)).build();
// @ts-expect-error A mapped builder only supports build.
mapped.where(() => true);
// @ts-expect-error Only built query plans can execute.
handle.execute("SELECT 1");
