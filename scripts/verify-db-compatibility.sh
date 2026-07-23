#!/usr/bin/env bash
set -euo pipefail

: "${OLD_IMAGE:?set OLD_IMAGE to the previous signed image digest}"
: "${NEW_IMAGE:?set NEW_IMAGE to the candidate signed image digest}"
: "${MIGRATION_UP_CMD:?set MIGRATION_UP_CMD to the reviewed expand migration command}"
: "${SMOKE_CMD:?set SMOKE_CMD to a command that executes the synthetic checkout}"

for image in "$OLD_IMAGE" "$NEW_IMAGE"; do
  case "$image" in
    *@sha256:*) ;;
    *) echo "OLD_IMAGE and NEW_IMAGE must be immutable digests" >&2; exit 1 ;;
  esac
done

if git diff "${BASE_REF:-HEAD~1}" -- migrations/*up.sql | rg -i '^\+.*(drop[[:space:]]+(column|table)|rename[[:space:]]+column|modify[[:space:]]+column)'; then
  echo "destructive contract migration detected; split it into a later release" >&2
  exit 1
fi

eval "$MIGRATION_UP_CMD"
IMAGE="$OLD_IMAGE" eval "$SMOKE_CMD"
IMAGE="$NEW_IMAGE" eval "$SMOKE_CMD"
IMAGE="$OLD_IMAGE" eval "$SMOKE_CMD"

echo "expand migration is compatible with old -> new -> old application sequence"
