# Build variables
LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe --tags) 
BUILDSTR := ${VERSION} (Commit: ${LAST_COMMIT_DATE} (${LAST_COMMIT}), Build: $(shell date +"%Y-%m-%d %H:%M:%S %z"))

# Binary names and paths
BIN_LIBREDESK := libredesk.bin
FRONTEND_DIR := frontend
FRONTEND_DIST := ${FRONTEND_DIR}/dist
STATIC := ${FRONTEND_DIST} i18n schema.sql static
GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin

# The default target to run when `make` is executed.
.DEFAULT_GOAL := build  

# Install stuffbin if it doesn't exist.
$(STUFFBIN):
	@echo "→ Installing stuffbin..."
	@go install github.com/knadh/stuffbin/...

# Install dependencies for both backend and frontend.
.PHONY: install-deps
install-deps: $(STUFFBIN)
	@echo "→ Installing frontend dependencies..."
	@cd ${FRONTEND_DIR} && pnpm install

# Build the frontend for production.
.PHONY: frontend-build
frontend-build:
	@echo "→ Building frontend for production..."
	@cd ${FRONTEND_DIR} && pnpm build

# Run the Go backend server in development mode.
.PHONY: run-backend
run-backend:
	@echo "→ Running backend..."
	@go run cmd/*.go

# Run the JS frontend server in development mode.
.PHONY: run-frontend
run-frontend:
	@echo "→ Installing frontend dependencies (if not already installed)..."
	@cd ${FRONTEND_DIR} && pnpm install
	@echo "→ Running frontend..."
	@export VUE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm dev

# Build the backend binary.
.PHONY: backend-build
backend-build: $(STUFFBIN)
	@echo "→ Building backend..."
	@CGO_ENABLED=0 go build -a\
		-ldflags="-X 'main.buildString=${BUILDSTR}' -X 'main.buildDate=${LAST_COMMIT_DATE}' -s -w" \
		-o ${BIN_LIBREDESK} cmd/*.go

# Main build target: builds both frontend and backend, then stuffs static assets into the binary.
.PHONY: build
build: frontend-build backend-build stuff
	@echo "→ Build successful. Current version: $(VERSION)"

# Stuff static assets into the binary using stuffbin.
.PHONY: stuff
stuff: $(STUFFBIN)
	@echo "→ Stuffing static assets into binary..."
	@$(STUFFBIN) -a stuff -in ${BIN_LIBREDESK} -out ${BIN_LIBREDESK} ${STATIC}