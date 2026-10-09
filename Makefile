.PHONY: all help build build-client build-server start run stop restart status dev dev-client dev-server clean docker-up docker-down

# Default target
all: help

## ── Help ─────────────────────────────────────────────────────────────
help:
	@echo "=========================================================="
	@echo "  RETRIVA — Universal Media Extractor & Permanent Vault"
	@echo "=========================================================="
	@echo ""
	@echo "  Usage: make <command>"
	@echo ""
	@echo "  Running & Managing:"
	@echo "    make run          Build (if needed) and run Retriva in foreground"
	@echo "    make start        Start Retriva in the background (daemon)"
	@echo "    make stop         Stop any running Retriva background process"
	@echo "    make restart      Stop, rebuild, and restart Retriva in background"
	@echo "    make status       Check if Retriva is running and ping /health"
	@echo ""
	@echo "  Development:"
	@echo "    make dev          Run frontend (Vite) and backend (Go) with live-reload"
	@echo "    make dev-client   Run Vite dev server only (port 5173)"
	@echo "    make dev-server   Run Go backend server directly with go run"
	@echo ""
	@echo "  Building:"
	@echo "    make build        Build production client bundle and Go server binary"
	@echo "    make build-client Build production React bundle (client/dist)"
	@echo "    make build-server Build single Go executable (bin/retriva)"
	@echo "    make clean        Remove compiled binaries and client build files"
	@echo ""
	@echo "  Docker:"
	@echo "    make docker-up    Start Retriva via Docker Compose"
	@echo "    make docker-down  Stop Docker Compose containers"
	@echo "=========================================================="

## ── Building ─────────────────────────────────────────────────────────
build: build-client build-server

build-client:
	@echo "--> Building React frontend..."
	@cd client && npm run build

build-server:
	@echo "--> Compiling Go server binary (bin/retriva)..."
	@mkdir -p bin
	@cd server && go build -o ../bin/retriva cmd/retriva/main.go
	@echo "--> Done: bin/retriva"

## ── Running ──────────────────────────────────────────────────────────
# Run in foreground
run:
	@if [ ! -f "./bin/retriva" ]; then \
		echo "--> Binary not found. Building Retriva first..."; \
		$(MAKE) build; \
	fi
	@echo "--> Starting Retriva on http://localhost:8085 ..."
	@./bin/retriva

# Start in background (daemon mode)
start:
	@if [ ! -f "./bin/retriva" ]; then \
		echo "--> Binary not found. Building Retriva first..."; \
		$(MAKE) build; \
	fi
	@mkdir -p data
	@if pgrep -f "./bin/retriva" > /dev/null; then \
		echo "--> Retriva is already running."; \
	else \
		nohup ./bin/retriva > data/retriva.log 2>&1 & \
		sleep 1; \
		if pgrep -f "./bin/retriva" > /dev/null; then \
			echo "--> Retriva started in background (PID: $$(pgrep -f './bin/retriva' | head -n1))."; \
			echo "--> Web UI available at: http://localhost:8085"; \
			echo "--> Logs written to: data/retriva.log"; \
		else \
			echo "--> Failed to start Retriva. Check data/retriva.log for errors."; \
		fi \
	fi

# Stop running instance
stop:
	@if pgrep -f "./bin/retriva" > /dev/null; then \
		echo "--> Stopping Retriva..."; \
		pkill -f "./bin/retriva"; \
		sleep 1; \
		echo "--> Retriva stopped."; \
	else \
		echo "--> Retriva is not running."; \
	fi

# Restart instance in background
restart: stop build start

# Check status & health
status:
	@echo "--> Checking Retriva process..."
	@if pgrep -f "./bin/retriva" > /dev/null; then \
		echo "    Status: RUNNING (PID: $$(pgrep -f './bin/retriva' | head -n1))"; \
		HEALTH=$$(curl -s http://localhost:8085/health 2>/dev/null); \
		if [ -n "$$HEALTH" ]; then \
			echo "    Health: OK ($$HEALTH)"; \
			echo "    Web UI: http://localhost:8085"; \
		else \
			echo "    Health: Port 8085 not responding"; \
		fi \
	else \
		echo "    Status: NOT RUNNING"; \
	fi

## ── Live Development ────────────────────────────────────────────────
dev:
	@echo "--> Starting live development mode (Client + Server)..."
	@$(MAKE) -j2 dev-client dev-server

dev-client:
	@cd client && npm run dev

dev-server:
	@cd server && go run cmd/retriva/main.go

## ── Clean ────────────────────────────────────────────────────────────
clean:
	@echo "--> Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf client/dist/
	@echo "--> Clean complete."

## ── Docker ───────────────────────────────────────────────────────────
docker-up:
	@docker compose up -d

docker-down:
	@docker compose down
