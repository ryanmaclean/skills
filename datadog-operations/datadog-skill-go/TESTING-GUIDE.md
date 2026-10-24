# Datadog Claude Code Plugin - Testing Guide

**Purpose:** Step-by-step guide for testing all 22 skills with real Datadog credentials.

## Prerequisites

### 1. Datadog Account Setup

You need a Datadog account with:
- API Key (DD_API_KEY)
- Application Key (DD_APP_KEY)
- Datadog Site (DD_SITE, e.g., datadoghq.com, datadoghq.eu, us5.datadoghq.com)

**Get Your Keys:**
1. Log in to Datadog: https://app.datadoghq.com
2. Navigate to Organization Settings → API Keys
3. Create or copy an API Key
4. Navigate to Organization Settings → Application Keys
5. Create or copy an Application Key

### 2. Environment Setup

Export your credentials:

```bash
export DD_API_KEY="your-api-key-here"
export DD_APP_KEY="your-app-key-here"
export DD_SITE="datadoghq.com"  # or your specific site
```

Or add to your shell profile (`~/.zshrc` or `~/.bashrc`):

```bash
echo 'export DD_API_KEY="your-api-key-here"' >> ~/.zshrc
echo 'export DD_APP_KEY="your-app-key-here"' >> ~/.zshrc
echo 'export DD_SITE="datadoghq.com"' >> ~/.zshrc
source ~/.zshrc
```

### 3. Verify CLI Installation

```bash
# Check CLI is installed
which dd
dd version

# Should show version 1.0.0 or later
```

### 4. Test API Connectivity

```bash
# Basic connectivity test
dd health --help

# If this fails, check:
# - Credentials are exported
# - CLI binary has execute permissions
# - Network connectivity to Datadog
```

## Testing Plan

### Phase 1: Core Operations (High Priority)

**Estimated Time:** 30 minutes

#### 1.1 Health Check (skill: health.md)

```bash
# Test auto-detection
dd health

# Test specific service (replace with your service)
dd health api-service

# Test with time range
dd health api-service --from 6h

# Expected: Health status, APM metrics, error logs, recommendations
```

**Validation:**
- [ ] Command executes without errors
- [ ] Returns JSON output
- [ ] Shows service health metrics
- [ ] Includes APM data (request rate, latency, errors)
- [ ] Includes recent error logs
- [ ] Provides health recommendations

#### 1.2 Deploy Safety (skill: deploy.md)

```bash
# Test auto-detection
dd deploy

# Test specific service
dd deploy api-service

# Test specific environment
dd deploy api-service --environment production

# Expected: Safety decision (SAFE/UNSAFE/WARNING), blocking issues, recommendations
```

**Validation:**
- [ ] Command executes without errors
- [ ] Returns safety decision
- [ ] Checks for active incidents
- [ ] Checks for error spikes
- [ ] Provides deployment recommendations
- [ ] Exit code matches safety status (0=safe, 1=unsafe, 2=warning)

### Phase 2: Observability & Monitoring (High Priority)

**Estimated Time:** 1 hour

#### 2.1 APM Traces (skill: apm.md)

```bash
# Query service traces
dd apm api-service

# Query error traces
dd apm api-service --status error --from 2h

# Query specific resource
dd apm api-service --resource "GET /api/users"

# Expected: Trace count, request rate, error rate, latency percentiles
```

**Validation:**
- [ ] Returns trace statistics
- [ ] Shows request rate (req/s)
- [ ] Shows error rate (%)
- [ ] Shows latency percentiles (p50, p95, p99)
- [ ] Filters by status work correctly
- [ ] Time range filtering works

#### 2.2 Logs (skill: logs.md)

```bash
# Search all logs
dd logs

# Search with query
dd logs "error database timeout"

# Filter by service and level
dd logs --service api-service --status error --from 24h

# Limit results
dd logs --service api-service --limit 50

# Expected: Log entries with timestamps, service, level, message
```

**Validation:**
- [ ] Returns log entries
- [ ] Query syntax works (AND, OR, wildcards)
- [ ] Service filtering works
- [ ] Status/level filtering works
- [ ] Time range filtering works
- [ ] Limit parameter works
- [ ] Returns JSON-formatted logs

