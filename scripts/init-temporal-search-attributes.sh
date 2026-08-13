#!/usr/bin/env bash
set -e

echo "Registering custom Temporal Search Attributes..."
if command -v docker >/dev/null 2>&1; then
    docker compose exec -T temporal temporal operator search-attribute create \
      --address 127.0.0.1:7233 \
      --name JourneyName --type Keyword \
      --name InternalWorkflowID --type Keyword >/dev/null 2>&1 || true
fi
echo "Temporal search attributes check complete."
