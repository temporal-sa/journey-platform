#!/usr/bin/env bash
set -euo pipefail

echo "=========================================================="
echo "          Journey Platform - Local WebMCP Setup           "
echo "=========================================================="
echo "WebMCP Bridge: Injected at /.webmcp/bridge.js"
echo "WebMCP Proxy:  /mcp -> http://localhost:8087/mcp"
echo "Frontend:      http://localhost:3002/"
echo ""

if command -v cloudflared >/dev/null 2>&1; then
    echo "cloudflared detected! To route this local dev instance via Cloudflare Tunnel:"
    echo "  cloudflared tunnel --url http://localhost:3002"
    echo ""
else
    echo "Note: To map localhost:3002 to a live Cloudflare domain with 1-click WebMCP:"
    echo "  1. Install cloudflared: brew install cloudflared"
    echo "  2. Run: cloudflared tunnel --url http://localhost:3002"
    echo "  3. In Cloudflare Dashboard -> Agent Readiness -> Labs, toggle WebMCP ON for your domain."
    echo ""
fi

echo "Starting dev stack..."
make dev
