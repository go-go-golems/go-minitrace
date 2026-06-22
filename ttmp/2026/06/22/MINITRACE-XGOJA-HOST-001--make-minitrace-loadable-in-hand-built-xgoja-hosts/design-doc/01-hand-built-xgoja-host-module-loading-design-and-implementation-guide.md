---
Title: Hand-built xgoja host module loading design and implementation guide
Ticket: MINITRACE-XGOJA-HOST-001
Status: active
Topics:
    - minitrace
    - xgoja
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/pkg/engine/factory.go
      Note: Defines default-registry versus explicit WithModules behavior
    - Path: go-minitrace/examples/xgoja/minitrace-command-provider/Makefile
      Note: Smoke target and xgoja command wiring to refresh
    - Path: go-minitrace/examples/xgoja/minitrace-command-provider/xgoja.yaml
      Note: Legacy spec that currently breaks make smoke
    - Path: go-minitrace/pkg/minitracejs/module.go
      Note: Owns ModuleName
    - Path: go-minitrace/pkg/minitracejs/provider/provider.go
      Note: Existing xgoja provider path that must remain compatible
    - Path: go-minitrace/pkg/minitracejs/typescript.go
      Note: Existing TypeScript descriptor to forward from the new native module adapter
    - Path: goja-text/pkg/template/module.go
      Note: Sibling self-registering module pattern to copy
ExternalSources:
    - https://github.com/go-go-golems/go-minitrace/issues/20
Summary: Design and intern-oriented implementation guide for making require("minitrace") available in hand-built go-go-goja/xgoja hosts while preserving generated provider behavior.
LastUpdated: 2026-06-22T17:15:00-04:00
WhatFor: 'Use this when implementing issue #20 or onboarding an engineer to minitracejs, go-go-goja module registration, and the xgoja command-provider example.'
WhenToUse: Before changing pkg/minitracejs module loading, refreshing examples/xgoja/minitrace-command-provider, or documenting hand-built xgoja hosts.
---


# Hand-built xgoja host module loading design and implementation guide

## Executive summary

GitHub issue #20 asks for `require("minitrace")` to work in a small hand-written Go program that creates a go-go-goja runtime and uses `minitrace`, `template`, and `fs` together. The request makes sense: the `minitracejs` package already contains the JavaScript-facing loader, builders, runtime settings, and TypeScript descriptor, but it is not registered in the go-go-goja default module registry. Today a generated xgoja provider can expose it, while a hand-built host must know a non-obvious explicit registration path.

The recommended implementation is to add a normal default-registry adapter in `pkg/minitracejs`: implement `modules.NativeModule`, optionally implement `modules.TypeScriptDeclarer`, delegate `Loader` to the existing `NewLoader(context.Background(), nil, "", RuntimeSettings{})`, and call `modules.Register(...)` from `init()`. This mirrors `goja-text/pkg/template`, keeps the provider path intact, and makes the package behave like other reusable `require()` modules.

The second part of the work is to refresh `examples/xgoja/minitrace-command-provider`. Its checked-in `xgoja.yaml` is still legacy-format; current `make smoke` fails at `xgoja doctor`. A temporary migration to v2 and build against the current workspace `go-go-goja` succeeds, so the example likely needs a spec migration and Makefile/doc update more than a deeper runtime fix.

## Problem statement and scope

The concrete desired user story is:

1. A colleague writes one small Go binary.
2. That binary imports the modules it needs so their `init()` functions run.
3. The binary builds a runtime with `engine.NewRuntimeFactoryBuilder().Build()`.
4. JavaScript code can call:
   - `require("minitrace")` for transcript/session loading and query helpers;
   - `require("template")` for rendering;
   - `require("fs")` for writing an artifact.
5. The binary produces a self-contained report without shelling out to the `go-minitrace` CLI or relying on a generated xgoja host.

This design covers the go-minitrace-owned work required for that story:

