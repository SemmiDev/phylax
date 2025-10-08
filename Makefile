.PHONY: build install clean test run deps help

APP_NAME=phylax
VERSION=2.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
BINARY_PATH=./bin/$(APP_NAME)
CONFIG_PATH=/etc/$(APP_NAME)
SERVICE_PATH=/etc/systemd/system/$(APP_NAME).service
LOG_DIR=/var/log/$(APP_NAME)
BACKUP_DIR=/var/backups/databases

LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $1, $2}'

build: ## Build the application
	@echo "Building $(APP_NAME) v$(VERSION)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o $(BINARY_PATH) ./cmd/backup

install: build ## Install as systemd service
	@echo "Installing $(APP_NAME)..."
	@sudo mkdir -p $(CONFIG_PATH)
	@sudo mkdir -p $(LOG_DIR)
	@sudo mkdir -p $(BACKUP_DIR)
	@sudo cp $(BINARY_PATH) /usr/local/bin/$(APP_NAME)
	@sudo cp configs/config.yaml $(CONFIG_PATH)/config.yaml
	@sudo cp scripts/backup.service $(SERVICE_PATH)
	@sudo chmod 600 $(CONFIG_PATH)/config.yaml
	@sudo systemctl daemon-reload
	@echo ""
	@echo "✓ Installation complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit configuration: sudo nano $(CONFIG_PATH)/config.yaml"
	@echo "  2. Start service: sudo systemctl start $(APP_NAME)"
	@echo "  3. Enable on boot: sudo systemctl enable $(APP_NAME)"
	@echo "  4. Check status: sudo systemctl status $(APP_NAME)"
	@echo "  5. View logs: sudo journalctl -u $(APP_NAME) -f"

uninstall: ## Uninstall the service
	@echo "Uninstalling $(APP_NAME)..."
	@sudo systemctl stop $(APP_NAME) || true
	@sudo systemctl disable $(APP_NAME) || true
	@sudo rm -f /usr/local/bin/$(APP_NAME)
	@sudo rm -f $(SERVICE_PATH)
	@sudo systemctl daemon-reload
	@echo "✓ Uninstallation complete (config and logs preserved)"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin

test: ## Run tests
	@echo "Running tests..."
	@go test -v -cover ./...

run: ## Run locally
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/backup -config configs/config.yaml

deps: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

check: ## Check for required tools
	@echo "Checking for required tools..."
	@command -v mysql >/dev/null 2>&1 || echo "⚠ mysql client not found"
	@command -v mysqldump >/dev/null 2>&1 || echo "⚠ mysqldump not found"
	@command -v psql >/dev/null 2>&1 || echo "⚠ psql not found"
	@command -v pg_dump >/dev/null 2>&1 || echo "⚠ pg_dump not found"
	@command -v mongodump >/dev/null 2>&1 || echo "⚠ mongodump not found"
	@echo "✓ Check complete"

logs: ## View service logs
	@sudo journalctl -u $(APP_NAME) -f

status: ## Check service status
	@sudo systemctl status $(APP_NAME)
