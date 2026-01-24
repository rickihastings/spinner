#!/bin/bash
set -e

# Trap Ctrl+C to exit gracefully
trap 'echo ""; echo "⚠️  Loop interrupted by user (Ctrl+C)"; exit 130' INT

# Validate required environment variables
if [ -z "$PROMPT" ]; then
  echo "Error: PROMPT environment variable is not set"
  exit 1
fi

if [ -z "$MAX_ITERATIONS" ]; then
  echo "Error: MAX_ITERATIONS environment variable is not set"
  exit 1
fi

echo "Starting Ralph loop with prompt: $PROMPT"
echo "Max iterations: $MAX_ITERATIONS"

ITERATION=0
COMPLETION_SIGNAL="~~ FEATURE_COMPLETED ~~"
RATE_LIMIT_WAIT_SECONDS=3660

LOG_DIR="${LOG_DIR:-/logs}"
LOG_FILE="$LOG_DIR/raw.log"

mkdir -p "$LOG_DIR"

wait_for_rate_limit() {
    echo ""
    echo "⚠️  Rate limit detected - waiting 61 minutes..."
    local remaining=$RATE_LIMIT_WAIT_SECONDS
    while [ $remaining -gt 0 ]; do
        sleep 1
        ((remaining--))
    done
    echo -e "\n✅ Wait complete - resuming..."
}

for ((ITERATION=1; ITERATION<=MAX_ITERATIONS; ITERATION++)); do
    echo -e "\n🔁 Iteration $ITERATION/$MAX_ITERATIONS"

    LIMIT_FLAG="/tmp/rate_limit_detected"
    AUTH_ERROR_FLAG="/tmp/auth_error_detected"
    rm -f "$LIMIT_FLAG"
    rm -f "$AUTH_ERROR_FLAG"

    # 1. Run claude
    # 2. Use 'tee' to send output to BOTH the log and a grep check
    # 3. The grep check will touch the flag file immediately upon seeing the error
    # Note: stdbuf -oL ensures line buffering for responsive output
    echo "$PROMPT" | claude -p --dangerously-skip-permissions --output-format=stream-json --verbose 2>&1 \
    | stdbuf -oL tee -a "$LOG_FILE" \
    | while IFS= read -r line; do
        # Check for errors
        if echo "$line" | grep -q 'rate_limit\|hit your limit'; then
            touch /tmp/rate_limit_detected
        fi
        if echo "$line" | grep -q 'authentication_error\|API Error: 401\|Invalid bearer token'; then
            touch /tmp/auth_error_detected
        fi

        # Extract and display text content from assistant message events
        # stream-json format: {"type":"message","role":"assistant","content":[{"type":"text","text":"..."}]}
        MESSAGE=$(echo "$line" | jq -r 'select(.type == "message") | .content[]? | select(.type == "text") | .text // empty' 2>/dev/null)

        if [ -n "$MESSAGE" ] && [ "$MESSAGE" != "null" ]; then
            echo "$MESSAGE"
            if [[ "$MESSAGE" == *"$COMPLETION_SIGNAL"* ]]; then
                echo "200" > /tmp/feature_complete
            fi
        fi
    done

    # Check for authentication error first
    if [ -f "$AUTH_ERROR_FLAG" ]; then
        rm -f "$AUTH_ERROR_FLAG"
        echo -e "\n\n❌ Authentication error detected - please run /login"
        exit 1
    fi

    # Check for feature completion
    if [ -f /tmp/feature_complete ]; then
        rm /tmp/feature_complete
        echo -e "\n\n🎯 ALL TASKS COMPLETE."
        exit 0
    fi

    # Check if the rate limit flag file exists
    if [ -f "$LIMIT_FLAG" ]; then
        rm -f "$LIMIT_FLAG"
        wait_for_rate_limit
        ((ITERATION--))
        continue
    fi

    echo -e "\n✓ Iteration complete."
done

echo -e "\n⚠️  Max iterations ($MAX_ITERATIONS) reached"
exit 1
