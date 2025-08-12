#!/bin/bash

if [ $# -ne 1 ]; then
    echo "Usage: $0 <timeout_in_seconds>"
    exit 1
fi

TIMEOUT=$1
START_TIME=$(date +%s)
PREV_HOSTNAME=$(jq -r '.kafka.brokers[0].hostname' < /tmp/test-clowdapp-watcher-kafka-strimzi-json)
PREV_PASSWORD=$(cat /tmp/test-clowdapp-watcher-kafka-strimzi-json-pw)
PASSWORD_CHANGED=false
HASHCACHE_CHANGED=false

while true; do
    # Check elapsed time
    CURRENT_TIME=$(date +%s)
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    if [ "$ELAPSED_TIME" -ge "$TIMEOUT" ]; then
        echo "Kafka SASL password check: FALSE"
        echo "HashCache diff comparison: FALSE"
        echo "Script timed out after $TIMEOUT seconds."
        exit 1
    fi

    # Execute commands
    sleep 5
    kubectl get secret --namespace=test-clowdapp-watcher-kafka-strimzi puptoo -o json > /tmp/test-clowdapp-watcher-kafka-strimzi2
    jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-strimzi2 | base64 -d > /tmp/test-clowdapp-watcher-kafka-strimzi2-json

    CURRENT_HOSTNAME=$(jq -r '.kafka.brokers[0].hostname' < /tmp/test-clowdapp-watcher-kafka-strimzi2-json)
    CURRENT_PASSWORD=$(jq -r '.kafka.brokers[0].sasl.password' < /tmp/test-clowdapp-watcher-kafka-strimzi2-json)
    jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-strimzi2-json > /tmp/test-clowdapp-watcher-kafka-strimzi-hash-cache2

    if [ "$CURRENT_HOSTNAME" != "$PREV_HOSTNAME" ]; then
        echo "Kafka broker hostname check: $CURRENT_HOSTNAME"
        PREV_HOSTNAME=$CURRENT_HOSTNAME
    fi

    if [ "$CURRENT_PASSWORD" != "$PREV_PASSWORD" ]; then
        echo "Kafka SASL password was changed check: TRUE"
        PASSWORD_CHANGED=true
        PREV_PASSWORD=$CURRENT_PASSWORD
    fi

    if diff /tmp/test-clowdapp-watcher-kafka-strimzi-hash-cache /tmp/test-clowdapp-watcher-kafka-strimzi-hash-cache2 > /dev/null; then
        HASHCACHE_CHANGED=false
    else
        echo "HashCache diff comparison: TRUE"
        HASHCACHE_CHANGED=true
    fi

    # Exit if both conditions are met
    if [ "$PASSWORD_CHANGED" = true ] && [ "$HASHCACHE_CHANGED" = true ]; then
        echo "Both conditions met, exiting with status 0."
        exit 0
    fi
done
