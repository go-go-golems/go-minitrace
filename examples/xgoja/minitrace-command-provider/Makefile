.DEFAULT_GOAL := smoke

WORKSPACE_ROOT := $(abspath ../../../..)
XGOJA_ROOT := $(WORKSPACE_ROOT)/go-go-goja
BIN := $(CURDIR)/dist/minitrace-command-provider
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
	rm -rf $(CURDIR)/dist/report
	$(BIN) traces reports markdown-summary \
		--archive-glob '$(CURDIR)/data/*.minitrace.json' \
		--out $(CURDIR)/dist/report \
		--title 'Minitrace Smoke Report' \
		--output json
	test -f $(CURDIR)/dist/report/minitrace-summary.md
	grep -q '# Minitrace Smoke Report' $(CURDIR)/dist/report/minitrace-summary.md
	grep -q 'Wire command providers into generated binaries' $(CURDIR)/dist/report/minitrace-summary.md

clean:
	rm -rf $(CURDIR)/dist
