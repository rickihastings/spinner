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
RATE_LIMIT_WAIT_SECONDS=3660  # 61 minutes

LOG_DIR="/logs"
LOG_FILE="$LOG_DIR/raw.log"

mkdir -p "$LOG_DIR"

echo "================================================"

wait_for_rate_limit() {
    echo ""
    echo "⚠️  Rate limit detected - waiting 61 minutes..."

    local remaining=$RATE_LIMIT_WAIT_SECONDS
    while [ $remaining -gt 0 ]; do
        printf "\r⏳ Time remaining: %02d:%02d:%02d" $((remaining/3600)) $((remaining%3600/60)) $((remaining%60))
        sleep 1
        ((remaining--))
    done

    echo ""
    echo "✅ Wait complete - resuming..."
}

for ((ITERATION=1; ITERATION<=MAX_ITERATIONS; ITERATION++)); do
    echo -e "\n🔁 Iteration $ITERATION/$MAX_ITERATIONS"
    echo "------------------------------------------------"

    STATUS_FILE=$(mktemp)
    echo "0" > "$STATUS_FILE"

    set +e
    echo "$PROMPT" | claude -p \
        --dangerously-skip-permissions \
        --output-format=stream-json --verbose 2>&1 | while read -r line; do

        # 1. Stream the raw line to the external log file immediately
        echo "$line" >> "$LOG_FILE"

        # 2. Check for the specific rate_limit_error in the JSON
        if echo "$line" | jq -e 'select(.error.type == "rate_limit_error")' >/dev/null 2>&1; then
            echo "202" > "$STATUS_FILE"
            # Kill the specific Claude process to stop the stream
            pkill -P $$ claude 2>/dev/null
            break
        fi

        # 3. Extract and print text content normally
        CLEAN_TEXT=$(echo "$line" | jq -r 'select(.message.content != null) | .message.content[] | select(.text != null) | .text' 2>/dev/null || true)

        # Print clean text to Docker STDOUT (You see this in 'docker logs')
        if [ -n "$CLEAN_TEXT" ]; then
            echo -n "$CLEAN_TEXT"

            # Check for completion signal
            if [[ "$CLEAN_TEXT" == *"$COMPLETION_SIGNAL"* ]]; then
                echo "200" > "$STATUS_FILE"
                pkill -P $$ claude 2>/dev/null
                break
            fi
        fi
    done
done

echo -e "\n⚠️  Max iterations ($MAX_ITERATIONS) reached"
exit 1
