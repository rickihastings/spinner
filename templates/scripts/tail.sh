#!/bin/bash

# Check if a filename was provided
if [ -z "$1" ]; then
    echo "Usage: $0 <log-file>"
    exit 1
fi

LOGFILE=$1

# Use tail to follow the file and jq to parse each line
tail -f "$LOGFILE" | while read -r line; do
    # Check if line is valid JSON to avoid errors
    if echo "$line" | jq -e . >/dev/null 2>&1; then
        echo "--------------------------------------------------"
        # Extract Timestamp/Model if available
        echo -e "\033[1;34mMODEL:\033[0m $(echo "$line" | jq -r '.message.model')"

        # Extract and format the Todo list items
        echo -e "\033[1;32mTODOS UPDATED:\033[0m"
        echo "$line" | jq -r '.message.content[].input.todos[] | "- [\(.status)] \(.content)"'

        echo "--------------------------------------------------"
    fi
done
