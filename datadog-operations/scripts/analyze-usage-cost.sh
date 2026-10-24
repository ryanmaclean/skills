#!/bin/bash
set -e

# Analyze Datadog usage and costs for FinOps optimization
# Queries usage/billing APIs to provide cost analysis and recommendations

# Check required environment variables
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo "[ERROR] DD_API_KEY and DD_APP_KEY must be set" >&2
    exit 1
fi

DD_SITE=${DD_SITE:-datadoghq.com}

# Parse arguments
DURATION="30d"
PRODUCT="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --product)
            PRODUCT="$2"
            shift 2
            ;;
        *)
            echo "[ERROR] Unknown argument: $1" >&2
            echo "Usage: $0 [--duration 7d|30d|90d] [--product apm|logs|infrastructure|all]" >&2
            exit 1
            ;;
    esac
done

# Validate duration
case $DURATION in
    7d|30d|90d)
        ;;
    *)
        echo "[ERROR] Invalid duration. Use: 7d, 30d, or 90d" >&2
        exit 1
        ;;
esac

# Validate product
case $PRODUCT in
    apm|logs|infrastructure|all)
        ;;
    *)
        echo "[ERROR] Invalid product. Use: apm, logs, infrastructure, or all" >&2
        exit 1
        ;;
esac

# Calculate time range (Datadog usage API uses YYYY-MM-DD format)
case $DURATION in
    7d)
        DAYS_AGO=7
        ;;
    30d)
        DAYS_AGO=30
        ;;
    90d)
        DAYS_AGO=90
        ;;
esac

# Calculate start and end dates
if date -v -1d > /dev/null 2>&1; then
    # BSD date (macOS)
    START_DATE=$(date -v-${DAYS_AGO}d +%Y-%m-%d)
    END_DATE=$(date +%Y-%m-%d)
else
    # GNU date (Linux)
    START_DATE=$(date -d "${DAYS_AGO} days ago" +%Y-%m-%d)
    END_DATE=$(date +%Y-%m-%d)
fi

echo "[INFO] Analyzing Datadog usage and costs" >&2
echo "[INFO] Duration: $DURATION ($START_DATE to $END_DATE)" >&2
echo "[INFO] Product filter: $PRODUCT" >&2
echo "" >&2

# Initialize results
APM_DATA="{}"
LOGS_DATA="{}"
INFRA_DATA="{}"
METRICS_DATA="{}"

# Function to query usage API with error handling
query_usage_api() {
    local endpoint=$1
    local response=$(curl -s -w "\n%{http_code}" -X GET "${endpoint}" \
      -H "DD-API-KEY: ${DD_API_KEY}" \
      -H "DD-APPLICATION-KEY: ${DD_APP_KEY}")

    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')

    if [ "$http_code" != "200" ]; then
        echo "[WARN] API returned HTTP $http_code for endpoint: $endpoint" >&2
        echo "{}"
    else
        echo "$body"
    fi
}

