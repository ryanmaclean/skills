#!/usr/bin/env bash
set -euo pipefail

: "${DD_API_KEY:?DD_API_KEY required}"
: "${DD_APP_KEY:?DD_APP_KEY required}"
DD_SITE=${DD_SITE:-datadoghq.com}
FROM=${1:-$(($(date +%s)-86400))}

curl -s "https://api.${DD_SITE}/api/v1/metrics?from=${FROM}" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" | jq -r '.metrics[]'
