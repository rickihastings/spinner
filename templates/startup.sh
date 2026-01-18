#!/bin/bash
set -e

# Check if REPO_URL is set
if [ -z "$REPO_URL" ]; then
  echo "Error: REPO_URL environment variable is not set"
  exit 1
fi

echo "Cloning repository: $REPO_URL"
git clone "$REPO_URL" .

echo "Repository cloned to /workspace"

echo "Verifying clone..."
git status

echo "hello world"

echo "Repository cloned successfully. Container is ready."

# Keep container running
tail -f /dev/null
