package minitracejs

import (
	"context"
	"os"
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
		const db = mt.db().sources(mt.sources().content(JSON.stringify({
			id: "empty", profile: "organic", turns: [], tool_calls: []
		}), {name: "fixture"}).build()).cache({mode: "memory"}).limits({maxRows: 10}).build();
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
			try { if (!other.execute(plan).error.includes("owns this plan")) throw Error("wrong database accepted"); }
			finally { other.close(); }
		} finally { db.close(); }
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTaggedSQLBinding(t *testing.T) {
	vm := goja.New()
	registry := new(require.Registry)
	registry.RegisterNativeModule(ModuleName, NewLoader(context.Background(), nil, "fixture", RuntimeSettings{}))
	registry.Enable(vm)
	_, err := vm.RunString(`
		const mt = require("minitrace");
		const db = mt.db().sources(mt.sources().content(JSON.stringify({
			id: "empty", profile: "organic", turns: [], tool_calls: []
		}), {name: "fixture"}).build()).cache({mode: "memory"}).limits({maxRows: 10}).build();
		function rows(plan) {
			const result = db.execute(plan);
			if (result.error || !result.complete) throw Error(JSON.stringify(result));
			return result.rows;
		}
		try {
			const label = "'?); --";
			const ordered = mt.sql` + "`" + `SELECT ${label} AS label, 2 AS n
				UNION ALL SELECT 'other', 1 ORDER BY n DESC` + "`" + `.build();
			if (rows(ordered)[0].label !== label) throw Error("binding or order changed");
			const source = mt.sql` + "`" + `SELECT session_id FROM sessions
				WHERE session_id = ${"empty"} -- retain line comment` + "`" + `.columns({session_id: mt.type.string()});
			const composed = source.select(s => ({id: s.session_id, label}))
				.where(s => s.label.eq(label)).build();
			if (rows(composed)[0].id !== "empty") throw Error("composed binding order changed");
			try {
				mt.sql` + "`" + `SELECT '${label}' AS label` + "`" + `;
				throw Error("expected tag to reject quoted interpolation");
			} catch (err) {
				if (!String(err).includes("outside quotes")) throw err;
			}
			const list = mt.sql` + "`" + `SELECT session_id FROM sessions
				WHERE instr(${"|656D707479|"}, '|' || hex(session_id) || '|') > 0` + "`" + `;
			if (rows(list.build()).length !== 1) throw Error("list binding failed");
		} finally { db.close(); }
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParametersAndFragments(t *testing.T) {
	runRelationalFixture(t, "parameters_fragments.js")
}

func TestMapRows(t *testing.T) {
	runRelationalFixture(t, "map_rows.js")
}

func runRelationalFixture(t *testing.T, fixture string) {
	t.Helper()
	source, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	registry := new(require.Registry)
	registry.RegisterNativeModule(ModuleName, NewLoader(context.Background(), nil, "fixture", RuntimeSettings{}))
	registry.Enable(vm)
	if _, err := vm.RunScript(fixture, string(source)); err != nil {
		t.Fatal(err)
	}
}