# Query APM usage
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "apm" ]; then
    echo "[INFO] Querying APM usage..." >&2

    # APM Hosts usage
    APM_HOSTS_RESPONSE=$(query_usage_api "https://api.${DD_SITE}/api/v1/usage/hosts?start_hr=${START_DATE}T00&end_hr=${END_DATE}T00")

    # APM spans (ingested and indexed)
    APM_TRACES_RESPONSE=$(query_usage_api "https://api.${DD_SITE}/api/v1/usage/traces?start_hr=${START_DATE}T00&end_hr=${END_DATE}T00")

    # Calculate APM metrics
    if [ "$APM_HOSTS_RESPONSE" != "{}" ]; then
        APM_HOST_HOURS=$(echo "$APM_HOSTS_RESPONSE" | jq '[.usage[] | select(.apm_host_count != null) | .apm_host_count] | add // 0')
        APM_AVG_HOSTS=$(echo "$APM_HOSTS_RESPONSE" | jq '[.usage[] | select(.apm_host_count != null) | .apm_host_count] | if length > 0 then (add / length | floor) else 0 end')
    else
        APM_HOST_HOURS=0
        APM_AVG_HOSTS=0
    fi

    if [ "$APM_TRACES_RESPONSE" != "{}" ]; then
        APM_INGESTED_SPANS=$(echo "$APM_TRACES_RESPONSE" | jq '[.usage[] | select(.ingested_events_bytes != null) | .ingested_events_bytes] | add // 0')
        APM_INDEXED_SPANS=$(echo "$APM_TRACES_RESPONSE" | jq '[.usage[] | select(.indexed_events_count != null) | .indexed_events_count] | add // 0')
        APM_INGESTED_GB=$(echo "scale=2; $APM_INGESTED_SPANS / 1073741824" | bc)
    else
        APM_INGESTED_SPANS=0
        APM_INDEXED_SPANS=0
        APM_INGESTED_GB="0.00"
    fi

    # Estimated APM costs (based on 2026 pricing)
    # APM host: $31/month, Indexed spans: $1.70 per million
    APM_HOST_COST=$(echo "scale=2; ($APM_AVG_HOSTS * 31)" | bc)
    APM_SPAN_COST=$(echo "scale=2; ($APM_INDEXED_SPANS / 1000000 * 1.70)" | bc)
    APM_TOTAL_COST=$(echo "scale=2; $APM_HOST_COST + $APM_SPAN_COST" | bc)

    APM_DATA=$(cat <<EOF
{
  "host_hours": $APM_HOST_HOURS,
  "avg_hosts": $APM_AVG_HOSTS,
  "ingested_spans_bytes": $APM_INGESTED_SPANS,
  "ingested_spans_gb": $APM_INGESTED_GB,
  "indexed_spans": $APM_INDEXED_SPANS,
  "estimated_cost_usd": {
    "hosts": $APM_HOST_COST,
    "indexed_spans": $APM_SPAN_COST,
    "total": $APM_TOTAL_COST
  }
}
EOF
)

    echo "[INFO] APM - Avg hosts: $APM_AVG_HOSTS, Indexed spans: $APM_INDEXED_SPANS, Est. cost: \$${APM_TOTAL_COST}" >&2
fi

# Query Logs usage
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "logs" ]; then
    echo "[INFO] Querying logs usage..." >&2

    LOGS_RESPONSE=$(query_usage_api "https://api.${DD_SITE}/api/v1/usage/logs?start_hr=${START_DATE}T00&end_hr=${END_DATE}T00")

    if [ "$LOGS_RESPONSE" != "{}" ]; then
        LOGS_INGESTED_BYTES=$(echo "$LOGS_RESPONSE" | jq '[.usage[] | select(.ingested_events_bytes != null) | .ingested_events_bytes] | add // 0')
        LOGS_INDEXED_BYTES=$(echo "$LOGS_RESPONSE" | jq '[.usage[] | select(.indexed_events_count != null) | .indexed_events_count] | add // 0')
        LOGS_INGESTED_GB=$(echo "scale=2; $LOGS_INGESTED_BYTES / 1073741824" | bc)
        LOGS_INDEXED_GB=$(echo "scale=2; $LOGS_INDEXED_BYTES / 1073741824" | bc)
    else
        LOGS_INGESTED_BYTES=0
        LOGS_INDEXED_BYTES=0
        LOGS_INGESTED_GB="0.00"
        LOGS_INDEXED_GB="0.00"
    fi

    # Estimated log costs (Ingested: $0.10/GB, Indexed: $1.70/million events)
    LOGS_INGESTED_COST=$(echo "scale=2; $LOGS_INGESTED_GB * 0.10" | bc)
    LOGS_INDEXED_COST=$(echo "scale=2; $LOGS_INDEXED_GB * 0.10" | bc)
    LOGS_TOTAL_COST=$(echo "scale=2; $LOGS_INGESTED_COST + $LOGS_INDEXED_COST" | bc)

    LOGS_DATA=$(cat <<EOF
{
  "ingested_bytes": $LOGS_INGESTED_BYTES,
  "ingested_gb": $LOGS_INGESTED_GB,
  "indexed_bytes": $LOGS_INDEXED_BYTES,
  "indexed_gb": $LOGS_INDEXED_GB,
  "estimated_cost_usd": {
    "ingested": $LOGS_INGESTED_COST,
    "indexed": $LOGS_INDEXED_COST,
    "total": $LOGS_TOTAL_COST
  }
}
EOF
)

    echo "[INFO] Logs - Ingested: ${LOGS_INGESTED_GB}GB, Indexed: ${LOGS_INDEXED_GB}GB, Est. cost: \$${LOGS_TOTAL_COST}" >&2
