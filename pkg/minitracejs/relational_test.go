package minitracejs

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func TestRelationalBindingThroughRequire(t *testing.T) {
	vm := goja.New()
	registry := new(require.Registry)
	registry.RegisterNativeModule(ModuleName, NewLoader(context.Background(), nil, "fixture", RuntimeSettings{}))
	registry.Enable(vm)
	_, err := vm.RunString(`
		const mt = require("minitrace");
		const db = mt.db().Content(JSON.stringify({
			id: "empty", profile: "organic", turns: [], tool_calls: []
		})).Build();
		try {
			const sessions = mt.from(db.table("sessions"));
			const tools = mt.from(db.table("tool_calls"));
			let calls = 0;
			const counts = sessions.use(q => q.select(s => {
				calls++;
				return {
					id: s.session_id,
					count: tools.where(t => t.session_id.eq(s.session_id)).count(),
				};
			})).orderBy(s => [s.id.asc(), s.count.desc()]);
			const plan = counts.build();
			counts.build();
			if (calls !== 1) throw Error("callbacks ran again during build");
			const result = db.execute(plan);
			if (result.error || !result.complete || result.rows.length !== 1 ||
			    result.rows[0].id !== "empty" || result.rows[0].count !== 0) {
				throw Error(JSON.stringify(result));
			}
			function rejects(fn, expected) {
				try { fn(); } catch (err) {
					if (String(err).includes(expected)) return;
					throw err;
				}
				throw Error("expected error containing " + expected);
			}
			rejects(() => sessions.where(s => s.typo.eq("x")), "available:");
			rejects(() => sessions.where(s => s.session_id === "x"), "instead of ===");
			rejects(() => sessions.select(async s => ({id: s.session_id})), "synchronously");
			const other = mt.db().Content(JSON.stringify({
				id: "other", profile: "organic", turns: [], tool_calls: []
			})).Build();
			try { rejects(() => other.execute(plan), "owns this plan"); }
			finally { other.close(); }
		} finally { db.close(); }
	`)
	if err != nil {
		t.Fatal(err)
	}
}
