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

# If PROMPT is set, run Ralph loop
if [ -n "$PROMPT" ]; then
  echo ""

  # Get default branch
  DEFAULT_BRANCH=$(git symbolic-ref refs/remotes/origin/HEAD | sed 's@^refs/remotes/origin/@@')
  echo "Default branch: $DEFAULT_BRANCH"

  # If BRANCH is specified, checkout or create it
  if [ -n "$BRANCH" ]; then
    echo "Branch specified: $BRANCH"

    # Check if branch exists remotely
    if git ls-remote --heads origin "$BRANCH" | grep -q "$BRANCH"; then
      echo "Branch exists remotely, checking out..."
      git checkout "$BRANCH"
    else
      echo "Branch does not exist, creating from $DEFAULT_BRANCH..."
      git checkout -b "$BRANCH"
    fi
  else
    echo "No branch specified, using default branch: $DEFAULT_BRANCH"
  fi

  echo "Current branch: $(git branch --show-current)"
  echo ""
  echo "Starting Ralph loop for autonomous implementation..."

  # Execute Ralph loop
  /usr/local/bin/ralph-loop.sh
else
  echo ""
  echo "Repository cloned successfully. Container is ready."

  # Keep container running
  tail -f /dev/null
fi
