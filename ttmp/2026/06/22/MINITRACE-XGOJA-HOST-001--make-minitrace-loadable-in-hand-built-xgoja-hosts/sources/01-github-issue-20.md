---
Title: GitHub issue 20 capture
Ticket: MINITRACE-XGOJA-HOST-001
Status: active
Topics:
    - minitrace
    - xgoja
    - architecture
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - "https://github.com/go-go-golems/go-minitrace/issues/20"
Summary: "Captured body of GitHub issue #20 for offline ticket evidence."
LastUpdated: 2026-06-22T17:15:00-04:00
WhatFor: "Use as the source issue text for MINITRACE-XGOJA-HOST-001."
WhenToUse: "When reviewing the original issue request without querying GitHub."
---

# Make `require("minitrace")` loadable in hand-built xgoja hosts (self-register module + refresh stale example)

URL: https://github.com/go-go-golems/go-minitrace/issues/20
State: OPEN
Labels: documentation, enhancement

## Summary

`require("minitrace")` cannot currently be used from a **single hand-built xgoja binary** without non-obvious registration work, even though `mt.db()` itself works perfectly once loaded. The shipped `examples/xgoja/minitrace-command-provider` is meant to demonstrate exactly this path but no longer builds against workspace `go-go-goja` HEAD.

The goal: a colleague should be able to write ~20 lines of Go that boots an xgoja runtime with `require("minitrace")` + `require("template")` + `require("fs")` together, run one JS verb that does `mt.session().File(...)` / `mt.db()` → `template` → `fs.writeFileSync`, and produce a self-contained artifact — no separate `go-minitrace` CLI and no Python/bash orchestration.

## Context / versions

- `github.com/go-go-golems/go-minitrace` **v0.2.3** (and workspace HEAD)
- `github.com/go-go-golems/go-go-goja` **v0.10.1** (published) and workspace HEAD (`v0.10.1-2-g2451b85`)
- `github.com/go-go-golems/goja-text` v0.1.1

## Problem A — `pkg/minitracejs` does not self-register into the default registry

`pkg/minitracejs/module.go` exports `NewLoader` and `const ModuleName = "minitrace"` (lines 13, 22), but there is **no `init()` that calls `modules.Register(...)`** anywhere in `pkg/minitracejs/`. Confirmed absent in both v0.2.3 and workspace HEAD.

Contrast with `goja-text`, which self-registers: `pkg/template/module.go:102-103`
```go
func init() {
    modules.Register(&module{})
}
```

Consequence: `engine.NewRuntimeFactoryBuilder().Build()` (the plain builder that loads the default-registry modules `fs`, `console`, `path`, `template`, `extract`, …) does **not** load `require("minitrace")`. A probe in such a runtime:

```js
require("fs");       // ok
require("template"); // ok
require("minitrace"); // -> GoError: Invalid module
```

The minitrace module is only reachable through its provider package (`pkg/minitracejs/provider` via `providerapi.ProviderRegistry`), which is fine for the generated `xgoja` host but leaves hand-written hosts with no ergonomic path.

## Problem B — explicit `WithModules(...)` drops the default registry

The natural workaround for A is to register minitrace explicitly:

```go
factory, _ := engine.NewRuntimeFactoryBuilder().
    WithModules(engine.NativeModuleRegistrar{
        ModuleName: minitracejs.ModuleName,
        Loader:     minitracejs.NewLoader(ctx, nil, "app", minitracejs.RuntimeSettings{}),
    }).
    Build()
```

This loads `minitrace`, but then **`fs`, `console`, `template`, etc. disappear** (`require("fs")` → `Invalid module`). This is by design per `go-go-goja@v0.10.1/pkg/engine/factory.go:138-141`:

> A plain `NewRuntimeFactoryBuilder().Build()` preserves the historical default of exposing all default-registry modules. Calling `UseModuleMiddleware` narrows or transforms that selection; **explicit `WithModules(...)` remains explicit and does not auto-append the default registry.**

