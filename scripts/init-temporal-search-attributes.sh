#!/usr/bin/env bash
set -e

TEMPORAL_ADDRESS="${TEMPORAL_ADDRESS:-temporal:7233}"

echo "Waiting for Temporal to be ready at ${TEMPORAL_ADDRESS}..."
until temporal operator cluster health --address "${TEMPORAL_ADDRESS}" > /dev/null 2>&1; do
  sleep 2
done

echo "Registering custom Temporal Search Attributes..."
temporal operator search-attribute create \
  --address "${TEMPORAL_ADDRESS}" \
  --name JourneyName --type Keyword \
  --name InternalWorkflowID --type Keyword || true

echo "Temporal search attributes registered successfully."
