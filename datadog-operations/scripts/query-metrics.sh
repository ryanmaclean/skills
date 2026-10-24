#!/bin/bash
set -e

# Source monitoring library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/datadog-monitoring.sh" 2>/dev/null || true
init_monitoring "$(basename "$0")"
trap 'finalize_monitoring $?' EXIT INT TERM

# Query Datadog Metrics API for time series analysis
# Provides statistical analysis, trend detection, and anomaly identification

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse arguments
METRIC=""
SERVICE=""
DURATION="1h"
AGGREGATION="avg"

while [[ $# -gt 0 ]]; do
    case $1 in
        --metric)
            METRIC="$2"
            shift 2
            ;;
        --service)
            SERVICE="$2"
            shift 2
            ;;
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --aggregation)
            AGGREGATION="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if [ -z "$METRIC" ]; then
    echo "[ERROR] --metric is required" >&2
    echo "Examples: system.cpu.user, trace.express.request.duration, custom.metric" >&2
    exit 1
fi

# Validate aggregation
case $AGGREGATION in
    avg|sum|min|max)
        ;;
    *)
        echo "[ERROR] Invalid aggregation. Use: avg, sum, min, or max" >&2
        exit 1
        ;;
esac

# Convert duration to Unix timestamps
case $DURATION in
    1h)
        FROM=$(($(date +%s) - 3600))
        ;;
    24h)
        FROM=$(($(date +%s) - 86400))
        ;;
    7d)
        FROM=$(($(date +%s) - 604800))
        ;;
    30d)
        FROM=$(($(date +%s) - 2592000))
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 1h, 24h, 7d, or 30d" >&2
        exit 1
        ;;
esac

TO=$(date +%s)

# Build query string
QUERY="${AGGREGATION}:${METRIC}"
if [ -n "$SERVICE" ]; then
    QUERY="${QUERY}{service:${SERVICE}}"
fi

echo "[INFO] Querying metrics..." >&2
echo "[INFO] Metric: $METRIC" >&2
echo "[INFO] Service: ${SERVICE:-all}" >&2
echo "[INFO] Duration: $DURATION" >&2
echo "[INFO] Aggregation: $AGGREGATION" >&2
echo "" >&2

# Query Metrics API
start_operation "api_call"
RESPONSE=$(curl -s -X GET "https://api.${DD_SITE}/api/v1/query" \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -G \
  --data-urlencode "query=${QUERY}" \
  --data-urlencode "from=${FROM}" \
  --data-urlencode "to=${TO}")

# Check for API errors
if echo "$RESPONSE" | jq -e '.errors' > /dev/null 2>&1; then
    end_operation "error" "error_type:api_error"
    echo "[ERROR] API error:" >&2
    echo "$RESPONSE" | jq '.errors' >&2
    exit 1
fi

end_operation "ok"

# Check if data exists
if ! echo "$RESPONSE" | jq -e '.series' > /dev/null 2>&1; then
    echo "[WARN] No metric data found" >&2
    send_metric "datadog.skill.metrics.datapoints.count" 0 "status:no_data"
    cat <<EOF
{
  "status": "no_data",
  "metric": "$METRIC",
  "service": "${SERVICE:-all}",
  "duration": "$DURATION",
  "message": "No data points found for the specified metric and time range"
}
EOF
    exit 0
fi

# Check if series has data points
SERIES_COUNT=$(echo "$RESPONSE" | jq '.series | length')
if [ "$SERIES_COUNT" -eq 0 ]; then
    echo "[WARN] No time series data found" >&2
    send_metric "datadog.skill.metrics.datapoints.count" 0 "status:no_data"
    cat <<EOF
{
  "status": "no_data",
  "metric": "$METRIC",
  "service": "${SERVICE:-all}",
  "duration": "$DURATION",
  "message": "Query returned empty series"
}
EOF
    exit 0
fi

echo "[INFO] Found $SERIES_COUNT time series" >&2
echo "" >&2

# Extract all data points from all series
ALL_POINTS=$(echo "$RESPONSE" | jq -c '[.series[].pointlist[][] | select(. != null)]')
POINT_COUNT=$(echo "$ALL_POINTS" | jq 'length')

if [ "$POINT_COUNT" -eq 0 ]; then
    echo "[WARN] No data points in time series" >&2
    send_metric "datadog.skill.metrics.datapoints.count" 0 "status:no_data"
    cat <<EOF
{
  "status": "no_data",
  "metric": "$METRIC",
  "service": "${SERVICE:-all}",
  "duration": "$DURATION",
  "message": "Time series contains no data points"
}
EOF
    exit 0
fi

echo "[INFO] Processing $POINT_COUNT data points" >&2

# Calculate statistics using jq
STATS=$(echo "$ALL_POINTS" | jq '{
  min: (min),
  max: (max),
  avg: (add / length),
  sum: (add),
  count: length,
  p50: (sort | .[length / 2 | floor]),
  p95: (sort | .[length * 0.95 | floor]),
  p99: (sort | .[length * 0.99 | floor])
}')

MIN=$(echo "$STATS" | jq '.min')
MAX=$(echo "$STATS" | jq '.max')
AVG=$(echo "$STATS" | jq '.avg')
P50=$(echo "$STATS" | jq '.p50')
P95=$(echo "$STATS" | jq '.p95')
P99=$(echo "$STATS" | jq '.p99')