So a hand-written host is stuck: plain builder omits minitrace; explicit `WithModules` omits the default modules. The only documented escape hatch is `UseModuleMiddleware(Add("minitrace"))`, which is non-obvious, or reaching into the provider machinery.

## Problem C — `examples/xgoja/minitrace-command-provider` smoke no longer builds

The example (the canonical reference for the pattern) is stale against the in-flight xgoja/v2 work in `go-go-goja`:

1. The shipped `examples/xgoja/minitrace-command-provider/xgoja.yaml` is **legacy format**. `xgoja doctor` rejects it: *"appears to be a legacy xgoja spec; run `xgoja migrate-spec`"*.
2. After `xgoja migrate-spec` to v2, `xgoja build` generates a `main.go` that fails to compile against workspace `go-go-goja` HEAD:
   ```
   ./main.go:66:67: unknown field RuntimePlanJSON in struct literal of type app.Options
   ./main.go:74:31: undefined: app.RuntimePlan
   ./main.go:75:22: undefined: app.RuntimePlan
   ```
   `app.RuntimePlan` exists in workspace go-go-goja at `pkg/xgoja/app/runtime_plan.go`, but the generator emits it at a different import path — a generator/runtime skew from the v2 refactor.

`make smoke` therefore fails at the `build` step.

## Reproduction

**Problem A + B probe** (hand-written host): build a runtime with minitrace registered via `NativeModuleRegistrar`, then from JS `require` each of `minitrace`, `fs`, `template`. You'll see minitrace load but fs/template fail; with the plain builder, minitrace fails and fs/template load.

**Problem C**: from the workspace,
```
cd examples/xgoja/minitrace-command-provider
go run <workspace>/go-go-goja/cmd/xgoja doctor -f ./xgoja.yaml        # legacy
go run <workspace>/go-go-goja/cmd/xgoja migrate-spec -f ./xgoja.yaml --out xgoja.v2.yaml
go run <workspace>/go-go-goja/cmd/xgoja build -f ./xgoja.v2.yaml --output ./dist/x --keep-work   # go build fails
```

## Suggested fixes (pick what fits the design)

1. **Self-register `minitracejs` into `modules.DefaultRegistry` via `init()`**, mirroring `goja-text` (`template`, `extract`). This is the lowest-friction fix: plain `engine.NewRuntimeFactoryBuilder().Build()` would then load `minitrace` alongside `fs`/`console`/`template`, and Problems A and B both disappear for hand-written hosts. The module already builds its own SQLite via `mt.db()` so it doesn't need host services for the common case.
2. If self-registration is undesirable (e.g. to keep the provider path the single source of truth), expose and document a small helper like `minitracejs.RegisterDefault()` and reference it from a "hand-written host" section in the README.
3. Refresh `examples/xgoja/minitrace-command-provider`: migrate `xgoja.yaml` to v2, confirm `make smoke` builds against current `go-go-goja`, and coordinate with whoever lands the xgoja/v2 generator changes (the `RuntimePlan` import-path mismatch is likely a go-go-goja generator fix, not a go-minitrace fix — but the example should track it).

(1) and (3) together would let someone write a single-binary transcript reporter — the concrete use case that surfaced this.

## Acceptance criteria

- [ ] `require("minitrace")` resolves in a runtime built with the plain `engine.NewRuntimeFactoryBuilder().Build()`, alongside `fs`/`console`/`template` (no `Invalid module` errors).
- [ ] `make smoke` in `examples/xgoja/minitrace-command-provider` passes against workspace `go-go-goja` HEAD.
- [ ] A short README section (or example) showing a hand-written Go host that loads minitrace + template + fs and runs one JS verb end-to-end.

## Related

- `go-go-goja` `pkg/engine/factory.go` (`WithModules` explicit-only semantics)
- `go-go-goja` xgoja/v2 generator (the `RuntimePlan` import-path skew in Problem C)
- Worked example of the desired end state: a single binary doing convert (`mt.session().File`) → query (`mt.db()`) → render (`require("template")`) → write (`require("fs")`), replacing a Python + bash + `go-minitrace` CLI pipeline.

