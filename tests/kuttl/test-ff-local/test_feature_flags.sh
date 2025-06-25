#!/bin/bash

FEATURE_FLAGS_POD=$(kubectl -n test-ff-local get pod -l env-app=test-ff-local-featureflags -l service=featureflags --output=jsonpath={.items..metadata.name})
ADMIN_TOKEN=$(kubectl -n test-ff-local get secret test-ff-local-featureflags  -o json | jq -r '.data.adminAccessToken | @base64d')
CLIENT_TOKEN=$(kubectl -n test-ff-local get secret test-ff-local-featureflags  -o json | jq -r '.data.clientAccessToken | @base64d')
FEATURE_TOGGLE_NAME='my-feature-toggle-1'

# Common HTTP retry function using kubectl exec
http_retry() {
    local max_attempts="$1"
    local method="$2"
    local auth_token="$3"
    local url="$4"
    local post_data="${5:-}"  # Default to empty string if not provided
    local delay=2
    local attempt=1

    # note: whenever the version of wget running in the container is >=1.18, we can use
    # the --retry-status flag and avoid this complicated retry logic
    while [ $attempt -le $max_attempts ]; do
        local cmd="kubectl exec -n test-ff-local \"$FEATURE_FLAGS_POD\" -- wget -q -O-"

        # Add authorization header
        if [ -n "$auth_token" ]; then
            cmd="$cmd --header \"Authorization: $auth_token\""
        fi

        # Check if this is a POST request
        if [ "$method" = "POST" ]; then
            # Always add Content-Type header for POST requests
            cmd="$cmd --header \"Content-Type: application/json\""
            # Add POST data - use single quotes to prevent shell interpretation of JSON quotes
            cmd="$cmd --post-data '$post_data'"
        fi

        cmd="$cmd \"$url\""

        if eval "$cmd"; then
            return 0
        fi

        if [ $attempt -lt $max_attempts ]; then
            echo "Attempt $attempt failed, retrying in $delay seconds..." >&2
            sleep $delay
        fi

        attempt=$((attempt + 1))
    done

    echo "All $max_attempts attempts failed" >&2
    return 1
}

get_request_edge() {
    local TOKEN="$1"
    local ENDPOINT="$2"
    local RETRIES="$3"
    http_retry "$RETRIES" "GET" "$TOKEN" "test-ff-local-featureflags-edge:3063${ENDPOINT}"
}

get_request() {
    local TOKEN="$1"
    local ENDPOINT="$2"
    local RETRIES="$3"
    http_retry "$RETRIES" "GET" "$TOKEN" "localhost:4242${ENDPOINT}"
}

post_request() {
    local TOKEN="$1"
    local ENDPOINT="$2"
    local DATA="$3"
    local RETRIES="$4"
    http_retry "$RETRIES" "POST" "$TOKEN" "localhost:4242${ENDPOINT}" "$DATA"
}

echo "Testing that feature flags service is ready..."
if ! get_request "$CLIENT_TOKEN" "/api/client/features" 15; then
    echo "Feature flags service not ready"
    exit 1
fi

echo "Testing that feature toggle '$FEATURE_TOGGLE_NAME' does not exist initially..."
if get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" 3; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should not exist"
    exit 1
fi

echo "Creating feature toggle '$FEATURE_TOGGLE_NAME'..."
if ! post_request "$ADMIN_TOKEN" \
    "/api/admin/projects/default/features" \
    "{ \"name\": \"$FEATURE_TOGGLE_NAME\" }" 3; then
    echo "Error creating feature flag!"
    exit 1
fi

echo "Verifying that feature toggle '$FEATURE_TOGGLE_NAME' exists after creation..."
if ! get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" 3; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should exist"
    exit 1
fi

echo "Verifying that feature toggle '$FEATURE_TOGGLE_NAME' is disabled by default..."
if [ 'true' != "$(get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" 3 | jq '.enabled==false')" ]; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should be disabled"
    exit 1
fi

echo "Enabling feature toggle '$FEATURE_TOGGLE_NAME'..."
if ! post_request "$ADMIN_TOKEN" \
    "/api/admin/projects/default/features/$FEATURE_TOGGLE_NAME/environments/development/on" "" 3; then
    echo "Error enabling feature toggle '$FEATURE_TOGGLE_NAME'"
    exit 1
fi

echo "Verifying that feature toggle '$FEATURE_TOGGLE_NAME' is enabled..."
if [ 'true' != "$(get_request "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" 3 | jq '.enabled==true')" ]; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should be enabled"
    exit 1
fi

echo "Testing that feature toggle '$FEATURE_TOGGLE_NAME' is available through edge service..."
if ! get_request_edge "$CLIENT_TOKEN" "/api/client/features/$FEATURE_TOGGLE_NAME" 15; then
    echo "Feature toggle '$FEATURE_TOGGLE_NAME' should be available through edge"
    exit 1
fi