#### 2.3 Metrics (skill: metrics.md)

```bash
# Query system metrics
dd metrics "avg:system.cpu.user{*}"

# Query APM metrics
dd metrics "avg:trace.servlet.request.duration{service:api}"

# Query with time range
dd metrics "sum:http.requests{env:prod}" --from 2h

# Expected: Timeseries data with timestamps and values
```

**Validation:**
- [ ] Returns metric data
- [ ] Supports aggregations (avg, sum, min, max)
- [ ] Tag filtering works
- [ ] Time range works
- [ ] Returns timeseries data points

#### 2.4 RUM (skill: rum.md)

```bash
# Query RUM data
dd rum

# Query specific application
dd rum my-web-app

# Filter by view
dd rum my-app --view "/checkout" --from 24h

# Expected: RUM metrics (Core Web Vitals, page load, errors)
```

**Validation:**
- [ ] Returns RUM data
- [ ] Shows frontend performance metrics
- [ ] Filters by application
- [ ] Filters by view/page
- [ ] Shows Core Web Vitals (LCP, FID, CLS)

#### 2.5 Network (skill: network.md)

```bash
# Query network traffic
dd network

# Filter by source
dd network --source api-service

# Filter by destination
dd network --dest database-service --from 1h

# Expected: Network traffic data, connections, latency
```

**Validation:**
- [ ] Returns network data
- [ ] Shows traffic volume
- [ ] Shows connections
- [ ] Shows network latency
- [ ] Filters by source/destination work

#### 2.6 Database (skill: database.md)

```bash
# Query database metrics
dd database

# Query specific database
dd database postgres-db

# Find slow queries
dd database --slow-queries --from 24h

# Expected: Database metrics, query performance, slow queries
```

**Validation:**
- [ ] Returns database metrics
- [ ] Shows query performance
- [ ] Identifies slow queries
- [ ] Shows connection metrics
- [ ] Provides optimization insights

#### 2.7 LLM Observability (skill: llm.md)

```bash
# Query LLM data
dd llm

# Query specific application
dd llm my-ai-app

# Filter by model
dd llm my-app --model gpt-4 --from 7d

# Expected: LLM metrics (costs, tokens, latency, quality)
```

**Validation:**
- [ ] Returns LLM observability data
- [ ] Shows cost metrics
- [ ] Shows token usage
- [ ] Shows latency metrics
- [ ] Filters by application and model

### Phase 3: Incident & Alert Management (Medium Priority)

**Estimated Time:** 30 minutes

#### 3.1 Incidents (skill: incidents.md)

```bash
# List incidents
dd incidents list

# Filter by status
dd incidents list --status active

# Create incident (TEST CAREFULLY)
dd incidents create --title "Test Incident" --severity SEV-5

# Update incident
dd incidents update <incident-id> --status stable

# Close incident
dd incidents close <incident-id>

# Expected: Incident data (ID, title, severity, status, service)
```

**Validation:**
- [ ] Lists incidents correctly
- [ ] Filters by status work
- [ ] Can create test incident
- [ ] Can update incident status
- [ ] Can close incident
- [ ] Returns incident IDs

**IMPORTANT:** Only create test incidents with SEV-5 (informational) severity!

#### 3.2 Monitors (skill: monitors.md)

```bash
# List monitors
dd monitors

# Filter by status
dd monitors --status alert

# Mute monitor (TEST CAREFULLY)
dd monitors <monitor-name> --mute --duration 10m --reason "Testing CLI"

# Unmute monitor
dd monitors <monitor-name> --unmute

# Expected: Monitor data (status, thresholds, alerts)
```

**Validation:**
- [ ] Lists monitors correctly
- [ ] Filters by status work
- [ ] Can mute monitors
- [ ] Can unmute monitors
- [ ] Shows current monitor status

**IMPORTANT:** Only mute test monitors, not production monitors!

### Phase 4: Advanced Features (Medium Priority)

**Estimated Time:** 1.5 hours

#### 4.1 SLOs (skill: slos.md)

```bash
# List SLOs
dd slos

# Query specific SLO
dd slos api-availability --from 30d

# Check error budget
dd slos api-latency

# Expected: SLO status, compliance %, error budget
```