- Register `pkg/minitracejs` as a reusable go-go-goja native module.
- Document the import/linking requirement for hand-written hosts.
- Add tests that prove `require("minitrace")` resolves in the plain runtime builder path.
- Refresh the stale xgoja command-provider example so interns have a working generated-host reference.

This design does not propose changing go-go-goja `WithModules(...)` semantics. Those semantics are deliberate and documented in `go-go-goja/pkg/engine/factory.go`; changing them would be a cross-repository behavior change with wider compatibility risk.

## Glossary for a new intern

- **goja**: The JavaScript interpreter used by go-go-goja.
- **goja_nodejs/require**: The CommonJS-like `require()` layer. Go code registers native loaders by module name; JavaScript later calls `require("name")`.
- **go-go-goja engine**: The runtime factory and lifecycle layer in `github.com/go-go-golems/go-go-goja/pkg/engine`.
- **Default registry**: A global registry in `github.com/go-go-golems/go-go-goja/modules`. Packages register themselves there from `init()`. A plain engine builder installs those registered modules into each runtime.
- **Native module**: A Go object implementing `modules.NativeModule` with `Name()`, `Doc()`, and `Loader(...)`.
- **Provider module**: An xgoja provider entry. This is a generated-host mechanism, not the same as the default registry mechanism.
- **Hand-built host**: A manually written Go program that creates an engine runtime directly instead of using `xgoja build` output.
- **Runtime settings**: Minitrace settings such as archive globs and DB paths. Provider-generated hosts can pass these from command context; default-registry hand-written hosts should start with empty settings and configure sources from JavaScript.

## Current-state architecture

### Big-picture module-loading paths

There are two module-loading paths that matter.

```text
Path A: plain hand-built host today

Go binary imports go-go-goja engine
        |
        v
engine.NewRuntimeFactoryBuilder().Build()
        |
        v
DefaultRegistry modules are selected
        |
        v
require("fs") works, require("minitrace") fails
```

```text
Path B: generated xgoja provider host today

xgoja spec imports pkg/minitracejs/provider
        |
        v
provider.Register(...) declares providerapi.Module{Name: "minitrace"}
        |
        v
generated host registers provider-created loader explicitly
        |
        v
require("minitrace") works inside generated command/provider runtime
```

The problem is not that `minitracejs` lacks a loader. The problem is that only Path B wires that loader into a runtime automatically.

### Evidence: minitracejs already owns the JS-facing loader

`/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/module.go` defines the public module name and the loader:

- `ModuleName = "minitrace"` at line 13.
- `RuntimeSettings` at lines 15-20.
- `NewLoader(ctx, conn, commandName, runtimeSettings)` at lines 22-71.
- The loader exports builder entrypoints: `importer`, `db`, `sources`, `importPolicy`, `cache`, `limits`, `query`, `view`, and `session` at lines 29-55.
- The loader also exports runtime metadata and SQL quoting helpers at lines 57-69.

That file does not import `github.com/go-go-golems/go-go-goja/modules`, does not define a `modules.NativeModule` adapter, and reaches EOF at line 114 without an `init()` registration. The absence is important because go-go-goja default modules are discovered through package `init()` registration.

### Evidence: provider-generated hosts already know how to construct the loader

`/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/provider/provider.go` contains the xgoja provider wrapper:

- `PackageID = "go-minitrace"` at line 19.
- Optional host services for `Conn`, `RuntimeSettings`, and `CommandName` at lines 21-25.
- `Register(registry *providerapi.ProviderRegistry)` at line 55.
- Provider module metadata with `Name: minitracejs.ModuleName` and `DefaultAs: minitracejs.ModuleName` at lines 57-60.
- The provider module factory reads host services if present and returns `minitracejs.NewLoader(...)` at lines 62-75.
- The same provider package also exposes a `queries` command set at lines 78-84.

The provider is useful for generated xgoja binaries because it can receive command-specific context. It is too much ceremony for a small hand-built host that only needs the normal `mt.session().File(...)` / `mt.db()` API.

### Evidence: go-go-goja default modules register by `init()`

The default-registry contract is in `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/common.go`:

