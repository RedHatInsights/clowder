#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-strimzi-ephemeral" "test-kafka-strimzi-ephemeral-kafka"

# Test commands from original yaml file
for i in {1..10}; do kubectl get kafka -n test-kafka-strimzi-ephemeral-kafka && kubectl get kafkaconnect -n test-kafka-strimzi-ephemeral-connect && break || sleep 1; done; echo "Kafka or KafkaConnect not found"; exit 1
bash -c 'CLUSTER_NAME=$(kubectl get kafka -n test-kafka-strimzi-ephemeral-kafka -o jsonpath="{.items[0].metadata.name}"); [[ "$CLUSTER_NAME" == "env-test-kafka-strimzi-ephemeral" ]]'
bash -c 'CLUSTER_NAME=$(kubectl get kafkaconnect -n test-kafka-strimzi-ephemeral-connect -o jsonpath="{.items[0].metadata.name}"); [[ "$CLUSTER_NAME" == "env-test-kafka-strimzi-ephemeral" ]]'
bash -c 'KAFKA_CLUSTER_NAME=$(kubectl get kafka -n test-kafka-strimzi-ephemeral-kafka -o jsonpath="{.items[0].metadata.name}"); CONNECT_BOOTSTRAP_SERVERS=$(kubectl get kafkaconnect -n test-kafka-strimzi-ephemeral-connect -o jsonpath="{.items[0].spec.bootstrapServers}"); [[ "$CONNECT_BOOTSTRAP_SERVERS" == "$KAFKA_CLUSTER_NAME-kafka-bootstrap.test-kafka-strimzi-ephemeral-kafka.svc:9092" ]]'
