#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

# These tests exercise local registration/discovery, TLS, readiness, graceful
# shutdown, and resolver recovery without requiring an external Consul cluster.
env GOCACHE="${GOCACHE:-/tmp/goshop-gocache}" go test -count=1 -race \
  ./gmicro/app \
  ./gmicro/registry/consul \
  ./gmicro/server/restserver \
  ./gmicro/server/rpcserver \
  ./gmicro/server/rpcserver/resolver/discovery