- `NativeModule` requires `Name()`, `Doc()`, and `Loader(*goja.Runtime, *goja.Object)` at lines 28-32.
- The registry comment says `Register` is typically called from a module package `init()` at lines 48-50.
- The global `DefaultRegistry` is created at line 93.
- Package-level `modules.Register(m)` appends to that registry at lines 95-98.
- `modules.GetModule` and `modules.ListDefaultModules` expose registry contents at lines 100-108.

The engine imports built-in module packages only for their side effects in `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/engine/runtime.go` lines 16-29. The comment there is the key mental model: blank imports ensure module `init()` functions run so those modules can register themselves in `modules.DefaultRegistry`.

A third-party module such as `go-minitrace/pkg/minitracejs` cannot be discovered unless the final Go binary imports it somewhere. Self-registration does not mean dynamic package discovery; it means "if this package is linked, its init registers the module."

### Evidence: plain builder vs explicit builder behavior

`/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/engine/factory.go` documents the builder behavior:

- `WithModules(...)` appends explicit runtime-aware module registrations at lines 77-81.
- `UseModuleMiddleware(...)` is the preferred way to control which default-registry modules are loaded at lines 84-97.
- During `Build()`, the code comments say a plain builder exposes all default-registry modules, while explicit `WithModules(...)` remains explicit and does not auto-append the default registry at lines 137-151.
- During runtime creation, data-only defaults are installed at lines 245-249, then explicit/factory modules are installed at lines 251-256, then `reg.Enable(vm)` installs `require` at line 258.

This explains the confusing behavior reported in the issue: adding minitrace explicitly does not imply that `fs`, `template`, or other default modules remain present.

### Evidence: `MiddlewareAdd("minitrace")` cannot add an unregistered module

`/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/engine/module_middleware.go` says `MiddlewareAdd` only appends named modules "if they exist in the available set" at lines 68-83. The available set comes from `allRegisteredModuleNames()`, which reads `modules.ListDefaultModules()` at lines 111-119.

Therefore `UseModuleMiddleware(engine.MiddlewareAdd("minitrace"))` is not an escape hatch until `minitrace` is registered in the default registry.

### Evidence: goja-text is the pattern to copy

`/home/manuel/workspaces/2026-06-07/club-meetup-site/goja-text/pkg/template/module.go` is a proven sibling pattern:

- `module` implements `modules.NativeModule` and `modules.TypeScriptDeclarer` at lines 11-14.
- `Name()` returns `"template"` at line 16.
- `Loader(...)` populates exports at lines 35-54.
- `init()` calls `modules.Register(&module{})` at lines 102-103.

This is the exact shape that would make `pkg/minitracejs` work as a default-registry module.

### Evidence: the example is stale, but the generator skew appears fixed locally

The checked-in example files are still legacy-style:

- `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider/xgoja.yaml` uses the old top-level `target`, `packages`, `modules`, `commands`, and `commandProviders` shape at lines 1-31.
- `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider/Makefile` runs `xgoja doctor`, `list-modules`, and `build` against that file at lines 10-19.
- The example README tells users to run `make smoke` at lines 11-15.

The ticket-local script `scripts/02-check-xgoja-example.sh` captured current behavior in `sources/03-xgoja-example-check-output.txt`:

```text
Error: .../xgoja.yaml appears to be a legacy xgoja spec; run xgoja migrate-spec ...
make: *** [Makefile:13: doctor] Error 1
```

The same script then temporarily migrated the spec to v2 and built with the current workspace `go-go-goja`; that build succeeded:

```text
validated xgoja/v2 plan .../xgoja.v2.tmp.yaml
xgoja build ok: .../dist/x.tmp
```

This means the issue's older `RuntimePlan` compile error may already be fixed in current `go-go-goja`; the stale example is still real and should be refreshed.

## Reproduction results captured for this ticket

The ticket script `scripts/01-probe-module-loading.sh` builds three runtime factories and asks JavaScript whether it can require `minitrace`, `fs`, and `template`. It imports `goja-text/pkg/template` only to make the `template` module available to the default registry for the probe.

