---
description: "Query Synthetic Monitoring tests - check uptime, API tests, browser tests, and test results"
argument-hint: "[TEST-NAME] [--type TYPE] [--from TIMERANGE] [--location LOCATION]"
---

# Datadog Synthetic Monitoring

Query Synthetic tests to proactively monitor uptime, API health, user journeys, and application availability from global locations.

## What is Synthetic Monitoring?

Synthetic Monitoring simulates user behavior to detect issues before users experience them:
- **API tests** - HTTP, SSL, TCP, DNS checks
- **Browser tests** - Simulate real user journeys
- **Uptime monitoring** - Availability from global locations
- **Multi-step tests** - Complex user workflows
- **Private locations** - Test internal applications

**Official Documentation**: https://docs.datadoghq.com/synthetics/

## Usage

```bash
# List all synthetic tests
dd synthetics list

# List filtered by test type
dd synthetics list --type api
dd synthetics list --type browser

# List filtered by location
dd synthetics list --location us-east

# Get specific test details
dd synthetics get --id abc123def

# Get test results
dd synthetics results --id abc123def --from 24h

# Create a new synthetic test
dd synthetics create --name "API Health Check" --type api --url "https://api.example.com/health"

# Pause a test
dd synthetics pause --id abc123def

# Resume a test
dd synthetics resume --id abc123def

# Delete a test
dd synthetics delete --id abc123def
```

## Test Types

**API Tests**:
- HTTP requests (GET, POST, PUT, DELETE)
- SSL certificate validation
- TCP connectivity checks
- DNS resolution tests
- WebSocket connections
- gRPC calls

**Browser Tests**:
- User login flows
- Checkout processes
- Navigation workflows
- Form submissions
- File uploads
- Multi-page journeys

**Multistep API Tests**:
- Authentication flows
- API workflow chains
- Sequential operations
- Variable passing
- Data-driven tests

**Private Location Tests**:
- Internal applications
- VPC-hosted services
- On-premise systems
- Staging environments

## Key Metrics

**Availability**:
- Uptime percentage
- Test success rate
- Alert frequency
- Downtime duration

**Performance**:
- Response time (p50, p95, p99)
- DNS lookup time
- SSL handshake time
- Time to first byte
- Full page load time

**Geographic**:
- Performance by location
- Regional outages
- CDN effectiveness
- Latency distribution

## Use Cases

### 1. Check Service Uptime
```bash
dd synthetics api-health-check --from 7d
```

Monitor API availability and response times across global locations.

### 2. Test User Journeys
```bash
dd synthetics checkout-flow --type browser
```

Ensure critical user workflows function correctly.

### 3. Pre-Deployment Validation
```bash
dd synthetics --status failing
```

Verify no synthetic tests are failing before deploying.

### 4. Monitor SSL Certificates
```bash
dd synthetics ssl-check --type ssl
```

Get alerts before certificates expire.

### 5. Regional Performance
```bash
dd synthetics api-test --location ap-southeast
```

Check performance from specific geographic regions.

## Why Use the CLI?

- **Pre-deploy checks** - Verify synthetics pass before releasing
- **Fast debugging** - Check test results in 3ms
- **CI/CD integration** - Block deployments if tests fail
- **Automation** - Script synthetic test creation
- **Context awareness** - Auto-detect service from git repo
- **Offline access** - View cached test configurations

## Test Locations

**AWS Regions**:
- us-east-1, us-west-1, eu-west-1
- ap-southeast-1, ap-northeast-1
- sa-east-1, ca-central-1

**Azure Regions**:
- eastus, westus, northeurope
- australiaeast, japaneast

**Private Locations**:
- Custom on-premise locations
- VPC-hosted test runners
- Internal network testing

## Example Prompts

> "Show me all failing synthetic tests"
> "Check the uptime of api-health-check"
> "What's the response time for the login flow?"
> "List all browser tests"
> "Show synthetic test results from us-east"

## Learn More

- [Synthetic Monitoring](https://docs.datadoghq.com/synthetics/)
- [API Tests](https://docs.datadoghq.com/synthetics/api_tests/)
- [Browser Tests](https://docs.datadoghq.com/synthetics/browser_tests/)
- [Private Locations](https://docs.datadoghq.com/synthetics/private_locations/)