fi

# Query Infrastructure usage
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "infrastructure" ]; then
    echo "[INFO] Querying infrastructure usage..." >&2

    INFRA_RESPONSE=$(query_usage_api "https://api.${DD_SITE}/api/v1/usage/hosts?start_hr=${START_DATE}T00&end_hr=${END_DATE}T00")

    if [ "$INFRA_RESPONSE" != "{}" ]; then
        INFRA_HOST_HOURS=$(echo "$INFRA_RESPONSE" | jq '[.usage[] | select(.host_count != null) | .host_count] | add // 0')
        INFRA_AVG_HOSTS=$(echo "$INFRA_RESPONSE" | jq '[.usage[] | select(.host_count != null) | .host_count] | if length > 0 then (add / length | floor) else 0 end')
        CONTAINER_HOURS=$(echo "$INFRA_RESPONSE" | jq '[.usage[] | select(.container_count != null) | .container_count] | add // 0')
        AVG_CONTAINERS=$(echo "$INFRA_RESPONSE" | jq '[.usage[] | select(.container_count != null) | .container_count] | if length > 0 then (add / length | floor) else 0 end')
    else
        INFRA_HOST_HOURS=0
        INFRA_AVG_HOSTS=0
        CONTAINER_HOURS=0
        AVG_CONTAINERS=0
    fi

    # Estimated infrastructure costs ($15/host/month, $1/container/month)
    INFRA_HOST_COST=$(echo "scale=2; ($INFRA_AVG_HOSTS * 15)" | bc)
    CONTAINER_COST=$(echo "scale=2; ($AVG_CONTAINERS * 1)" | bc)
    INFRA_TOTAL_COST=$(echo "scale=2; $INFRA_HOST_COST + $CONTAINER_COST" | bc)

    INFRA_DATA=$(cat <<EOF
{
  "host_hours": $INFRA_HOST_HOURS,
  "avg_hosts": $INFRA_AVG_HOSTS,
  "container_hours": $CONTAINER_HOURS,
  "avg_containers": $AVG_CONTAINERS,
  "estimated_cost_usd": {
    "hosts": $INFRA_HOST_COST,
    "containers": $CONTAINER_COST,
    "total": $INFRA_TOTAL_COST
  }
}
EOF
)

    echo "[INFO] Infrastructure - Avg hosts: $INFRA_AVG_HOSTS, Avg containers: $AVG_CONTAINERS, Est. cost: \$${INFRA_TOTAL_COST}" >&2
fi

# Query Custom Metrics usage
if [ "$PRODUCT" = "all" ]; then
    echo "[INFO] Querying custom metrics usage..." >&2

    METRICS_RESPONSE=$(query_usage_api "https://api.${DD_SITE}/api/v1/usage/timeseries?start_hr=${START_DATE}T00&end_hr=${END_DATE}T00")

    if [ "$METRICS_RESPONSE" != "{}" ]; then
        CUSTOM_METRICS=$(echo "$METRICS_RESPONSE" | jq '[.usage[] | select(.num_custom_timeseries != null) | .num_custom_timeseries] | if length > 0 then (add / length | floor) else 0 end')
    else
        CUSTOM_METRICS=0
    fi

    # Estimated custom metrics cost ($0.05 per metric per month for first 100)
    METRICS_COST=$(echo "scale=2; ($CUSTOM_METRICS * 0.05)" | bc)

    METRICS_DATA=$(cat <<EOF
{
  "avg_custom_metrics": $CUSTOM_METRICS,
  "estimated_cost_usd": $METRICS_COST
}
EOF
)

    echo "[INFO] Custom metrics - Avg: $CUSTOM_METRICS, Est. cost: \$${METRICS_COST}" >&2