Captured output in `sources/02-module-loading-probe-output.txt`:

```text
middleware-add={"minitrace":"GoError: Invalid module","fs":true,"template":true} err=<nil>
plain={"minitrace":"GoError: Invalid module","fs":true,"template":true} err=<nil>
explicit={"minitrace":true,"fs":"GoError: Invalid module","template":"GoError: Invalid module"} err=<nil>
```

Interpretation:

- A plain builder has the default modules that are registered and linked, but not `minitrace`.
- `MiddlewareAdd("minitrace")` cannot add a module that is not registered.
- Explicit `WithModules(minitrace)` loads minitrace but opts out of the full default registry selection.

## Gap analysis

### Gap 1: no default-registry adapter in `pkg/minitracejs`

The minitrace loader exists, but it is only exposed as a raw `require.ModuleLoader` factory and through the provider package. A default-registry adapter is missing.

### Gap 2: no hand-built host documentation

The README does not currently show a small host that imports/link-registers minitrace, imports `goja-text` for template, uses `fs`, and runs one JavaScript verb end-to-end. Without that example, users infer the provider-generated path is required.

### Gap 3: stale generated-host example

The xgoja command-provider example is valuable because it demonstrates command providers, mounted queries, and file-writing report generation. It currently fails before build because the YAML is legacy.

### Gap 4: tests do not pin the default-registry behavior

There is no integration test in `pkg/minitracejs` that creates a plain go-go-goja runtime and proves `require("minitrace")` resolves. Without that test, future refactors can accidentally regress the hand-built host path.

## Proposed architecture

### Target runtime-loading flow after the fix

```text
Hand-written Go host
        |
        | imports:
        |   _ "github.com/go-go-golems/go-minitrace/pkg/minitracejs"
        |   _ "github.com/go-go-golems/goja-text/pkg/template"       (if template is needed)
        v
package init() calls modules.Register(...)
        |
        v
engine.NewRuntimeFactoryBuilder().Build()
        |
        v
factory selects all linked default-registry modules
        |
        v
JavaScript:
  const mt = require("minitrace")
  const template = require("template")
  const fs = require("fs")
```

### New minitrace default module adapter

Add a small file such as `/pkg/minitracejs/default_module.go`:

```go
package minitracejs

import (
    "context"

    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/modules"
)

type module struct{}

var _ modules.NativeModule = (*module)(nil)
var _ modules.TypeScriptDeclarer = (*module)(nil)

func (*module) Name() string { return ModuleName }

func (*module) Doc() string {
    return `The minitrace module loads transcript archives, opens queryable DB handles, and builds session/view/query helpers for JavaScript.`
}

func (*module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    NewLoader(context.Background(), nil, "", RuntimeSettings{})(vm, moduleObj)
}

func (*module) TypeScriptModule() *spec.Module {
    return TypeScriptModule()
}

func init() {
    modules.Register(&module{})
}
```

The pseudocode above omits the `spec` import in the snippet header for brevity. The real file must import `github.com/go-go-golems/go-go-goja/pkg/tsgen/spec` if it implements `TypeScriptDeclarer` directly.

The loader should delegate to `NewLoader` instead of duplicating exports. That keeps the provider path and default-registry path aligned.

### Why `context.Background()` and empty runtime settings are acceptable defaults

The default-registry path is for hand-built hosts that configure work from JavaScript. Empty `RuntimeSettings` still supports the core APIs:

- `mt.db().File(path).Build()`
- `mt.db().Content(content, name).Build()`
- `mt.session().File(path).Open()`
- `mt.sources().File(path).Build()`

The default settings only make runtime-provided features empty:

- `runtime.archiveGlob` is empty.
- `RuntimeArchives()` reports that no runtime archive glob is configured.
- `runtime.commandName`, `runtime.dbPath`, and `runtime.tableName` are empty strings.

Generated xgoja provider hosts can continue using `pkg/minitracejs/provider`, which passes command-scoped context and settings into `NewLoader`.

### API sketch: hand-built Go host

