#!/bin/bash

set -e  # Exit on error

# Validate required environment variables
if [ -z "$SPINNER_PROJECT" ]; then
  echo "Error: SPINNER_PROJECT environment variable is required"
  exit 1
fi

if [ -z "$SPINNER_ZONE" ]; then
  echo "Error: SPINNER_ZONE environment variable is required"
  exit 1
fi

if [ -z "$SPINNER_STATE_BUCKET" ]; then
  echo "Error: SPINNER_STATE_BUCKET environment variable is required"
  exit 1
fi

echo "GCP Configuration:"
echo "  Project: $SPINNER_PROJECT"
echo "  Zone: $SPINNER_ZONE"
echo "  Bucket: $SPINNER_STATE_BUCKET"
echo ""

echo "Setting up dev environment..."
./scripts/dev-setup.sh

echo ""
echo "Running setup with GCP backend..."
./dist/spinner setup \
  --name default \
  --backend gcp

echo ""
echo "Running spin with watch on GCP..."
./dist/spinner spin \
  --image default \
  --branch test-watch \
  --repo https://github.com/rickihastings/spinner.git \
  --prompt "1+1 then print ~~ FEATURE_COMPLETED ~~" \
  --backend gcp \
  --watch