fi

echo "" >&2

# Generate recommendations based on usage patterns
echo "[INFO] Generating cost optimization recommendations..." >&2
echo "" >&2

RECOMMENDATIONS="[]"

# APM recommendations
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "apm" ]; then
    if [ "$APM_INDEXED_SPANS" -gt 0 ] && [ "$APM_INGESTED_SPANS" -gt 0 ]; then
        RETENTION_RATE=$(echo "scale=4; $APM_INDEXED_SPANS / ($APM_INGESTED_SPANS / 1000)" | bc)
        RETENTION_PCT=$(echo "scale=2; $RETENTION_RATE * 100" | bc)

        if (( $(echo "$RETENTION_RATE > 0.15" | bc -l) )); then
            RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
                "category": "apm",
                "priority": "high",
                "issue": "High APM span retention rate",
                "detail": "Retaining '"$RETENTION_PCT"'% of ingested spans. Industry standard is 10-15%.",
                "recommendation": "Review tag-based retention filters to reduce indexed spans. Focus on high-value traces (errors, slow requests) and sample normal traffic.",
                "potential_savings_usd": '"$(echo "scale=2; $APM_SPAN_COST * 0.30" | bc)"'
            }]')
        fi
    fi

    if [ "$APM_AVG_HOSTS" -gt 10 ]; then
        RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
            "category": "apm",
            "priority": "medium",
            "issue": "APM enabled on '"$APM_AVG_HOSTS"' hosts",
            "detail": "Verify all hosts require APM monitoring.",
            "recommendation": "Disable APM on non-production hosts, batch jobs, and internal services. Consider host-based pricing tier for large deployments.",
            "potential_savings_usd": '"$(echo "scale=2; $APM_HOST_COST * 0.20" | bc)"'
        }]')
    fi
fi

# Logs recommendations
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "logs" ]; then
    if (( $(echo "$LOGS_INGESTED_GB > 100" | bc -l) )); then
        RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
            "category": "logs",
            "priority": "high",
            "issue": "High log ingestion volume",
            "detail": "Ingesting '"$LOGS_INGESTED_GB"'GB of logs per '"$DURATION"'.",
            "recommendation": "Implement log filtering at source. Exclude debug logs, health checks, and high-frequency events. Use log patterns to identify noisy services.",
            "potential_savings_usd": '"$(echo "scale=2; $LOGS_TOTAL_COST * 0.40" | bc)"'
        }]')
    fi

    if [ "$LOGS_INDEXED_BYTES" -gt 0 ] && [ "$LOGS_INGESTED_BYTES" -gt 0 ]; then
        INDEXED_RATIO=$(echo "scale=4; $LOGS_INDEXED_BYTES / $LOGS_INGESTED_BYTES" | bc)

        if (( $(echo "$INDEXED_RATIO > 0.30" | bc -l) )); then
            INDEXED_PCT=$(echo "scale=2; $INDEXED_RATIO * 100" | bc)
            RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
                "category": "logs",
                "priority": "medium",
                "issue": "High log indexing rate",
                "detail": "Indexing '"$INDEXED_PCT"'% of ingested logs.",
                "recommendation": "Review log indexing rules. Index only logs needed for search/alerting. Use archives for compliance and historical analysis.",
                "potential_savings_usd": '"$(echo "scale=2; $LOGS_INDEXED_COST * 0.25" | bc)"'
            }]')
        fi
    fi
fi

# Infrastructure recommendations
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "infrastructure" ]; then
    if [ "$AVG_CONTAINERS" -gt 50 ]; then
        RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
            "category": "infrastructure",
            "priority": "medium",
            "issue": "High container count",
            "detail": "Monitoring '"$AVG_CONTAINERS"' containers on average.",
            "recommendation": "Exclude ephemeral and test containers. Use container exclusion rules for short-lived workloads. Consider Fargate pricing for containerized workloads.",
            "potential_savings_usd": '"$(echo "scale=2; $CONTAINER_COST * 0.15" | bc)"'
        }]')
    fi
