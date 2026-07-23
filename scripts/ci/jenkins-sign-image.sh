#!/usr/bin/env bash
set -euo pipefail
image="${1:?image reference required}"
artifact="${2:?artifact prefix required}"
: "${COSIGN_KEY:?COSIGN_KEY is required}"
: "${COSIGN_PASSWORD:?COSIGN_PASSWORD is required}"
command -v cosign >/dev/null || { echo "cosign is required" >&2; exit 1; }
digest="$(docker inspect --format='{{index .RepoDigests 0}}' "$image")"
test -n "$digest"
cosign sign --yes --key "$COSIGN_KEY" "$digest"
cosign attest --yes --key "$COSIGN_KEY" --type spdxjson --predicate "${artifact}.spdx.json" "$digest"
printf '%s\n' "$digest" > "${artifact}.digest"
