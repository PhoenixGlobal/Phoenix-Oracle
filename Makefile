.DEFAULT_GOAL := build

ENVIRONMENT ?= release

GOPATH ?= $(HOME)/go
BUILDER ?= smartcontract/builder
COMMIT_SHA ?= $(shell git rev-parse HEAD)
VERSION = $(shell cat VERSION)
GOBIN ?= $(GOPATH)/bin
GO_LDFLAGS := $(shell build/shells/ldflags)
GOFLAGS = -ldflags "$(GO_LDFLAGS)"


.PHONY: install
install: statics-autoinstall install-phoenix-autoinstall ## Install phoenix and dependencies.


.PHONY: install-phoenix-autoinstall
install-phoenix-autoinstall: | gomod install-phoenix
.PHONY: statics-autoinstall
statics-autoinstall: | yarndep statics

.PHONY: gomod
gomod: ## install phoenix's go dependencies
	@if [ -z "`which gencodec`" ]; then \
		go get github.com/smartcontractkit/gencodec; \
	fi || true
	go mod download

.PHONY: yarndep
yarndep: ## installed all yarn dependencies
	yarn install --frozen-lockfile --prefer-offline
	mkdir -p ./node_modules/@0x/sol-compiler
	./build/shells/restore-solc-cache

.PHONY: install-phoenix
install-phoenix: phoenix ## Install the phoenix binary.
	mkdir -p $(GOBIN)
	cp $< $(GOBIN)/phoenix

phoenix: statics ## Build the phoenix binary.
	CGO_ENABLED=0 go run build/main.go "${CURDIR}/core/service/ethereum" ## embed contracts in .go file
	go build $(GOFLAGS) -o $@ ./core/

.PHONY: statics
statics: ## Build the frontend UI.
	yarn setup:phoenix
	PHOENIX_VERSION="$(VERSION)@$(COMMIT_SHA)" yarn workspace @phoenix/statics build
	CGO_ENABLED=0 go run build/main.go "${CURDIR}/core/service"





help:
	@echo ""
	@echo "         .__           .__       .__  .__        __"
	@echo "    ____ |  |__ _____  |__| ____ |  | |__| ____ |  | __"
	@echo "  _/ ___\|  |  \\\\\\__  \ |  |/    \|  | |  |/    \|  |/ /"
	@echo "  \  \___|   Y  \/ __ \|  |   |  \  |_|  |   |  \    <"
	@echo "   \___  >___|  (____  /__|___|  /____/__|___|  /__|_ \\"
	@echo "       \/     \/     \/        \/             \/     \/"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
