# Build variables
LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe --tags)
BUILDSTR := ${VERSION} (Commit: ${LAST_COMMIT_DATE} (${LAST_COMMIT}), Build: $(shell date +"%Y-%m-%d %H:%M:%S %z"))

# Binary names and paths
BIN_LIBRE_DESK := libredesk.bin
FRONTEND_DIR := frontend
FRONTEND_DIST := ${FRONTEND_DIR}/dist
STATIC := ${FRONTEND_DIST} i18n schema.sql static
GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin

# Default target
.DEFAULT_GOAL := build

$(STUFFBIN):
	@echo "→ Installing stuffbin..."
	@go install github.com/knadh/stuffbin/...

.PHONY: install-deps
install-deps: $(STUFFBIN)
	@echo "→ Installing frontend dependencies..."
	@cd ${FRONTEND_DIR} && pnpm install

# Frontend builds
.PHONY: frontend-build
frontend-build:
	@echo "→ Building frontend for production..."
	@cd ${FRONTEND_DIR} && pnpm build

# Backend builds
.PHONY: backend-build
backend-build: $(STUFFBIN)
	@echo "→ Building backend..."
	@CGO_ENABLED=0 go build \
		-ldflags="-X 'main.buildString=${BUILDSTR}' -X 'main.buildDate=${LAST_COMMIT_DATE}' -s -w" \
		-o ${BIN_LIBRE_DESK} cmd/*.go

# Main build targets
.PHONY: build
build: frontend-build backend-build stuff
	@echo "→ Build successful. Current version: $(VERSION)"

# Stuff static assets into binary
.PHONY: stuff
stuff: $(STUFFBIN)
	@echo "→ Stuffing static assets into binary..."
	@$(STUFFBIN) -a stuff -in ${BIN_LIBRE_DESK} -out ${BIN_LIBRE_DESK} ${STATIC}
