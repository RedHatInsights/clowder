#!/usr/bin/env bash
# Using portable shebang for better compatibility across systems

if [ $# -ne 1 ]; then
    echo "Usage: $0 <timeout_in_seconds>"
    exit 1
fi

TIMEOUT=$1
START_TIME=$(date +%s)
PREV_HOSTNAME=$(jq -r '.kafka.brokers[0].hostname' < /tmp/test-clowdapp-watcher-kafka-msk-env-json)
PREV_USERNAME=$(cat /tmp/test-clowdapp-watcher-kafka-msk-env-json-user)
USERNAME_MATCH=false
HASHCACHE_CHANGED=false

while true; do
    # Check elapsed time
    CURRENT_TIME=$(date +%s)
    ELAPSED_TIME=$((CURRENT_TIME - START_TIME))
    if [ "$ELAPSED_TIME" -ge "$TIMEOUT" ]; then
        echo "Kafka SASL username check: FALSE"
        echo "HashCache diff comparison: FALSE"
        echo "Script timed out after $TIMEOUT seconds."
        exit 1
    fi

    # Execute commands
    sleep 5
    kubectl get secret --namespace=test-clowdapp-watcher-kafka-msk-env puptoo -o json > /tmp/test-clowdapp-watcher-kafka-msk-env2
    jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-msk-env2 | base64 -d > /tmp/test-clowdapp-watcher-kafka-msk-env2-json

    CURRENT_HOSTNAME=$(jq -r '.kafka.brokers[0].hostname' < /tmp/test-clowdapp-watcher-kafka-msk-env2-json)
    CURRENT_USERNAME=$(jq -r '.kafka.brokers[0].sasl.username' < /tmp/test-clowdapp-watcher-kafka-msk-env2-json)
    jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-msk-env2-json > /tmp/test-clowdapp-watcher-kafka-msk-env-hash-cache2

    if [ "$CURRENT_HOSTNAME" != "$PREV_HOSTNAME" ]; then
        echo "Kafka broker hostname check: $CURRENT_HOSTNAME"
        PREV_HOSTNAME=$CURRENT_HOSTNAME
    fi

    if [ "$CURRENT_USERNAME" != "$PREV_USERNAME" ]; then
        if [ "$CURRENT_USERNAME" = "test-clowdapp-watcher-kafka-msk-connect2" ]; then
            echo "Kafka SASL username check: TRUE"
            USERNAME_MATCH=true
        else
            USERNAME_MATCH=false
        fi
        PREV_USERNAME=$CURRENT_USERNAME
    fi

    if diff /tmp/test-clowdapp-watcher-kafka-msk-env-hash-cache /tmp/test-clowdapp-watcher-kafka-msk-env-hash-cache2 > /dev/null; then
        HASHCACHE_CHANGED=false
    else
        echo "HashCache diff comparison: TRUE"
        HASHCACHE_CHANGED=true
    fi

    # Exit if both conditions are met
    if [ "$USERNAME_MATCH" = true ] && [ "$HASHCACHE_CHANGED" = true ]; then
        echo "Both conditions met, exiting with status 0."
        exit 0
    fi

done
