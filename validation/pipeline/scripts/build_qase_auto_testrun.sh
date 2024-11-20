#!/bin/bash
set -e
cd $(dirname $0)/../../../../../

  echo "building qase auto testrun bin"
  env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o validation/testrun ./validation/pipeline/qase/testrun