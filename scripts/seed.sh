#!/usr/bin/env bash
set -euo pipefail

echo "Executing deterministic seed data loader..."
go run ./cmd/control-api --seed
echo "Local data seeded successfully."
