# Agentrix developer and CI entry points. Quality gates check and fail; they
# never rewrite files (use `make fmt` explicitly to format).
SHELL := /bin/bash
.DEFAULT_GOAL := help
COMPOSE ?= docker compose
GO ?= go
GOFLAGS_RO := -mod=readonly
PG_TEST_PORT ?= 55432

.PHONY: help fmt fmt-check vet test test-race test-integration engine engine-test web-install web-lint web-test web-build web-e2e \
        check demo-up demo-down demo-reset demo-smoke openapi-check

help:
	@echo "make check            gofmt check, vet, unit + race tests"
	@echo "make test-integration PostgreSQL + sandbox + engine integration suite (no skips allowed)"
	@echo "make engine           build bin/starfighter-engine from the commit pinned in engine.lock"
	@echo "make engine-test      rustfmt check, clippy -D warnings and tests of ../agentrix_engine"
	@echo "make web-lint|web-test|web-build|web-e2e"
	@echo "make demo-up          compose stack from scratch + demo bootstrap"
	@echo "make demo-smoke       run the HTTP smoke against the compose stack"
	@echo "make demo-reset       remove ONLY this project's containers and volumes, then demo-up"

fmt:
	gofmt -w $$(git ls-files '*.go')

fmt-check:
	@out=$$(gofmt -l $$(git ls-files '*.go' 2>/dev/null || find . -name '*.go' -not -path './web/*')); \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet $(GOFLAGS_RO) ./...

test:
	$(GO) test $(GOFLAGS_RO) -count=1 $$($(GO) list ./... | grep -v /test/integration)

test-race:
	$(GO) test $(GOFLAGS_RO) -race -count=1 $$($(GO) list ./... | grep -v /test/integration)

check: fmt-check vet test-race

# Requires PostgreSQL 16 reachable with DB_* variables, bubblewrap and the
# pinned engine binary. Any skipped test fails the target.
test-integration:
	@set -o pipefail; AGENTRIX_REQUIRE_DB=1 AGENTRIX_REQUIRE_SANDBOX=1 AGENTRIX_REQUIRE_ENGINE=1 \
	  $(GO) test $(GOFLAGS_RO) -race -count=1 -v -timeout 20m ./test/integration/ ./src/executor/ | tee integration.log
	@if grep -q -- '--- SKIP' integration.log; then echo "unexpected skipped tests"; grep -- '--- SKIP' integration.log; exit 1; fi

openapi-check:
	$(GO) test $(GOFLAGS_RO) -count=1 -run 'TestOpenAPI|TestRoutePolicy' ./src/server/

engine:
	./script/build_engine.sh

engine-test:
	cd ../agentrix_engine && cargo fmt --check && cargo clippy --locked --all-targets -- -D warnings && cargo test --locked

web-install:
	cd web && npm ci

web-lint:
	cd web && npm run lint

web-test:
	cd web && npm test && npm run test:parity

web-build:
	cd web && npm run build && npm run size

web-e2e:
	cd web && npx playwright test

demo-up:
	./script/checkout_engine.sh
	$(COMPOSE) --profile demo up --build -d
	$(COMPOSE) --profile demo wait bootstrap 2>/dev/null || $(COMPOSE) logs -f bootstrap

demo-smoke:
	$(COMPOSE) --profile smoke run --rm smoke

demo-down:
	$(COMPOSE) down

# Scoped to the compose project "agentrix-demo": removes its containers and
# named volumes only.
demo-reset:
	$(COMPOSE) --profile demo --profile smoke down --volumes --remove-orphans
	$(MAKE) demo-up
