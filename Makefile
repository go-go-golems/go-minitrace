.PHONY: all lint test build goreleaser release frontend dev dev-serve dev-logs dev-test dev-annotate dev-clean glazed-lint-build glazed-lint

all: lint test build

BINARY ?= go-minitrace
MODULE ?= github.com/go-go-golems/go-minitrace
GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_TOOL_VERSION ?= v1.3.5
GLAZED_LINT_FLAGS ?= -glazedclilint.allow-paths=pkg/analysis/,pkg/cli/,pkg/cmds/fields/,pkg/cmds/logging/,pkg/cmds/sources/,pkg/help/
GLAZED_LINT_DIRS ?= ./cmd/... ./pkg/...
CMD_DIR ?= ./cmd/$(BINARY)

VERSION ?= v0.0.0
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v
glazed-lint-build:
	@echo "Building glazed-lint from pinned tool module..."
	@echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION)"
	@GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION)

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)


lint: glazed-lint-build
	golangci-lint run -v
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

lintmax: glazed-lint-build
	golangci-lint run -v --max-same-issues=100
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated -exclude=G101,G304,G301,G306 -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	go test ./...

build:
	go generate ./...
	go build ./...

frontend:
	go generate ./cmd/go-minitrace/cmds/serve

build-bin:
	go generate ./...
	mkdir -p ./dist
	go build -o ./dist/$(BINARY) $(CMD_DIR)

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOPROXY=proxy.golang.org go list -m $(MODULE)@$(shell svu current)

bump-glazed:
	go get github.com/go-go-golems/glazed@latest
	go get github.com/go-go-golems/clay@latest
	go mod tidy

install:
	go generate ./...
	go install $(CMD_DIR)

# ── Dev environment ─────────────────────────────────────────────────────────

# Sessions used for local dev/testing (see tmp/sessions/ and tmp/output/)
DEV_SESSION_GLOB := ./tmp/output/active/*/*.minitrace.json
DEV_PORT := 8080
DEV_DB := ./tmp/output/analysis.dev.db
DEV_WEB_DIR := ./web
DEV_WEB_PORT := 5173
DEV_TMUX := go-minitrace-dev

# Start a full dev environment in a tmux session:
#   - top pane:   serve backend (Go) on :$(DEV_PORT)
#   - middle pane: frontend (Vite) on :$(DEV_WEB_PORT)
#   - bottom pane: shell
# Open with: tmux attach -t $(DEV_TMUX)
# Kill all with: make dev-clean
dev:
	@echo "Starting dev environment in tmux session '$(DEV_TMUX)'..."
	@echo "  Backend: http://localhost:$(DEV_PORT)"
	@echo "  Frontend: http://localhost:$(DEV_WEB_PORT)"
	@echo "  Attach:  tmux attach -t $(DEV_TMUX)"
	@echo "  Clean:   make dev-clean"
	@if tmux has-session -t $(DEV_TMUX) 2>/dev/null; then \
		echo "Session '$(DEV_TMUX)' already running. Attach with: tmux attach -t $(DEV_TMUX)"; \
		exit 1; \
	fi
	@mkdir -p ./tmp/output
	@# Remove stale DB so serve creates a fresh one
	@rm -f $(DEV_DB)
	@# Build binary
	go build -o ./go-minitrace ./cmd/go-minitrace/
	@# Create detached tmux session, then send commands
	tmux new-session -d -s $(DEV_TMUX) -n 'serve'
	@# Pane 1: serve backend (full height, will split later)
	tmux send-keys -t $(DEV_TMUX) './go-minitrace serve \
		--archive-glob "$(DEV_SESSION_GLOB)" \
		--db-path $(DEV_DB) \
		--port $(DEV_PORT) \
		--dev' Enter
	@# Wait for serve to bind port, then split for frontend
	sleep 2
	tmux split-window -t $(DEV_TMUX) -h
	@# Pane 2 (right): frontend dev server
	tmux send-keys -t $(DEV_TMUX):.1 'cd $(DEV_WEB_DIR) && pnpm run dev' Enter
	@# Split bottom pane for logs/shell
	tmux split-window -t $(DEV_TMUX) -v
	@# Pane 3 (bottom): instructions
	tmux send-keys -t $(DEV_TMUX):.2 'echo "=== go-minitrace dev environment ===" && \
		echo "Backend: http://localhost:$(DEV_PORT)" && \
		echo "Frontend: http://localhost:$(DEV_WEB_PORT)" && \
		echo "Sessions: $(DEV_SESSION_GLOB)" && \
		echo "" && \
		echo "Quick test:" && \
		echo "  curl http://localhost:$(DEV_PORT)/api/sessions | python3 -m json.tool | head -20" && \
		echo "  curl http://localhost:$(DEV_PORT)/api/annotations" && \
		echo "  ./go-minitrace annotate add --output-dir ./tmp/output --session <id> --category ai-failure --title test" && \
		echo "" && \
		echo "Tmux keybindings: Ctrl-b d=detach  Ctrl-b o=cycle panes  Ctrl-b x=kill pane"' Enter
	tmux attach -t $(DEV_TMUX)

# Start only the serve backend (no tmux)
dev-serve:
	@mkdir -p ./tmp/output
	@rm -f $(DEV_DB)
	go build -o ./go-minitrace ./cmd/go-minitrace/
	./go-minitrace serve \
		--archive-glob "$(DEV_SESSION_GLOB)" \
		--db-path $(DEV_DB) \
		--port $(DEV_PORT) \
		--dev

# Tail logs from the dev serve process (requires running make dev or dev-serve)
dev-logs:
	@# RequiresDEV_DB log file — serve doesn't write one by default.
	@# Use this after starting serve in foreground or redirecting stderr.
	@echo "Serve logs not captured by default. Start with:"
	@echo "  ./go-minitrace serve --archive-glob '$(DEV_SESSION_GLOB)' --db-path $(DEV_DB) --port $(DEV_PORT) 2>&1 | tee serve.log"

# Run the E2E test scripts against the dev environment
dev-test:
	@echo "Running E2E annotation tests..."
	BIN=$(PWD)/go-minitrace bash ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/scripts/08-e2e-annotate-cli.sh
	BIN=$(PWD)/go-minitrace bash ttmp/2026/04/04/ANNOTATE-CLI--go-minitrace-annotation-cli-and-storage-backend-design/scripts/10-e2e-api.sh
	@echo "All E2E tests passed."

# Run annotate CLI interactively against the dev sessions
dev-annotate:
	./go-minitrace annotate list --output-dir ./tmp/output

# Tear down the dev tmux session
dev-clean:
	tmux kill-session -t $(DEV_TMUX) 2>/dev/null && echo "Killed tmux session '$(DEV_TMUX)'" || echo "No tmux session '$(DEV_TMUX)' running"
	rm -f $(DEV_DB)
	rm -f ./serve.log
