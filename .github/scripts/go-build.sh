#!/bin/bash
set -e

oldPWD="$(pwd)"

dirs=("./actions" "./")

for dir in "${dirs[@]}"; do
    echo "Building $dir"
    cd "$dir"
    go build ./...
    cd "$oldPWD"
done
