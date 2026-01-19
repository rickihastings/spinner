#!/bin/bash
set -e

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

while [ $ITERATION -lt $MAX_ITERATIONS ]; do
  ITERATION=$((ITERATION + 1))
  echo ""
  echo "=== Ralph Loop Iteration $ITERATION/$MAX_ITERATIONS ==="
  echo ""

  # Run claude with the prompt and capture output
  OUTPUT=$(echo "$PROMPT" | claude --dangerously-skip-permissions 2>&1 || true)
  echo "$OUTPUT"

  # Check if feature is completed
  if echo "$OUTPUT" | grep -q "~~ FEATURE_COMPLETED ~~"; then
    echo ""
    echo "Feature completed after $ITERATION iterations. Exiting."
    exit 0
  fi
done

echo ""
echo "Max iterations ($MAX_ITERATIONS) reached. Exiting."
exit 0