fi

# Custom metrics recommendations
if [ "$PRODUCT" = "all" ] && [ "$CUSTOM_METRICS" -gt 100 ]; then
    RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
        "category": "metrics",
        "priority": "low",
        "issue": "High custom metrics count",
        "detail": "Using '"$CUSTOM_METRICS"' custom metrics.",
        "recommendation": "Audit custom metrics for unused or redundant metrics. Consolidate similar metrics. Use metric tags instead of separate metrics where possible.",
        "potential_savings_usd": '"$(echo "scale=2; $METRICS_COST * 0.10" | bc)"'
    }]')
fi

# Add general recommendations
RECOMMENDATIONS=$(echo "$RECOMMENDATIONS" | jq '. += [{
    "category": "general",
    "priority": "info",
    "issue": "Cost visibility",
    "detail": "Regular cost analysis enables proactive optimization.",
    "recommendation": "Schedule weekly cost reviews. Set up usage alerts for anomaly detection. Use Datadog Cost Management dashboards.",
    "potential_savings_usd": 0
}]')

RECOMMENDATIONS_COUNT=$(echo "$RECOMMENDATIONS" | jq 'length')
echo "[INFO] Generated $RECOMMENDATIONS_COUNT recommendations" >&2

# Calculate total estimated costs
TOTAL_COST=0
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "apm" ]; then
    TOTAL_COST=$(echo "scale=2; $TOTAL_COST + $APM_TOTAL_COST" | bc)
fi
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "logs" ]; then
    TOTAL_COST=$(echo "scale=2; $TOTAL_COST + $LOGS_TOTAL_COST" | bc)
fi
if [ "$PRODUCT" = "all" ] || [ "$PRODUCT" = "infrastructure" ]; then
    TOTAL_COST=$(echo "scale=2; $TOTAL_COST + $INFRA_TOTAL_COST" | bc)
fi
if [ "$PRODUCT" = "all" ]; then
    TOTAL_COST=$(echo "scale=2; $TOTAL_COST + $METRICS_COST" | bc)
fi

POTENTIAL_SAVINGS=$(echo "$RECOMMENDATIONS" | jq '[.[].potential_savings_usd] | add // 0')

echo "" >&2
echo "[SUMMARY] Total estimated cost: \$${TOTAL_COST} per month" >&2
echo "[SUMMARY] Potential savings: \$${POTENTIAL_SAVINGS} per month" >&2
echo "" >&2

# Show top recommendations
echo "[RECOMMENDATIONS] Top cost optimization opportunities:" >&2
echo "$RECOMMENDATIONS" | jq -r '.[] | select(.priority == "high") | "  [\(.priority | ascii_upcase)] \(.issue) - Potential savings: $\(.potential_savings_usd)"' >&2
echo "" >&2

# Output structured JSON
cat <<EOF
{
  "status": "ok",
  "analysis_period": {
    "start_date": "$START_DATE",
    "end_date": "$END_DATE",
    "duration": "$DURATION"
  },
  "product_filter": "$PRODUCT",
  "usage_summary": {
    "apm": $APM_DATA,
    "logs": $LOGS_DATA,
    "infrastructure": $INFRA_DATA,
    "custom_metrics": $METRICS_DATA
  },
  "cost_summary": {
    "total_estimated_monthly_usd": $TOTAL_COST,
    "potential_savings_usd": $POTENTIAL_SAVINGS,
    "optimization_opportunity_pct": $(echo "scale=2; ($POTENTIAL_SAVINGS / $TOTAL_COST) * 100" | bc 2>/dev/null || echo "0")
  },
  "recommendations": $RECOMMENDATIONS,
  "next_steps": [
    "Review high-priority recommendations for immediate cost reduction",
    "Implement APM sampling for high-volume services",
    "Configure log exclusion filters for noisy patterns",
    "Audit and remove unused custom metrics",
    "Set up cost anomaly alerts in Datadog"
  ]
}
EOF
