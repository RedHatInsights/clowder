#!/bin/bash

FEATURE_FLAGS_POD=$(kubectl -n test-ff-local get pod -l env-app=test-ff-local-featureflags --output=jsonpath={.items..metadata.name})
ADMIN_TOKEN=$(kubectl -n test-ff-local get secret test-ff-local-featureflags  -o json | jq -r '.data.adminAccessToken | @base64d')
CLIENT_TOKEN=$(kubectl -n test-ff-local get secret test-ff-local-featureflags  -o json | jq -r '.data.clientAccessToken | @base64d')
FEATURE_TOGGLE_NAME='my-feature-toggle-1'

get_request_edge() {

    local TOKEN="$1"
    local ENDPOINT="$2"

    kubectl exec -n test-ff-local "$FEATURE_FLAGS_POD" -- wget -q -O- \
        --header "Authorization: $TOKEN" "test-ff-local-featureflags-edge:3063${ENDPOINT}"
}

get_request() {

    local TOKEN="$1"
    local ENDPOINT="$2"

    kubectl exec -n test-ff-local "$FEATURE_FLAGS_POD" -- wget -q -O- \
        --header "Authorization: $TOKEN" "localhost:4242${ENDPOINT}"
}

post_request() {

    local TOKEN="$1"
    local ENDPOINT="$2"
    local DATA="$3"

    kubectl exec -n test-ff-local "$FEATURE_FLAGS_POD" -- wget -q -O- \
        --post-data "$DATA" \
        --header "Content-Type: application/json" \
        --header "Authorization: $TOKEN" \
        "localhost:4242${ENDPOINT}"
}

if get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME"; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should not exist"
    exit 1
fi


if ! post_request "$ADMIN_TOKEN" \
    "/api/admin/projects/default/features" \
    "{ \"name\": \"$FEATURE_TOGGLE_NAME\" }"; then
    echo "Error creating feature flag!"
    exit 1
fi

if ! get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME"; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should exist"
    exit 1
fi

if [ 'true' != "$(get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" | jq '.enabled==false')" ]; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should be disabled"
    exit 1
fi

if ! post_request "$ADMIN_TOKEN" \
    "/api/admin/projects/default/features/$FEATURE_TOGGLE_NAME/environments/development/on" ; then
    echo "Error enabling feature toggle '$FEATURE_TOGGLE_NAME'"
    exit 1
fi

if [ 'true' != "$(get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" | jq '.enabled==true')" ]; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should be enabled"
    exit 1
fi

# Unleash seems to refresh the cache every 5 seconds, didn't find a way to force it
# see https://github.com/Unleash/unleash-edge/blob/12cf9e3f87099d3c0dce1884bcd305604c1e68ff/server/src/http/refresher/feature_refresher.rs#L443
echo "Waiting for edge to sync"
sleep 6

if ! get_request_edge "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME"; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should be available through edge"
    exit 1
fi
