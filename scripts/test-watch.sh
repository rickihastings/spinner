#!/bin/bash

set -e  # Exit on error

echo "Building spinner..."
go build -o dist/spinner

echo ""
echo "Setting up environment..."
./dist/spinner setup --name default --dockerfile Dockerfile

echo ""
echo "Running spin with watch..."
./dist/spinner spin \
  --image spinner:default \
  --branch claude/abstract-claude-layer-xel91 \
  --repo https://github.com/rickihastings/spinner.git \
  --prompt "1+1 then print ~~ FEATURE_COMPLETED ~~" \
  --watch
