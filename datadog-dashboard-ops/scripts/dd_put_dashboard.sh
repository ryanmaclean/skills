#!/usr/bin/env bash
set -euo pipefail

ID=${1:?"dashboard id required"}
FILE=${2:?"json file required"}

: "${DD_API_KEY:?DD_API_KEY required}"
: "${DD_APP_KEY:?DD_APP_KEY required}"
DD_SITE=${DD_SITE:-datadoghq.com}

curl -s -X PUT "https://api.${DD_SITE}/api/v1/dashboard/${ID}" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -H "Content-Type: application/json" \
  --data @"${FILE}" | jq -r '.id'
