.DEFAULT_GOAL := smoke

WORKSPACE_ROOT := $(abspath ../../../..)
XGOJA_ROOT := $(WORKSPACE_ROOT)/go-go-goja
BIN := $(CURDIR)/dist/minitrace-jsverbs
SAMPLE := $(CURDIR)/data/session-a.minitrace.json
XGOJA := cd $(XGOJA_ROOT) && GOWORK=off go run ./cmd/xgoja

.PHONY: smoke doctor list build run clean

smoke: doctor list build run

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

list:
	$(XGOJA) list-modules -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --xgoja-replace $(XGOJA_ROOT) --keep-work

run:
	mkdir -p $(CURDIR)/dist
	$(BIN) verbs inspect summary $(SAMPLE) --output json > $(CURDIR)/dist/summary.json
	grep -q '"session_id"' $(CURDIR)/dist/summary.json
	grep -q '"Wire command providers into generated binaries"' $(CURDIR)/dist/summary.json
	$(BIN) verbs inspect tools $(SAMPLE) --limit 5 --output json > $(CURDIR)/dist/tools.json
	grep -q '"tool_name"' $(CURDIR)/dist/tools.json
	grep -q '"calls"' $(CURDIR)/dist/tools.json

clean:
	rm -rf $(CURDIR)/dist