**Validation:**
- [ ] Lists SLOs
- [ ] Shows SLO compliance percentage
- [ ] Shows error budget status
- [ ] Shows burn rate
- [ ] Time range filtering works

#### 4.2 Synthetics (skill: synthetics.md)

```bash
# List synthetic tests
dd synthetics

# Filter by type
dd synthetics --type api
dd synthetics --type browser

# Check test results
dd synthetics login-flow --from 24h

# Expected: Test results, uptime, response times
```

**Validation:**
- [ ] Lists synthetic tests
- [ ] Filters by type (api, browser)
- [ ] Shows test results
- [ ] Shows uptime metrics
- [ ] Shows response times by location

#### 4.3 CI/CD Visibility (skill: cicd.md)

```bash
# Query pipelines
dd cicd

# Query specific pipeline
dd cicd my-app-pipeline --from 7d

# Filter by status
dd cicd --status failed --from 24h

# Expected: Pipeline data, build times, test results
```

**Validation:**
- [ ] Lists pipelines
- [ ] Shows build duration
- [ ] Shows failure rates
- [ ] Shows test analytics
- [ ] Filters by branch and status work

#### 4.4 Workflows (skill: workflows.md)

```bash
# List workflows
dd workflows

# Trigger workflow (TEST CAREFULLY)
dd workflows test-workflow --trigger

# Check workflow status
dd workflows test-workflow --status

# View history
dd workflows --history --from 24h

# Expected: Workflow data, execution status
```

**Validation:**
- [ ] Lists workflows
- [ ] Can trigger workflows (use test workflows only!)
- [ ] Shows execution status
- [ ] Shows execution history

**IMPORTANT:** Only trigger test workflows, not production automations!

#### 4.5 Security Monitoring (skill: security.md)

```bash
# Query security signals
dd security

# Filter by severity
dd security --severity critical --from 24h

# Filter by type
dd security --type "SQL Injection"

# Expected: Security signals, threats, vulnerabilities
```

**Validation:**
- [ ] Returns security signals
- [ ] Shows threat detection data
- [ ] Filters by severity work
- [ ] Shows vulnerability information

#### 4.6 Watchdog (skill: watchdog.md)

```bash
# Query Watchdog alerts
dd watchdog

# Filter by service
dd watchdog --service api-service

# Time range
dd watchdog --from 24h

# Expected: Anomalies detected by AI
```

**Validation:**
- [ ] Returns Watchdog alerts
- [ ] Shows anomaly detection results
- [ ] Filters by service work
- [ ] Shows AI-suggested correlations

#### 4.7 Cost Management (skill: cost.md)

```bash
# Query cloud costs
dd cost

# Filter by cloud provider
dd cost --cloud aws --from 30d

# Group by tags
dd cost --group-by team

# Expected: Cost data by service, team, environment
```

**Validation:**
- [ ] Returns cost data
- [ ] Filters by cloud provider work
- [ ] Group-by tag filtering works
- [ ] Shows cost trends

### Phase 5: Infrastructure & Discovery (Low Priority)

**Estimated Time:** 30 minutes

#### 5.1 Service Catalog (skill: catalog.md)

```bash
# List services
dd catalog

# Query specific service
dd catalog api-service

# Filter by team
dd catalog --team platform

# Expected: Service metadata, ownership, dependencies
```

**Validation:**
- [ ] Lists services from catalog
- [ ] Shows service metadata
- [ ] Shows ownership information
- [ ] Shows dependencies
- [ ] Filters by team work

#### 5.2 Dashboards (skill: dashboards.md)

```bash
# List dashboards
dd dashboards

# Query specific dashboard
dd dashboards production-overview

# Export dashboard
dd dashboards my-dashboard --export

# Expected: Dashboard data, configurations
```

**Validation:**
- [ ] Lists dashboards
- [ ] Can query specific dashboards
- [ ] Can export dashboard configs
- [ ] Returns dashboard URLs

#### 5.3 Context Detection (skill: context.md)

```bash
# Show current context
dd context

# Set context manually
dd context --set api-service

# Clear context
dd context --clear

# Expected: Detected service from git/environment
```

