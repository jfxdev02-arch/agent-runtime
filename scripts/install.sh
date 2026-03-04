#!/bin/bash
set -e

# ============================================================
# Agent Runtime — Install Script
# Creates .env (if missing), builds, and installs as a systemd service
# ============================================================

SERVICE_NAME="agent-runtime"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY_NAME="agent-runtime"
CURRENT_USER="$(whoami)"

echo "=========================================="
echo "  Agent Runtime — Installer"
echo "=========================================="
echo ""
echo "Project dir : $PROJECT_DIR"
echo "User        : $CURRENT_USER"
echo ""

# --- Step 1: Create .env if it doesn't exist ---
ENV_FILE="$PROJECT_DIR/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo "[1/4] Creating .env from template..."
    cp "$PROJECT_DIR/.env.example" "$ENV_FILE"
    # Auto-fill paths based on current user/location
    sed -i "s|WORKSPACE_ROOT=.*|WORKSPACE_ROOT=$HOME|" "$ENV_FILE"
    sed -i "s|PROMPTS_DIR=.*|PROMPTS_DIR=$PROJECT_DIR/prompts|" "$ENV_FILE"
    sed -i "s|DB_PATH=.*|DB_PATH=$PROJECT_DIR/agent.db|" "$ENV_FILE"
    echo "  Created $ENV_FILE"
    echo "  ⚠ EDIT .env with your API keys before starting!"
    echo ""
else
    echo "[1/4] .env already exists, skipping."
fi

# --- Step 2: Build ---
echo "[2/4] Building..."
cd "$PROJECT_DIR"
go mod tidy 2>/dev/null || true
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
LDFLAGS="-w -s -X github.com/dev/agent-runtime/internal/updater.Version=$VERSION"
go build -ldflags="$LDFLAGS" -o "$BINARY_NAME" cmd/agent/main.go
echo "  Built $PROJECT_DIR/$BINARY_NAME ($VERSION)"

# --- Step 3: Create systemd service ---
echo "[3/4] Creating systemd service..."

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

sudo bash -c "cat > $SERVICE_FILE" << EOF
[Unit]
Description=Agent Runtime
After=network.target

[Service]
Type=simple
User=$CURRENT_USER
WorkingDirectory=$PROJECT_DIR
EnvironmentFile=$PROJECT_DIR/.env
ExecStart=$PROJECT_DIR/$BINARY_NAME
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "  Created $SERVICE_FILE"

# --- Step 4: Enable and start ---
echo "[4/4] Enabling and starting service..."
sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE_NAME"
sudo systemctl start "$SERVICE_NAME"

echo ""
echo "=========================================="
echo "  ✅ Installation complete!"
echo "=========================================="
echo ""
echo "  Status  : sudo systemctl status $SERVICE_NAME"
echo "  Logs    : sudo journalctl -u $SERVICE_NAME -f"
echo "  Stop    : sudo systemctl stop $SERVICE_NAME"
echo "  Restart : sudo systemctl restart $SERVICE_NAME"
echo "  Web UI  : http://$(hostname -I | awk '{print $1}'):$(grep PORT $ENV_FILE | cut -d= -f2)"
echo ""
