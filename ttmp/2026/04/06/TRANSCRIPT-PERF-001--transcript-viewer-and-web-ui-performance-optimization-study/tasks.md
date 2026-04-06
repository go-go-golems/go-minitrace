# Tasks

## Complete

- [x] Create a new performance optimization ticket for the web UI
- [x] Map the relevant frontend and backend code paths for transcript rendering
- [x] Add a repeatable Playwright baseline measurement script under `scripts/`
- [x] Capture baseline measurements for the current local dev stack
- [x] Write the primary design / implementation guide
- [x] Write the investigation diary
- [x] Relate key files to the ticket docs

## Recommended implementation phases

### Phase 1 — Transcript mount hygiene

- [ ] Add `unmountOnExit` to transcript block collapse regions
- [ ] Add `unmountOnExit` to tool-call detail collapse regions
- [ ] Rerun baseline measurements and compare DOM counts / load times

### Phase 2 — Split block header from block body

- [ ] Separate lightweight block header UI from expensive block body UI
- [ ] Keep collapsed block footprint minimal
- [ ] Verify focused-target and annotation navigation still work correctly

### Phase 3 — Transcript virtualization

- [ ] Choose a virtualization approach for block lists
- [ ] Render only visible blocks plus overscan
- [ ] Integrate focused target scrolling with virtualized rows
- [ ] Rerun baseline measurements on the same session

### Phase 4 — Session Browser scalability

- [ ] Add virtualization or threshold-based virtualization to Session Browser
- [ ] Rerun route-load measurements on a larger archive if available

### Phase 5 — Query results scalability

- [ ] Memoize `ResultsTable` sorting
- [ ] Evaluate result-table virtualization for large result sets
- [ ] Revisit 3-second polling behavior in Query Editor

### Phase 6 — Backend shaping (optional)

- [ ] Evaluate whether transcript summary/body payload splitting is justified
- [ ] Prototype a block-body-on-demand API only if frontend wins are insufficient