```go
package main

import (
    "context"
    "fmt"

    "github.com/go-go-golems/go-go-goja/pkg/engine"

    // These blank imports are important: they link the packages and run init().
    _ "github.com/go-go-golems/go-minitrace/pkg/minitracejs"
    _ "github.com/go-go-golems/goja-text/pkg/template"
)

func main() {
    factory, err := engine.NewRuntimeFactoryBuilder().Build()
    if err != nil {
        panic(err)
    }
    rt, err := factory.NewRuntime(
        engine.WithStartupContext(context.Background()),
        engine.WithLifetimeContext(context.Background()),
    )
    if err != nil {
        panic(err)
    }
    defer rt.Close(context.Background())

    _, err = rt.Owner.Call(context.Background(), "render-report", func(ctx context.Context, vm *goja.Runtime) (any, error) {
        return vm.RunString(reportJS)
    })
    if err != nil {
        panic(err)
    }
    fmt.Println("report written")
}
```

The key teaching point for interns is that Go packages are not dynamically loaded by module string. `require("minitrace")` succeeds because the Go binary imported `pkg/minitracejs`, which ran `init()`, which registered the module in the default registry.

### API sketch: JavaScript report verb

```js
const mt = require("minitrace");
const template = require("template");
const fs = require("fs");

const session = mt.session()
  .File("./session.minitrace.json")
  .InteractiveCache("./.cache/minitrace")
  .Open();

try {
  const summary = session.summary();
  const rows = session.query(`
    select role, count(*) as n
    from messages
    group by role
    order by n desc
  `);

  const markdown = template.renderText(`# {{ .Title }}\n\nSession: {{ .SessionID }}\n`, {
    Title: "Minitrace report",
    SessionID: session.id(),
    Summary: summary,
    Rows: rows,
  }).Text;

  fs.writeFileSync("./dist/report.md", markdown);
} finally {
  session.close();
}
```

This shows the desired end state: convert/open/query/render/write in one JS verb inside one Go binary.

## Decision records

### Decision: self-register `pkg/minitracejs` as a default-registry module

- **Context:** Hand-built hosts can use default modules such as `fs` and linked third-party modules such as `template`, but `minitrace` is missing because `pkg/minitracejs` does not register itself.
- **Options considered:** Provider-only usage; an exported `RegisterDefault()` helper; changing go-go-goja `WithModules` to include defaults; adding a default-registry adapter with `init()`.
- **Decision:** Add a `modules.NativeModule` adapter in `pkg/minitracejs` and register it from `init()`.
- **Rationale:** This matches the go-go-goja module contract and the goja-text pattern. It keeps generated provider behavior unchanged and makes the common hand-built host path simple.
- **Consequences:** Any binary that imports `pkg/minitracejs` will make `minitrace` available when the default registry is selected. Hosts that need stricter module selection should use `UseModuleMiddleware(MiddlewareOnly(...))` or avoid importing this package.
- **Status:** proposed.

### Decision: keep provider runtime settings separate from default-registry settings

- **Context:** The provider path can receive host services and command runtime settings. The default registry path cannot assume those services exist.
- **Options considered:** Global mutable runtime settings; a default loader with empty settings; forcing hand-built hosts to use explicit `WithModules` when settings are needed.
- **Decision:** The self-registered module uses empty settings and `context.Background()`. Advanced hosts that need runtime settings can still register an explicit loader.
- **Rationale:** The most important hand-built APIs accept sources directly from JavaScript. Empty runtime settings are predictable and avoid hidden global state.
- **Consequences:** `RuntimeArchives()` is not useful unless runtime archive globs are configured through a provider or explicit loader. Documentation must say this clearly.
- **Status:** proposed.

### Decision: do not change go-go-goja `WithModules` behavior for this ticket

- **Context:** Explicit `WithModules(...)` dropping default modules feels surprising in this use case, but it is documented behavior in go-go-goja.
- **Options considered:** Change go-go-goja to always append defaults; add a go-go-goja helper to append default modules; solve go-minitrace by self-registration.
- **Decision:** Do not change go-go-goja in this ticket.
- **Rationale:** Changing `WithModules` would affect all hosts and tests that rely on explicit module sets. Self-registration fixes the specific missing module without changing engine semantics.
- **Consequences:** Documentation should teach `UseModuleMiddleware` for default-registry selection and explicit `WithModules` for fully controlled runtimes.
- **Status:** proposed.

### Decision: migrate the example instead of deleting it

- **Context:** The xgoja command-provider example currently fails because its spec is legacy, but temporary migration and build succeed locally.
- **Options considered:** Delete the example; leave it stale but document the hand-built host path; migrate it to v2 and keep it as a smoke test.
- **Decision:** Migrate the example to v2 and keep `make smoke` as an acceptance test.
- **Rationale:** The example exercises a different but important path: generated xgoja provider modules and command providers. It is useful for interns learning the whole system.
- **Consequences:** The Makefile and README must align with current xgoja commands and workspace replacement behavior.
- **Status:** proposed.

## Implementation guide for an intern

### Phase 1: add the native module adapter

Create `pkg/minitracejs/default_module.go`.

Checklist:

- Import `context`, `goja`, `modules`, and `tsgen/spec` if implementing TypeScript descriptor forwarding.
- Define an unexported `module` type.
- Add compile-time assertions:
  - `var _ modules.NativeModule = (*module)(nil)`
  - `var _ modules.TypeScriptDeclarer = (*module)(nil)`
- `Name()` returns `ModuleName`.
- `Doc()` explains the module in 1-2 paragraphs and lists the main exports.
- `Loader(...)` delegates to `NewLoader(context.Background(), nil, "", RuntimeSettings{})`.
- `TypeScriptModule()` returns the existing package-level descriptor from `typescript.go`.
- `init()` calls `modules.Register(&module{})`.

Pseudocode:

```go
func (*module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
    loader := NewLoader(context.Background(), nil, "", RuntimeSettings{})
    loader(vm, moduleObj)
}
```

Do not duplicate export wiring from `module.go`; duplication would make provider and default module behavior drift.

### Phase 2: add a runtime integration test

Create `pkg/minitracejs/default_module_test.go`.

The test should prove the acceptance criterion owned by go-minitrace: importing `pkg/minitracejs` makes `require("minitrace")` resolve in a plain builder alongside go-go-goja default modules such as `fs`.

Pseudocode:

```go
package minitracejs_test

