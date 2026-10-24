#!/usr/bin/env bash
set -euo pipefail

ID=${1:?"dashboard id required"}
OUT=${2:-"/tmp/dd_dashboard_${ID}.json"}

: "${DD_API_KEY:?DD_API_KEY required}"
: "${DD_APP_KEY:?DD_APP_KEY required}"
DD_SITE=${DD_SITE:-datadoghq.com}

curl -s "https://api.${DD_SITE}/api/v1/dashboard/${ID}" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  > "${OUT}"

echo "${OUT}"
