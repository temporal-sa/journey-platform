#!/usr/bin/env bash
set -e

BOOTSTRAP_SERVER="${KAFKA_BOOTSTRAP_SERVERS:-kafka:29092}"

echo "Waiting for Kafka to be ready at ${BOOTSTRAP_SERVER}..."
until kafka-topics --bootstrap-server "${BOOTSTRAP_SERVER}" --list > /dev/null 2>&1; do
  sleep 2
done

echo "Kafka is ready. Creating topics..."

TOPICS=(
  "journey-events:3:1"
  "journey-outcomes:3:1"
  "journey-commands:3:1"
  "journey-dlq:1:1"
  "journey-metrics:1:1"
)

for topic_config in "${TOPICS[@]}"; do
  IFS=":" read -r topic partitions replicas <<< "${topic_config}"
  echo "Creating topic '${topic}' (partitions: ${partitions}, replication-factor: ${replicas})..."
  kafka-topics --bootstrap-server "${BOOTSTRAP_SERVER}" \
    --create --if-not-exists \
    --topic "${topic}" \
    --partitions "${partitions}" \
    --replication-factor "${replicas}"
done

echo "Kafka topic initialization completed."
