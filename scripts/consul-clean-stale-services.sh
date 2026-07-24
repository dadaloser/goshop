#!/usr/bin/env bash
set -euo pipefail

CONSUL_HTTP_ADDR="${CONSUL_HTTP_ADDR:-http://192.168.1.92:8500}"

services=(
  goshop-user-srv
  goshop-goods-srv
  goshop-inventory-srv
  goshop-order-srv
  goshop-review-srv
  goshop-api
  goshop-admin
)

for service in "${services[@]}"; do
  echo "==> scanning ${service}"
  entries="$(curl -fsS "${CONSUL_HTTP_ADDR}/v1/catalog/service/${service}" \
    | python3 -c 'import json, sys
for item in json.load(sys.stdin):
    print(item.get("Node", ""), item.get("ServiceID", ""), sep="\t")
')"

  if [[ -z "${entries}" ]]; then
    echo "    no instances"
    continue
  fi

  while IFS=$'\t' read -r node id; do
    [[ -z "${node}" ]] && continue
    [[ -z "${id}" ]] && continue
    echo "    deregister node=${node} service_id=${id}"
    curl -fsS -X PUT \
      --data "{\"Node\":\"${node}\",\"ServiceID\":\"${id}\"}" \
      "${CONSUL_HTTP_ADDR}/v1/catalog/deregister" >/dev/null
  done <<< "${entries}"
done

echo "done"
