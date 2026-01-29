#!/bin/bash
set -e

# Check if GITHUB_TOKEN is set
if [ -z "$GITHUB_TOKEN" ]; then
  echo "Error: GITHUB_TOKEN environment variable is not set"
  echo "Please set GITHUB_TOKEN before running spin"
  exit 1
fi

# Configure GitHub authentication
echo "Configuring GitHub authentication..."
gh auth setup-git

# Configure git credential cache with 1-year timeout
git config --global credential.helper 'cache --timeout=31536000'

# Check if REPO_URL is set
if [ -z "$REPO_URL" ]; then
  echo "Error: REPO_URL environment variable is not set"
  exit 1
fi

# Check if repository is already cloned (e.g., on container restart)
if [ -d ".git" ]; then
  echo "Repository already exists, verifying..."

  # Verify it's the correct repository
  CURRENT_REMOTE=$(git config --get remote.origin.url || echo "")
  if [ "$CURRENT_REMOTE" != "$REPO_URL" ]; then
    echo "Error: Existing repository URL ($CURRENT_REMOTE) doesn't match expected URL ($REPO_URL)"
    exit 1
  fi

  echo "Repository verified. Fetching latest changes..."
  git fetch origin
else
  echo "Cloning repository: $REPO_URL"
  git clone "$REPO_URL" .
  echo "Repository cloned to /home/spinner/workspace"
fi

echo "Verifying repository..."
git status

# If PROMPT is set, run Ralph loop
if [ -n "$PROMPT" ]; then
  echo ""

  # Get default branch
  DEFAULT_BRANCH=$(git symbolic-ref refs/remotes/origin/HEAD | sed 's@^refs/remotes/origin/@@')
  echo "Default branch: $DEFAULT_BRANCH"

  # If BRANCH is specified, checkout or create it
  if [ -n "$BRANCH" ]; then
    # Strip surrounding quotes from BRANCH if present (from escapeShellArg)
    BRANCH="${BRANCH#\'}"
    BRANCH="${BRANCH%\'}"

    echo "Branch specified: $BRANCH"

    # Check if branch exists locally
    if git rev-parse --verify "$BRANCH" >/dev/null 2>&1; then
      echo "Branch exists locally, checking out..."
      git checkout "$BRANCH"
    # Check if branch exists remotely
    elif git ls-remote --heads origin "$BRANCH" | grep -q "$BRANCH"; then
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
