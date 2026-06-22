# Tasks

## TODO

- [x] Create docmgr ticket under `go-minitrace/ttmp`.
- [x] Capture GitHub issue #20 as a ticket source.
- [x] Inspect go-minitrace minitracejs loader/provider files.
- [x] Inspect go-go-goja default-registry and engine builder behavior.
- [x] Inspect goja-text template module as the self-registration pattern.
- [x] Add module-loading reproduction script and captured output.
- [x] Add xgoja example check script and captured output.
- [x] Write detailed intern-oriented design and implementation guide.
- [x] Write investigation diary.
- [x] Relate key files and update changelog.
- [x] Run docmgr doctor and resolve hygiene issues if needed.
- [x] Upload documentation bundle to reMarkable.

## Future implementation tasks

- [x] Implement `pkg/minitracejs` default-registry native module adapter.
- [x] Add runtime integration test for plain builder `require("minitrace")`.
- [x] Update README with hand-built host example.
- [x] Migrate `examples/xgoja/minitrace-command-provider` to current xgoja v2 spec.
- [x] Validate `make smoke` in the xgoja example.
- [ ] Validate `GOWORK=off go test ./... -count=1` before release.
- [x] Commit initial ticket documentation before code changes
- [x] Implement default-registry minitracejs module adapter
- [x] Add runtime integration tests for default builder require("minitrace")
- [x] Document hand-built host usage in README
- [x] Migrate xgoja command-provider example and validate make smoke
- [ ] Run final repository validation including GOWORK=off
