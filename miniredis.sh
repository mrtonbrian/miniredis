#!/bin/sh

set -e # Exit early if any commands fail

(
  cd "$(dirname "$0")" # Ensure compile steps are run within the repository directory
  go build -o /tmp/miniredis ./cmd/server
)

exec /tmp/miniredis "$@"