echo "[STATS] Min: $MIN" >&2
echo "[STATS] Max: $MAX" >&2
echo "[STATS] Avg: $AVG" >&2
echo "[STATS] P50: $P50" >&2
echo "[STATS] P95: $P95" >&2
echo "[STATS] P99: $P99" >&2
echo "" >&2

# Calculate standard deviation
STDDEV=$(echo "$ALL_POINTS" | jq --argjson avg "$AVG" '
  map(. - $avg | . * .) | add / length | sqrt
')

echo "[STATS] Standard Deviation: $STDDEV" >&2
echo "" >&2

# Trend analysis - compare first half vs second half
HALF_POINT=$((POINT_COUNT / 2))
FIRST_HALF_AVG=$(echo "$ALL_POINTS" | jq --argjson half "$HALF_POINT" '
  .[:$half] | add / length
')
SECOND_HALF_AVG=$(echo "$ALL_POINTS" | jq --argjson half "$HALF_POINT" '
  .[$half:] | add / length
')

# Calculate trend percentage change
TREND_PCT=$(echo "$FIRST_HALF_AVG $SECOND_HALF_AVG" | jq -n --argjson first "$(cat -)" --argjson second "$(cat -)" '
  if $first == 0 then 0 else (($second - $first) / $first * 100) end
' 2>/dev/null || echo "0")

# Determine trend direction
if (( $(echo "$TREND_PCT > 10" | bc -l 2>/dev/null || echo "0") )); then
    TREND="increasing"
    TREND_STATUS="warning"
elif (( $(echo "$TREND_PCT < -10" | bc -l 2>/dev/null || echo "0") )); then
    TREND="decreasing"
    TREND_STATUS="improving"
else
    TREND="stable"
    TREND_STATUS="normal"
fi

echo "[TREND] Direction: $TREND" >&2
echo "[TREND] Change: ${TREND_PCT}%" >&2
echo "[TREND] Status: $TREND_STATUS" >&2
echo "" >&2

# Anomaly detection - find points > 2 standard deviations from mean
UPPER_THRESHOLD=$(echo "$AVG $STDDEV" | jq -n '
  input + (input * 2)
')
LOWER_THRESHOLD=$(echo "$AVG $STDDEV" | jq -n '
  input - (input * 2)
')

ANOMALIES=$(echo "$ALL_POINTS" | jq --argjson upper "$UPPER_THRESHOLD" --argjson lower "$LOWER_THRESHOLD" '
  [.[] | select(. > $upper or . < $lower)]
')

ANOMALY_COUNT=$(echo "$ANOMALIES" | jq 'length')
ANOMALY_PCT=$(echo "$ANOMALY_COUNT $POINT_COUNT" | jq -n '
  (input / input * 100)
')

echo "[ANOMALY] Detected: $ANOMALY_COUNT points (${ANOMALY_PCT}%)" >&2
echo "[ANOMALY] Threshold: $LOWER_THRESHOLD to $UPPER_THRESHOLD" >&2

if [ "$ANOMALY_COUNT" -gt 0 ]; then
    echo "[WARN] Found anomalous data points exceeding 2 standard deviations" >&2
    ANOMALY_STATUS="detected"
else
    echo "[INFO] No anomalies detected" >&2
    ANOMALY_STATUS="none"
fi

echo "" >&2

# Extract time series data with timestamps
TIME_SERIES=$(echo "$RESPONSE" | jq '[.series[].pointlist[] | {
  timestamp: .[0],
  value: .[1]
}]')

# Get metadata
UNIT=$(echo "$RESPONSE" | jq -r '.series[0].unit // "unknown"')
SCOPE=$(echo "$RESPONSE" | jq -r '.series[0].scope // "unknown"')

# Determine overall status
if [ "$ANOMALY_COUNT" -gt $(($POINT_COUNT / 10)) ]; then
    OVERALL_STATUS="critical"
elif [ "$ANOMALY_COUNT" -gt 0 ] || [ "$TREND" = "increasing" ]; then
    OVERALL_STATUS="warning"
else
    OVERALL_STATUS="ok"
fi

echo "[RESULT] Overall status: $OVERALL_STATUS" >&2
echo "" >&2

# Send metrics
send_metric "datadog.skill.metrics.datapoints.count" "$POINT_COUNT" "status:ok"
send_metric "datadog.skill.metrics.anomalies.count" "$ANOMALY_COUNT" "status:ok"

# Output structured JSON
cat <<EOF
{
  "status": "$OVERALL_STATUS",
  "metadata": {
    "metric": "$METRIC",
    "service": "${SERVICE:-all}",
    "duration": "$DURATION",
    "aggregation": "$AGGREGATION",
    "unit": "$UNIT",
    "scope": "$SCOPE",
    "query": "$QUERY"
  },
  "statistics": {
    "count": $POINT_COUNT,
    "min": $MIN,
    "max": $MAX,
    "avg": $AVG,
    "p50": $P50,
    "p95": $P95,
    "p99": $P99,
    "stddev": $STDDEV
  },
  "trend": {
    "direction": "$TREND",
    "change_percent": $TREND_PCT,
    "status": "$TREND_STATUS",
    "first_half_avg": $FIRST_HALF_AVG,
    "second_half_avg": $SECOND_HALF_AVG
  },
  "anomalies": {
    "status": "$ANOMALY_STATUS",
    "count": $ANOMALY_COUNT,
    "percentage": $ANOMALY_PCT,
    "threshold_lower": $LOWER_THRESHOLD,
    "threshold_upper": $UPPER_THRESHOLD,
    "detected_values": $ANOMALIES
  },
  "time_series": $TIME_SERIES
}
EOF
