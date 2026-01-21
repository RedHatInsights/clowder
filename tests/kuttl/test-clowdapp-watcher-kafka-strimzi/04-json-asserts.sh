#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-strimzi"

# Test commands from original yaml file
sleep 5
sh kafka_secret_check.sh 30
