#!/bin/bash
set -e

# ============================================================
# Agent Runtime — Uninstall Script
# Stops and removes the systemd service
# ============================================================

SERVICE_NAME="agent-runtime"

echo "=========================================="
echo "  Agent Runtime — Uninstaller"
echo "=========================================="
echo ""

# Stop service
echo "[1/3] Stopping service..."
sudo systemctl stop "$SERVICE_NAME" 2>/dev/null || true

# Disable service
echo "[2/3] Disabling service..."
sudo systemctl disable "$SERVICE_NAME" 2>/dev/null || true

# Remove service file
echo "[3/3] Removing service file..."
sudo rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
sudo systemctl daemon-reload

echo ""
echo "  ✅ Service removed."
echo "  Note: Binary, .env, and database were NOT deleted."
echo "  To fully remove, delete the project directory manually."
echo ""
