#!/bin/sh
set -e

ADDR="${TEMPORAL_ADDRESS:-127.0.0.1:7233}"
echo "Registering custom Temporal Search Attributes against ${ADDR}..."

if command -v docker >/dev/null 2>&1; then
    for _ in $(seq 1 30); do
        if docker compose exec -T temporal temporal operator search-attribute list --address 127.0.0.1:7233 >/dev/null 2>&1; then
            break
        fi
        sleep 1
    done

    docker compose exec -T temporal temporal operator search-attribute create \
      --address 127.0.0.1:7233 \
      --name JourneyName --type Keyword \
      --name InternalWorkflowID --type Keyword \
      --name IsStaticList --type Bool \
      --name StaticListID --type Keyword >/dev/null 2>&1 || true
elif command -v temporal >/dev/null 2>&1; then
    for _ in $(seq 1 30); do
        if temporal operator search-attribute list --address "${ADDR}" >/dev/null 2>&1; then
            break
        fi
        sleep 1
    done

    temporal operator search-attribute create \
      --address "${ADDR}" \
      --name JourneyName --type Keyword \
      --name InternalWorkflowID --type Keyword \
      --name IsStaticList --type Bool \
      --name StaticListID --type Keyword >/dev/null 2>&1 || true
fi

echo "Temporal search attributes check complete."
