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

LOG_DIR="/logs"
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
    rm -f "$LIMIT_FLAG"

    # 1. Run claude
    # 2. Use 'tee' to send output to BOTH the log and a grep check
    # 3. The grep check will touch the flag file immediately upon seeing the error
    echo "$PROMPT" | claude -p --dangerously-skip-permissions --output-format=stream-json --verbose 2>&1 \
    | tee -a "$LOG_FILE" \
    | stdbuf -oL awk '{
        print $0;
        if ($0 ~ /rate_limit/ || $0 ~ /hit your limit/) {
            system("touch /tmp/rate_limit_detected");
        }
    }' | while read -r line; do

        # Extract clean text for the console
        CLEAN_TEXT=$(echo "$line" | jq -r '.message.content[].text // .result // empty' 2>/dev/null | grep -v "null" || true)

        if [ -n "$CLEAN_TEXT" ] && [[ "$CLEAN_TEXT" != *"hit your limit"* ]]; then
            echo -n "$CLEAN_TEXT"
            if [[ "$CLEAN_TEXT" == *"$COMPLETION_SIGNAL"* ]]; then
                echo "200" > /tmp/feature_complete
            fi
        fi
    done

    # Check for feature completion first
    if [ -f /tmp/feature_complete ]; then
        rm /tmp/feature_complete
        echo -e "\n\n🎯 ALL TASKS COMPLETE."
        exit 0
    fi

    # Check if the flag file exists
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
