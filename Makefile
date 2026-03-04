.PHONY: build install uninstall start stop restart status logs clean

BINARY = agent-runtime

# Build the binary (optimized)
build:
	go mod tidy
	go build -ldflags="-w -s" -o $(BINARY) cmd/agent/main.go
	@echo "✅ Built ./$(BINARY)"

# Install as systemd service (creates .env, builds, enables)
install:
	@chmod +x scripts/install.sh
	@bash scripts/install.sh

# Remove systemd service
uninstall:
	@chmod +x scripts/uninstall.sh
	@bash scripts/uninstall.sh

# Run locally (not as service)
run: build
	@if [ ! -f .env ]; then cp .env.example .env; echo "⚠ Edit .env with your API keys first!"; exit 1; fi
	@set -a && source .env && set +a && ./$(BINARY)

# Service management shortcuts
start:
	sudo systemctl start agent-runtime

stop:
	sudo systemctl stop agent-runtime

restart:
	sudo systemctl restart agent-runtime

status:
	sudo systemctl status agent-runtime

logs:
	sudo journalctl -u agent-runtime -f

# Build for Raspberry Pi (ARM64) from another machine
build-pi:
	GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o $(BINARY) cmd/agent/main.go
	@echo "✅ Built ./$(BINARY) for linux/arm64"

# Clean build artifacts
clean:
	rm -f $(BINARY)
	@echo "✅ Cleaned"