import (
    "context"
    "testing"

    "github.com/dop251/goja"
    "github.com/go-go-golems/go-go-goja/pkg/engine"
    _ "github.com/go-go-golems/go-minitrace/pkg/minitracejs"
)

func TestDefaultBuilderCanRequireMinitrace(t *testing.T) {
    factory, err := engine.NewRuntimeFactoryBuilder().Build()
    if err != nil { t.Fatal(err) }

    rt, err := factory.NewRuntime(
        engine.WithStartupContext(context.Background()),
        engine.WithLifetimeContext(context.Background()),
    )
    if err != nil { t.Fatal(err) }
    defer rt.Close(context.Background())

    ret, err := rt.Owner.Call(context.Background(), "require-minitrace", func(_ context.Context, vm *goja.Runtime) (any, error) {
        value, runErr := vm.RunString(`
            const mt = require("minitrace");
            const fs = require("fs");
            JSON.stringify({
              hasSession: typeof mt.session === "function",
              hasDB: typeof mt.db === "function",
              hasFS: typeof fs === "object" || typeof fs.readFileSync === "function"
            });
        `)
        if runErr != nil { return nil, runErr }
        return value.String(), nil
    })
    if err != nil { t.Fatal(err) }
    if ret != `{"hasSession":true,"hasDB":true,"hasFS":true}` {
        t.Fatalf("unexpected require result: %v", ret)
    }
}
```

Avoid adding a `goja-text` dependency to go-minitrace only for this test unless maintainers explicitly want that cross-repository dependency. The README can show `template` usage as a consumer-level example.

### Phase 3: document the hand-built host path

Update `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/README.md` with a section such as "Embedding minitrace in a hand-built go-go-goja host".

The section should teach four facts:

1. Importing `pkg/minitracejs` for side effects links and registers `require("minitrace")`.
2. Importing `goja-text/pkg/template` for side effects links and registers `require("template")`.
3. `fs` is provided by go-go-goja when the default registry is enabled.
4. If a host uses explicit `WithModules(...)`, it is responsible for registering every module it needs.

Suggested wording for the README prose:

A plain `engine.NewRuntimeFactoryBuilder().Build()` exposes modules that are linked into the Go binary and registered with `modules.DefaultRegistry`. For minitrace, add this Go import:

```go
import _ "github.com/go-go-golems/go-minitrace/pkg/minitracejs"
```

Then JavaScript can call:

```js
const mt = require("minitrace");
const session = mt.session().File("session.minitrace.json").Open();
```

### Phase 4: refresh the xgoja example

From `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider`:

1. Run migration with the current xgoja tool:

   ```bash
   cd /home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja
   GOWORK=off go run ./cmd/xgoja migrate-spec \
     -f ../go-minitrace/examples/xgoja/minitrace-command-provider/xgoja.yaml \
     --out ../go-minitrace/examples/xgoja/minitrace-command-provider/xgoja.v2.yaml
   ```

2. Decide whether to replace `xgoja.yaml` in place or keep `xgoja.v2.yaml` and point the Makefile at it. Prefer one canonical file to reduce confusion.
3. Update `Makefile` targets to use the v2 spec.
4. Run:

   ```bash
   cd /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider
   make smoke
   ```

5. If `make smoke` fails after build, inspect the generated command invocation and the local query command at `queries/reports/markdown-summary.js`.

### Phase 5: update issue notes and acceptance checklist

After implementation, comment on issue #20 or update the PR description with:

- `require("minitrace")` works in a plain builder when the package is imported.
- Explicit `WithModules(...)` remains explicit by design.
- The xgoja example has been migrated and smoke-tested.
- Any remaining limitation around runtime settings or `RuntimeArchives()`.

## Testing and validation strategy

Run tests in layers:

1. **Focused module tests**

   ```bash
   cd /home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace
   go test ./pkg/minitracejs -count=1
   ```

2. **Repository tests**

   ```bash
   go test ./... -count=1
   ```

3. **Isolated module mode**

   ```bash
   GOWORK=off go test ./... -count=1
   ```

   This matters because `go-minitrace/go.mod` currently requires `github.com/go-go-golems/go-go-goja v0.8.3`, while the workspace has a newer `go-go-goja` checkout. If the new adapter depends on APIs only present in workspace HEAD, bump the go-go-goja dependency or adjust the implementation.

4. **Hand-built host probe**

   Re-run this ticket's script after implementation:

   ```bash
   ./ttmp/2026/06/22/MINITRACE-XGOJA-HOST-001--make-minitrace-loadable-in-hand-built-xgoja-hosts/scripts/01-probe-module-loading.sh
   ```

   Expected after self-registration:

   ```text
   plain={"minitrace":true,"fs":true,"template":true} err=<nil>
   middleware-add={"minitrace":true,"fs":true,"template":true} err=<nil>
   ```

   The exact `explicit=` result should still show only explicitly registered modules unless the probe is changed.

5. **xgoja example smoke**

   ```bash
   cd examples/xgoja/minitrace-command-provider
   make smoke
   ```

## Risks and review points

### Risk: accidental duplicate module registration in generated hosts

Because `pkg/minitracejs/provider` imports `pkg/minitracejs`, adding an `init()` registration means generated provider hosts will link a default-registry entry too. This should not matter for the current provider path because explicit modules are used and the full default registry is not auto-appended when `WithModules(...)` is present. Still, reviewers should inspect generated xgoja runtime construction if the migrated example uses module middleware that selects all defaults plus explicit provider modules.

### Risk: confusing runtime settings semantics

The default-registry module should use empty runtime settings. If documentation implies `RuntimeArchives()` works without host-provided archive globs, users will get confusing errors. The README should explicitly say direct `File`, `Files`, `Content`, `Dir`, and `Glob` sources are the default hand-built-host style.

### Risk: go.mod / workspace skew

The local workspace `go-go-goja` is newer than the version required in `go-minitrace/go.mod`. Always validate `GOWORK=off` before considering the change release-ready.

### Risk: expanding default runtime surface

A binary that imports `pkg/minitracejs` and uses the plain default builder will expose minitrace to JavaScript. That is the desired behavior for this ticket. For untrusted JavaScript, hosts should use explicit module middleware and avoid importing host-capability modules that should not be exposed.

## Alternatives considered

### Alternative 1: keep provider-only usage

This would preserve the current architecture but fail the hand-built-host user story. It also keeps the example as the only ergonomic path, and that example is currently stale.

### Alternative 2: add `minitracejs.RegisterDefault()` but no `init()`

This is explicit and avoids side effects. It is also less consistent with go-go-goja modules and still requires users to know a special minitrace registration function. If maintainers strongly dislike `init()` registration for third-party modules, this is the fallback, but the README must be extremely clear.

### Alternative 3: change go-go-goja `WithModules(...)` to append defaults

This would fix the natural workaround but risks breaking hosts that rely on explicit minimal module sets. It should not be done as part of this go-minitrace issue.

### Alternative 4: put hand-built hosts through providerapi

A hand-built host could construct a provider registry, resolve the provider module, and install the loader. That path is too complex for the stated goal of a roughly 20-line embedding example.

## File reference map

| File | Why it matters |
|---|---|
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/module.go` | Owns `ModuleName`, `RuntimeSettings`, and `NewLoader`; this is where exports are defined. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/typescript.go` | Owns the TypeScript descriptor that should be forwarded by the default module adapter. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/provider/provider.go` | Existing generated xgoja provider path; should keep working unchanged. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/db_builder.go` | Shows the `DBBuilder` and `DBHandle` APIs used by JavaScript via `mt.db()`. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/pkg/minitracejs/query_view_session.go` | Shows the `SessionBuilder` and `SessionHandle` APIs used by JavaScript via `mt.session()`. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider/xgoja.yaml` | Legacy spec that must be migrated. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-minitrace/examples/xgoja/minitrace-command-provider/Makefile` | Failing smoke workflow and future acceptance target. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/modules/common.go` | Defines `modules.NativeModule` and `modules.Register`. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/engine/factory.go` | Defines default-vs-explicit module builder behavior. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/go-go-goja/pkg/engine/module_middleware.go` | Explains why `MiddlewareAdd("minitrace")` cannot help before registration. |
| `/home/manuel/workspaces/2026-06-07/club-meetup-site/goja-text/pkg/template/module.go` | Working sibling pattern for self-registering a reusable module. |

## Acceptance criteria

- [ ] A hand-built host that imports `github.com/go-go-golems/go-minitrace/pkg/minitracejs` can call `require("minitrace")` in a runtime created by `engine.NewRuntimeFactoryBuilder().Build()`.
- [ ] The same runtime can still require normal default modules such as `fs`.
- [ ] Documentation shows a hand-built host example and explains the blank import requirement.
- [ ] `examples/xgoja/minitrace-command-provider` is migrated to current xgoja spec format.
- [ ] `make smoke` passes in `examples/xgoja/minitrace-command-provider`.
- [ ] Tests cover the default-registry loading behavior.
- [ ] `GOWORK=off go test ./... -count=1` passes or any unrelated failures are documented.

## References

- GitHub issue: https://github.com/go-go-golems/go-minitrace/issues/20
- Captured issue body: `sources/01-github-issue-20.md`
- Probe script: `scripts/01-probe-module-loading.sh`
- Probe output: `sources/02-module-loading-probe-output.txt`
- xgoja example check script: `scripts/02-check-xgoja-example.sh`
- xgoja example output: `sources/03-xgoja-example-check-output.txt`
