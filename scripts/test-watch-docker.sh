#!/bin/bash

set -e  # Exit on error

echo "Building..."
make build

echo ""
echo "Running setup..."
./dist/spinner setup --name default --dockerfile Dockerfile

echo ""
echo "Running spin with watch..."
./dist/spinner spin \
  --image spinner:default \
  --branch test-watch \
  --repo https://github.com/rickihastings/spinner.git \
  --prompt "1+1 then print ~~ FEATURE_COMPLETED ~~" \
  --watch
