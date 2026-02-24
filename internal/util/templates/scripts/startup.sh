#!/bin/bash
set -e

# Check if REPO_URL is set
if [ -z "$REPO_URL" ]; then
  echo "Error: REPO_URL environment variable is not set"
  exit 1
fi

# Check for encrypted secrets blob
if [ ! -f "/run/spinner/secrets.enc" ]; then
  echo "Error: Encrypted secrets blob not found at /run/spinner/secrets.enc"
  echo "Secrets must be configured via 'spinner secret set' before spinning"
  exit 1
fi

# Decrypt secrets and run git auth setup + clone in one shot.
# gh auth setup-git configures git credential helper.
# Credential cache (1-year timeout) persists auth after this block exits.
# After this, GITHUB_TOKEN is no longer needed as an env var for git operations.
spinner secret inject -- sh -c '
  gh auth setup-git
  git config --global credential.helper "cache --timeout=31536000"
  if [ -d ".git" ]; then
    CURRENT_REMOTE=$(git config --get remote.origin.url || echo "")
    if [ "$CURRENT_REMOTE" != "'"$REPO_URL"'" ]; then
      echo "Error: Existing repo URL ($CURRENT_REMOTE) does not match expected ('"$REPO_URL"')"
      exit 1
    fi
    echo "Repository verified. Fetching latest changes..."
    git fetch origin
  else
    echo "Cloning repository: '"$REPO_URL"'"
    git init
    git remote add origin "'"$REPO_URL"'"
    git fetch origin
    DEFAULT_REF=$(git remote show origin | sed -n "s/.*HEAD branch: //p")
    git checkout -f -b "${DEFAULT_REF}" "origin/${DEFAULT_REF}"
    echo "Repository cloned to /home/spinner/workspace"
  fi
'

# Configure git user identity from host
if [ -n "$GIT_USER_NAME" ]; then
  git config --global user.name "$GIT_USER_NAME"
fi

if [ -n "$GIT_USER_EMAIL" ]; then
  git config --global user.email "$GIT_USER_EMAIL"
fi

echo "Verifying repository..."
git status

# Copy user's env file into workspace if it exists
if [ -f "/tmp/.env" ]; then
  echo "Copying env file to workspace..."
  cp /tmp/.env .env
fi

# Override an environment variable from a state file if the file exists and is non-empty
# Usage: override_from_file <file_path> <var_name>
override_from_file() {
  local file="$1" var="$2"
  if [ -f "$file" ]; then
    local value
    value=$(cat "$file")
    if [ -n "$value" ]; then
      export "$var=$value"
    fi
  fi
}

# Check for config overrides from the state directory (written by spin on restart)
override_from_file "/state/prompt.txt" "PROMPT"
override_from_file "/state/max-iterations.txt" "MAX_ITERATIONS"
override_from_file "/state/model.txt" "ANTHROPIC_MODEL"

# If PROMPT is set, run the iteration loop
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
  echo "Starting autonomous implementation loop..."

  # Execute spinner exec command
  spinner exec
else
  echo ""
  echo "Repository cloned successfully. Container is ready."

  # Keep container running
  tail -f /dev/null
fi