**Validation:**
- [ ] Detects service from git repository
- [ ] Reads DD_SERVICE environment variable
- [ ] Can manually set context
- [ ] Can clear context
- [ ] Shows current context

#### 5.4 Generic CLI Access (skill: datadog.md)

```bash
# Test generic command access
dd help

# Use any command
dd health api-service
dd apm checkout-service --status error

# Expected: All commands accessible
```

**Validation:**
- [ ] All commands work through generic skill
- [ ] Help command works
- [ ] Flags and arguments pass through correctly

## Test Results Template

Create a file `TEST-RESULTS.md` to track your testing:

```markdown
# Test Results

Date: [DATE]
Tester: [YOUR NAME]
Datadog Site: [DD_SITE]

## Phase 1: Core Operations
- [ ] health - PASS/FAIL - [notes]
- [ ] deploy - PASS/FAIL - [notes]

## Phase 2: Observability & Monitoring
- [ ] apm - PASS/FAIL - [notes]
- [ ] logs - PASS/FAIL - [notes]
- [ ] metrics - PASS/FAIL - [notes]
- [ ] rum - PASS/FAIL - [notes]
- [ ] network - PASS/FAIL - [notes]
- [ ] database - PASS/FAIL - [notes]
- [ ] llm - PASS/FAIL - [notes]

## Phase 3: Incident & Alert Management
- [ ] incidents - PASS/FAIL - [notes]
- [ ] monitors - PASS/FAIL - [notes]

## Phase 4: Advanced Features
- [ ] slos - PASS/FAIL - [notes]
- [ ] synthetics - PASS/FAIL - [notes]
- [ ] cicd - PASS/FAIL - [notes]
- [ ] workflows - PASS/FAIL - [notes]
- [ ] security - PASS/FAIL - [notes]
- [ ] watchdog - PASS/FAIL - [notes]
- [ ] cost - PASS/FAIL - [notes]

## Phase 5: Infrastructure & Discovery
- [ ] catalog - PASS/FAIL - [notes]
- [ ] dashboards - PASS/FAIL - [notes]
- [ ] context - PASS/FAIL - [notes]
- [ ] datadog - PASS/FAIL - [notes]

## Bugs Found
1. [Bug description]
2. [Bug description]

## Improvements Needed
1. [Improvement suggestion]
2. [Improvement suggestion]
```

## Common Issues & Solutions

### Issue: "Authentication failed"
**Solution:** Check DD_API_KEY and DD_APP_KEY are exported and valid.

### Issue: "Service not found"
**Solution:** Service names are case-sensitive. Check exact service name in Datadog.

### Issue: "Command not found: dd"
**Solution:** Ensure CLI binary is in PATH or use full path to binary.

### Issue: "No data returned"
**Solution:** Check time range. Service may not have data in specified time period.

### Issue: "Rate limit exceeded"
**Solution:** Datadog API has rate limits. Wait a few minutes between tests.

### Issue: "Permission denied"
**Solution:** Application key may not have sufficient permissions. Check API key permissions in Datadog.

## Safety Guidelines

**DO:**
- Test with non-production services first
- Use SEV-5 (informational) when creating test incidents
- Only mute test monitors
- Only trigger test workflows
- Use short mute durations (10-15 minutes)

**DON'T:**
- Create high-severity incidents in production
- Mute production monitors without reason
- Trigger production workflows without understanding impact
- Delete or modify existing incidents/monitors/SLOs
- Run destructive operations

## Next Steps After Testing

1. Document all bugs found in GitHub issues
2. Update skill documentation with any corrections
3. Add real-world examples from testing to skills
4. Create troubleshooting guide based on issues encountered
5. Share test results with team

---

**Testing Checklist:**
- [ ] Phase 1: Core Operations (2 skills)
- [ ] Phase 2: Observability & Monitoring (7 skills)
- [ ] Phase 3: Incident & Alert Management (2 skills)
- [ ] Phase 4: Advanced Features (7 skills)
- [ ] Phase 5: Infrastructure & Discovery (4 skills)
- [ ] All 22 skills tested
- [ ] Test results documented
- [ ] Bugs reported
