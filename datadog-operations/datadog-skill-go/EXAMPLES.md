# Datadog CLI - Examples & Recipes

**Real-world scenarios and complete workflows**

---

## Table of Contents

1. [Service Health Monitoring](#service-health-monitoring)
2. [Incident Investigation](#incident-investigation)
3. [SLO Management](#slo-management)
4. [Cost Optimization](#cost-optimization)
5. [Deployment Validation](#deployment-validation)
6. [Performance Analysis](#performance-analysis)
7. [Log Analysis](#log-analysis)
8. [Predictive Operations](#predictive-operations)
9. [CI/CD Integration](#cicd-integration)
10. [Multi-Service Correlation](#multi-service-correlation)
11. [Foundation Commands](#foundation-commands)

---

## Service Health Monitoring

### Scenario: Daily Health Check for Production Services

**Goal**: Check health of all production services and alert if issues found.

```bash
#!/bin/bash
# daily-health-check.sh

# Set environment
export DD_API_KEY="your-api-key"
export DD_APP_KEY="your-app-key"

# Get all services
services=$(dd apm services --env production --from 1h --json | jq -r '.services[].name')

# Check health for each service
for service in $services; do
  echo "Checking $service..."

  # Get health status
  health=$(dd health check --service "$service" --json)
  status=$(echo "$health" | jq -r '.status')

  if [ "$status" != "healthy" ]; then
    echo "❌ $service is $status"

    # Get detailed metrics
    dd apm services --service "$service" --from 1h
    dd logs search --query "error" --service "$service" --from 1h --limit 10
  else
    echo "✅ $service is healthy"
  fi
done
```

**Use in cron**:
```bash
# Run every hour
0 * * * * /path/to/daily-health-check.sh >> /var/log/dd-health.log 2>&1
```

---

## Incident Investigation

### Scenario: Investigate Service Outage

**Goal**: Quickly diagnose why a service is failing.

```bash
#!/bin/bash
# investigate-outage.sh <service-name>

SERVICE=$1

echo "=== Investigating $SERVICE ==="

# 1. Check current service status
echo -e "\n1. Service Health Status:"
dd health check --service "$SERVICE"

# 2. Recent error logs (last 15 minutes)
echo -e "\n2. Recent Errors:"
dd logs search --query "status:error" --service "$SERVICE" --from 15m --limit 20

# 3. Failed APM traces
echo -e "\n3. Failed Traces:"
dd apm traces --service "$SERVICE" --from 15m --filter "error:true" --limit 10

# 4. Service dependencies
echo -e "\n4. Service Dependencies:"
dd service-map query --service "$SERVICE" --json | jq '.dependencies'

# 5. Correlate with other incidents
echo -e "\n5. Related Issues:"
dd correlation analyze --service "$SERVICE" --from 15m

# 6. Check if deployment happened recently
echo -e "\n6. Recent Deployments:"
dd deploy list --service "$SERVICE" --from 1h

# 7. Anomaly detection
echo -e "\n7. Detected Anomalies:"
dd anomalies detect --service "$SERVICE" --from 30m

# Generate report
echo -e "\n=== Investigation Complete ==="
echo "Check output above for root cause indicators"
```

**Usage**:
```bash
./investigate-outage.sh api-gateway
```

---

## SLO Management

### Scenario: Monitor SLO Compliance and Alert on Budget Depletion

**Goal**: Track SLO error budgets and alert when < 10% remains.

```bash
#!/bin/bash
# slo-monitor.sh

# Get all SLOs
slos=$(dd slos list --json | jq -r '.slos[].id')

ALERTS=()

for slo_id in $slos; do
  # Get SLO details
  slo=$(dd slos get --id "$slo_id" --json)
  name=$(echo "$slo" | jq -r '.name')
  budget=$(echo "$slo" | jq -r '.error_budget_remaining')

  echo "SLO: $name - Budget Remaining: ${budget}%"

  # Alert if budget < 10%
  if (( $(echo "$budget < 10" | bc -l) )); then
    ALERTS+=("⚠️  $name has only ${budget}% error budget remaining!")

    # Get SLO history
    dd slo-history query --slo-id "$slo_id" --from 7d > "slo_${slo_id}_history.json"
  fi
done

# Send alerts if any
if [ ${#ALERTS[@]} -gt 0 ]; then
  echo -e "\n=== ALERTS ==="
  printf '%s\n' "${ALERTS[@]}"

  # Send to Slack/PagerDuty/etc
  # curl -X POST -H 'Content-type: application/json' \
  #   --data "{\"text\":\"$(printf '%s\n' "${ALERTS[@]}")\"}" \
  #   $SLACK_WEBHOOK_URL
fi
```

**Advanced: SLO Correction Window**

```bash
# Apply correction window for planned maintenance
dd slo-corrections create \
  --slo-id "abc123" \
  --category "Planned Maintenance" \
  --start "2026-01-25T02:00:00Z" \
  --end "2026-01-25T04:00:00Z" \
  --description "Database migration"
```

---

## Cost Optimization

### Scenario: Identify and Reduce High-Cost Services

**Goal**: Find services with high costs and get optimization recommendations.

```bash
#!/bin/bash
# cost-optimization.sh

echo "=== Cost Analysis ==="

# 1. Get monthly cost estimate
echo -e "\n1. Total Monthly Cost:"
dd cost estimate --from 1M

# 2. Get usage insights
echo -e "\n2. Usage Breakdown:"
dd usage-insights analyze --from 1M --json > usage.json

# 3. Find high-cost services
echo -e "\n3. Top 10 Expensive Services:"
cat usage.json | jq -r '.services | sort_by(-.cost) | .[0:10] | .[] | "\(.name): $\(.cost)"'

# 4. Get optimization recommendations
echo -e "\n4. Cost-Saving Recommendations:"
dd recommendations suggest --category cost --json > recommendations.json

# 5. Estimate savings
total_savings=$(cat recommendations.json | jq '[.recommendations[].estimated_savings] | add')
echo -e "\nPotential Monthly Savings: \$$total_savings"

# 6. Detailed recommendations
cat recommendations.json | jq -r '.recommendations[] | "- \(.title): $\(.estimated_savings) (\(.confidence)% confidence)"'

# Generate CSV report
echo -e "\n5. Generating CSV Report..."
echo "Service,Cost,Recommendation,Potential Savings" > cost-report.csv
cat recommendations.json | jq -r '.recommendations[] | "\(.service),\(.current_cost),\(.title),\(.estimated_savings)"' >> cost-report.csv

echo "Report saved to: cost-report.csv"
```

**Output Example**:
```
=== Cost Analysis ===
Total Monthly Cost: $12,450

Top 10 Expensive Services:
api-gateway: $4,200
user-service: $2,800
payment-processor: $1,900

Cost-Saving Recommendations:
- Reduce log retention: $450 (85% confidence)
- Optimize metric collection: $320 (78% confidence)
- Consolidate APM spans: $180 (92% confidence)

Potential Monthly Savings: $950
```

---

## Deployment Validation

### Scenario: Validate Deployment Safety Before Production Rollout

**Goal**: Check if deployment is safe based on health metrics.

```bash
#!/bin/bash
# validate-deployment.sh <service> <version>

SERVICE=$1
VERSION=$2

echo "=== Validating Deployment: $SERVICE v$VERSION ==="

# 1. Check current production health (baseline)
echo -e "\n1. Current Production Health:"
dd health check --service "$SERVICE" --env production

# 2. Check staging health (new version)
echo -e "\n2. Staging Health (v$VERSION):"
dd health check --service "$SERVICE" --env staging

# 3. Compare error rates
echo -e "\n3. Error Rate Comparison:"
prod_errors=$(dd apm services --service "$SERVICE" --env production --from 1h --json | jq -r '.services[0].error_rate')
staging_errors=$(dd apm services --service "$SERVICE" --env staging --from 1h --json | jq -r '.services[0].error_rate')

echo "Production Error Rate: ${prod_errors}%"
echo "Staging Error Rate: ${staging_errors}%"

# 4. Check for anomalies in staging
echo -e "\n4. Anomaly Detection (Staging):"
dd anomalies detect --service "$SERVICE" --env staging --from 1h

# 5. Run deployment safety check
echo -e "\n5. Deployment Safety Check:"
safety=$(dd deploy validate --service "$SERVICE" --version "$VERSION" --json)
safe=$(echo "$safety" | jq -r '.safe_to_deploy')

if [ "$safe" = "true" ]; then
  echo "✅ SAFE TO DEPLOY"
  exit 0
else
  echo "❌ NOT SAFE TO DEPLOY"
  echo "Reasons:"
  echo "$safety" | jq -r '.reasons[]'
  exit 1
fi
```

**Usage in CI/CD**:
```yaml
# .github/workflows/deploy.yml
- name: Validate Deployment
  run: |
    ./validate-deployment.sh api-gateway ${{ github.sha }}
    if [ $? -ne 0 ]; then
      echo "Deployment validation failed"
      exit 1
    fi

- name: Deploy to Production
  run: ./deploy.sh production
```

---

## Performance Analysis

### Scenario: Analyze Service Performance Degradation

**Goal**: Identify performance bottlenecks and slow endpoints.

```bash
#!/bin/bash
# performance-analysis.sh <service>

SERVICE=$1

echo "=== Performance Analysis: $SERVICE ==="

# 1. Get service metrics (last 24 hours)
echo -e "\n1. Service Metrics (24h):"
dd apm services --service "$SERVICE" --from 24h --json > metrics.json

# Extract key metrics
p95_latency=$(cat metrics.json | jq -r '.services[0].p95_latency')
p99_latency=$(cat metrics.json | jq -r '.services[0].p99_latency')
throughput=$(cat metrics.json | jq -r '.services[0].requests_per_second')

echo "P95 Latency: ${p95_latency}ms"
echo "P99 Latency: ${p99_latency}ms"
echo "Throughput: ${throughput} req/s"

# 2. Find slow traces
echo -e "\n2. Slowest Traces (P99):"
dd apm traces --service "$SERVICE" --from 24h --filter "duration:>1000" --limit 10

# 3. Database query analysis
echo -e "\n3. Slow Database Queries:"
dd database queries --service "$SERVICE" --from 24h --sort duration --limit 10

# 4. Detect performance anomalies
echo -e "\n4. Performance Anomalies:"
dd anomalies detect --metric latency --service "$SERVICE" --from 7d

# 5. Get optimization recommendations
echo -e "\n5. Performance Recommendations:"
dd recommendations suggest --service "$SERVICE" --category performance

# 6. Historical comparison
echo -e "\n6. Week-over-Week Comparison:"
current=$(dd metrics query --metric apm.service.hits --service "$SERVICE" --from 1d --json | jq '.average')
previous=$(dd metrics query --metric apm.service.hits --service "$SERVICE" --from 8d --to 7d --json | jq '.average')

change=$(echo "scale=2; (($current - $previous) / $previous) * 100" | bc)
echo "Traffic change: ${change}%"
```

---

## Log Analysis

### Scenario: Aggregate and Analyze Error Patterns

**Goal**: Find common error patterns and their frequency.

```bash
#!/bin/bash
# analyze-errors.sh <service> <hours>

SERVICE=$1
HOURS=${2:-24}

echo "=== Error Analysis: $SERVICE (last ${HOURS}h) ==="

# 1. Get all error logs
dd logs search --query "status:error" --service "$SERVICE" --from "${HOURS}h" --json > errors.json

# 2. Count total errors
total=$(cat errors.json | jq '.logs | length')
echo -e "\nTotal Errors: $total"

# 3. Group by error message
echo -e "\n3. Top Error Messages:"
cat errors.json | jq -r '.logs[].message' | sort | uniq -c | sort -rn | head -10

# 4. Group by status code
echo -e "\n4. Error by Status Code:"
cat errors.json | jq -r '.logs[].status_code' | sort | uniq -c | sort -rn

# 5. Error timeline (hourly buckets)
echo -e "\n5. Error Timeline:"
cat errors.json | jq -r '.logs[].timestamp' | cut -c1-13 | uniq -c

# 6. Find correlated errors across services
echo -e "\n6. Cross-Service Correlation:"
dd correlation analyze --service "$SERVICE" --from "${HOURS}h" --type error

# 7. Generate error report
cat > "error-report-${SERVICE}.txt" <<EOF
Error Report: $SERVICE
Generated: $(date)
Time Range: Last ${HOURS} hours

Total Errors: $total

Top Error Messages:
$(cat errors.json | jq -r '.logs[].message' | sort | uniq -c | sort -rn | head -5)

Recommendations:
$(dd recommendations suggest --service "$SERVICE" --category reliability)
EOF

echo -e "\nReport saved to: error-report-${SERVICE}.txt"
```

**Advanced: Real-time Error Monitoring**

```bash
# Stream errors in real-time
dd logs tail --query "status:error" --service api-gateway --follow | \
  while read line; do
    # Parse and alert on specific patterns
    if echo "$line" | grep -q "DatabaseTimeout"; then
      echo "🚨 Database timeout detected!"
      # Trigger auto-remediation
      dd auto-remediate execute --issue database_timeout --service api-gateway
    fi
  done
```

---

## Predictive Operations

### Scenario: Predict and Prevent Incidents

**Goal**: Use ML to predict potential issues before they occur.

```bash
#!/bin/bash
# predictive-ops.sh

echo "=== Predictive Operations Dashboard ==="

# 1. Train ML models on historical data
echo -e "\n1. Training ML Models..."
services=("api-gateway" "user-service" "payment-processor")

for service in "${services[@]}"; do
  echo "Training model for $service..."
  dd ml-insights train --service "$service" --from 30d
done

# 2. Predict incidents for next 24 hours
echo -e "\n2. Incident Predictions (24h horizon):"
predictions=$(dd predictions predict --target incidents --horizon 24h --json)

incident_count=$(echo "$predictions" | jq '.predictions | length')
echo "Predicted incidents: $incident_count"

if [ $incident_count -gt 0 ]; then
  echo -e "\n⚠️  Predicted Incidents:"
  echo "$predictions" | jq -r '.predictions[] | "- \(.service) at \(.predicted_time): \(.type) (confidence: \(.confidence)%)"'
fi

# 3. Capacity predictions
echo -e "\n3. Capacity Planning (7d horizon):"
dd capacity-scale predict --horizon 7d --json > capacity.json

echo "$capacity" | jq -r '.services[] | select(.needs_scaling == true) | "⚠️  \(.name) will need scaling on \(.scale_date)"'

# 4. Cost predictions
echo -e "\n4. Cost Forecast (30d):"
dd predictions predict --target cost --horizon 30d

# 5. Generate proactive recommendations
echo -e "\n5. Proactive Recommendations:"
dd recommendations suggest --json > recommendations.json

high_priority=$(cat recommendations.json | jq '.recommendations[] | select(.priority == "high")')

if [ -n "$high_priority" ]; then
  echo -e "\n🔴 High Priority Actions:"
  echo "$high_priority" | jq -r '"- \(.title) (ROI: \(.roi)x)"'
fi

# 6. Auto-remediation setup
echo -e "\n6. Setting up Auto-Remediation Rules:"
dd auto-remediate configure \
  --trigger "cpu_high" \
  --action "scale_up" \
  --service "api-gateway" \
  --threshold 80

echo "✅ Predictive monitoring configured"
```

---

## CI/CD Integration

### Scenario: Automated Deployment Pipeline with Datadog Validation

**Goal**: Integrate Datadog CLI into CI/CD for automated validation.

```yaml
# .github/workflows/deploy.yml
name: Deploy with Datadog Validation

on:
  push:
    branches: [main]

env:
  DD_API_KEY: ${{ secrets.DD_API_KEY }}
  DD_APP_KEY: ${{ secrets.DD_APP_KEY }}
  SERVICE_NAME: api-gateway

jobs:
  deploy:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Install Datadog CLI
        run: |
          curl -L -o dd https://github.com/your-org/datadog-cli/releases/download/v0.1.0/dd-linux-amd64
          chmod +x dd
          sudo mv dd /usr/local/bin/

      - name: Pre-Deployment Health Check
        run: |
          dd health check --service $SERVICE_NAME --env production

      - name: Check SLO Status
        run: |
          SLO_ID=$(dd slos list --service $SERVICE_NAME --json | jq -r '.slos[0].id')
          BUDGET=$(dd error-budgets check --slo-id $SLO_ID --json | jq -r '.remaining')

          if (( $(echo "$BUDGET < 5" | bc -l) )); then
            echo "❌ SLO error budget too low: ${BUDGET}%"
            exit 1
          fi

      - name: Deploy to Staging
        run: ./deploy.sh staging

      - name: Validate Staging Deployment
        run: |
          sleep 30  # Wait for deployment
          dd deploy validate --service $SERVICE_NAME --env staging

      - name: Run Smoke Tests
        run: |
          # Monitor for errors during smoke tests
          dd logs tail --service $SERVICE_NAME --env staging --follow &
          LOG_PID=$!

          ./smoke-tests.sh

          kill $LOG_PID

      - name: Check for Anomalies
        run: |
          ANOMALIES=$(dd anomalies detect --service $SERVICE_NAME --env staging --from 5m --json | jq '.anomalies | length')

          if [ $ANOMALIES -gt 0 ]; then
            echo "⚠️  Anomalies detected in staging"
            dd anomalies detect --service $SERVICE_NAME --env staging --from 5m
            exit 1
          fi

      - name: Deploy to Production
        run: ./deploy.sh production

      - name: Post-Deployment Validation
        run: |
          sleep 60  # Wait for rollout

          dd health check --service $SERVICE_NAME --env production
          dd apm services --service $SERVICE_NAME --env production --from 5m

      - name: Create Deployment Event
        run: |
          dd events create \
            --title "Deployed $SERVICE_NAME" \
            --text "SHA: ${{ github.sha }}" \
            --tags "service:$SERVICE_NAME,env:production,deployment:success"

      - name: Monitor Post-Deployment
        run: |
          # Monitor for 5 minutes after deployment
          for i in {1..5}; do
            echo "Monitoring minute $i/5..."

            ERROR_RATE=$(dd apm services --service $SERVICE_NAME --env production --from 1m --json | jq -r '.services[0].error_rate')

            if (( $(echo "$ERROR_RATE > 5" | bc -l) )); then
              echo "❌ High error rate detected: ${ERROR_RATE}%"
              echo "Rolling back deployment..."
              ./rollback.sh
              exit 1
            fi

            sleep 60
          done

          echo "✅ Deployment successful and stable"
```

---

## Multi-Service Correlation

### Scenario: Trace Request Flow Across Microservices

**Goal**: Follow a request through multiple services to find bottlenecks.

```bash
#!/bin/bash
# trace-request-flow.sh <trace-id>

TRACE_ID=$1

echo "=== Tracing Request Flow: $TRACE_ID ==="

# 1. Get full trace
trace=$(dd spans search --trace-id "$TRACE_ID" --json)

# 2. Extract services in the flow
echo -e "\n1. Services in Request Flow:"
echo "$trace" | jq -r '.spans[].service' | sort -u

# 3. Timeline of spans
echo -e "\n2. Request Timeline:"
echo "$trace" | jq -r '.spans[] | "\(.start_time) | \(.service).\(.operation) | \(.duration)ms"' | sort

# 4. Find slowest span
echo -e "\n3. Slowest Operation:"
echo "$trace" | jq -r '.spans | max_by(.duration) | "Service: \(.service)\nOperation: \(.operation)\nDuration: \(.duration)ms"'

# 5. Check for errors in trace
echo -e "\n4. Errors in Trace:"
echo "$trace" | jq -r '.spans[] | select(.error == true) | "❌ \(.service).\(.operation): \(.error_message)"'

# 6. Service dependency map
echo -e "\n5. Service Dependencies:"
dd service-map query --trace-id "$TRACE_ID"

# 7. Correlate with logs
echo -e "\n6. Related Logs:"
services=$(echo "$trace" | jq -r '.spans[].service' | sort -u | tr '\n' ' ')

for service in $services; do
  echo -e "\nLogs for $service:"
  dd logs search --query "trace_id:$TRACE_ID" --service "$service" --limit 5
done

# 8. Performance comparison
echo -e "\n7. Performance Comparison:"
endpoint=$(echo "$trace" | jq -r '.spans[0].resource')
echo "This request: $(echo "$trace" | jq -r '.duration')ms"

avg_duration=$(dd apm traces --resource "$endpoint" --from 1h --json | jq -r '.traces | map(.duration) | add / length')
echo "Average (1h): ${avg_duration}ms"
```

---

## Bonus: Complete Monitoring Setup

### Scenario: Set Up Complete Monitoring for New Service

**Goal**: Configure comprehensive monitoring for a new microservice.

```bash
#!/bin/bash
# setup-monitoring.sh <service-name>

SERVICE=$1

echo "=== Setting Up Monitoring for $SERVICE ==="

# 1. Create SLO
echo -e "\n1. Creating SLO..."
dd slos create \
  --name "$SERVICE Availability" \
  --type "metric" \
  --metric "trace.servlet.request.hits" \
  --target 99.9 \
  --service "$SERVICE" \
  --json > slo.json

SLO_ID=$(cat slo.json | jq -r '.id')
echo "SLO created: $SLO_ID"

# 2. Create monitors
echo -e "\n2. Creating Monitors..."

# High error rate monitor
dd monitors create \
  --name "$SERVICE - High Error Rate" \
  --type "metric alert" \
  --query "avg(last_5m):sum:trace.servlet.request.errors{service:$SERVICE}.as_count() > 10" \
  --message "@pagerduty-$SERVICE High error rate detected" \
  --tags "service:$SERVICE,severity:critical"

# High latency monitor
dd monitors create \
  --name "$SERVICE - High Latency" \
  --type "metric alert" \
  --query "avg(last_10m):avg:trace.servlet.request.duration{service:$SERVICE} > 1000" \
  --message "@slack-ops Latency spike in $SERVICE" \
  --tags "service:$SERVICE,severity:warning"

# 3. Create dashboard
echo -e "\n3. Creating Dashboard..."
dd dashboards create \
  --name "$SERVICE - Overview" \
  --template "apm_service" \
  --service "$SERVICE" \
  --json > dashboard.json

DASHBOARD_ID=$(cat dashboard.json | jq -r '.id')
echo "Dashboard created: $DASHBOARD_ID"

# 4. Train ML model
echo -e "\n4. Training ML Model (may take 1-2 minutes)..."
dd ml-insights train --service "$SERVICE" --from 30d

# 5. Set up auto-remediation
echo -e "\n5. Configuring Auto-Remediation..."
dd auto-remediate configure \
  --trigger "high_error_rate" \
  --action "restart_pods" \
  --service "$SERVICE" \
  --threshold 5

# 6. Create synthetic test
echo -e "\n6. Creating Synthetic Test..."
dd synthetics create \
  --name "$SERVICE Health Check" \
  --type "api" \
  --url "https://$SERVICE.example.com/health" \
  --interval 300 \
  --locations "aws:us-east-1,aws:eu-west-1"

# 7. Summary
cat << EOF

✅ Monitoring Setup Complete!

SLO ID: $SLO_ID
Dashboard: https://app.datadoghq.com/dashboard/$DASHBOARD_ID
Monitors: 2 created (error rate, latency)
ML Model: Trained on 30 days of data
Auto-Remediation: Configured for high error rate
Synthetic Test: Running every 5 minutes

Next steps:
1. Review dashboard: dd dashboards get --id $DASHBOARD_ID
2. Check SLO status: dd slos get --id $SLO_ID
3. Monitor health: dd health check --service $SERVICE
EOF
```

---

## Tips for Using Examples

1. **Customize thresholds** - Adjust error rates, latencies, and budgets for your environment
2. **Add notifications** - Integrate with Slack, PagerDuty, email
3. **Combine commands** - Chain multiple commands for complex workflows
4. **Use JSON output** - Parse with `jq` for automation
5. **Schedule regularly** - Use cron for continuous monitoring

---

## Foundation Commands

### Scenario: Security Monitoring and Service Catalog Integration

**Goal**: Monitor security signals and query service metadata for compliance.

#### Security Monitoring Signals

```bash
# Query recent security signals
dd security list --from 24h --json

# Search for specific security signals
dd security search --query "severity:high" --from 7d

# Get specific security signal details
dd security get --id "signal-123"

# Filter by service
dd security list --service api --from 24h
```

#### Watchdog Anomaly Detection

```bash
# Query Watchdog for automated anomaly detection
dd watchdog list --from 24h --json

# Get anomalies for specific service
dd watchdog search --service api --from 7d

# Get specific anomaly details
dd watchdog get --id "anomaly-456"

# Filter by severity
dd watchdog list --severity high --from 24h
```

#### Service Catalog Metadata

```bash
# Query Service Catalog for service metadata
dd catalog list --json

# Get metadata for specific service
dd catalog get --service api

# Search services by team
dd catalog search --team platform --json

# Get service ownership and SLOs
dd catalog get --service api --json | jq '{owner, team, slos}'
```

#### Combined Security and Compliance Workflow

```bash
#!/bin/bash
# security-compliance-check.sh

# Check for high-severity security signals
security_signals=$(dd security list --severity high --from 24h --json)
signal_count=$(echo "$security_signals" | jq '.signals | length')

if [ $signal_count -gt 0 ]; then
  echo "⚠️  Found $signal_count high-severity security signals"

  # Get affected services
  services=$(echo "$security_signals" | jq -r '.signals[].service' | sort -u)

  # For each affected service, get metadata
  for service in $services; do
    echo "\n📋 Service: $service"

    # Get service owner from catalog
    owner=$(dd catalog get --service "$service" --json | jq -r '.owner')
    echo "   Owner: $owner"

    # Check for Watchdog anomalies
    anomalies=$(dd watchdog search --service "$service" --from 24h --json)
    anomaly_count=$(echo "$anomalies" | jq '.anomalies | length')
    echo "   Watchdog Anomalies: $anomaly_count"

    # Notify owner if both security signals and anomalies present
    if [ $anomaly_count -gt 0 ]; then
      echo "   🚨 Alert: Security signals AND anomalies detected!"
      # Send notification to owner
    fi
  done
fi
```

**Use cases**:
- Security compliance monitoring
- Automated anomaly correlation
- Service ownership tracking
- Integrated security and observability

---

## More Resources

- **QUICK-REFERENCE.md** - One-page command reference
- **FAQ.md** - Common questions and answers
- **TROUBLESHOOTING.md** - Issue resolution guide
- **README.md** - Complete user guide

---

**Datadog CLI v0.1.0** - 54 Commands • Real-World Examples
**From Reactive to Predictive Operations**
