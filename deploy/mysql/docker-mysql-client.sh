#!/usr/bin/env sh
# Optional mysql-client adapter for a local Docker-only MySQL installation.
# MYSQL_PWD is inherited from initialize.sh and passed only into the container.
set -eu
: "${MYSQL_DOCKER_CONTAINER:=mysql-server}"
exec docker exec -i -e MYSQL_PWD="${MYSQL_PWD:-}" "${MYSQL_DOCKER_CONTAINER}" mysql "$@"
