---
description: "Query serverless function monitoring for AWS Lambda, Azure Functions, and Google Cloud Functions"
argument-hint: "[--function FUNCTION] [--provider aws|azure|gcp] [--metric invocations|duration|errors|cold-starts]"
---

# Datadog Serverless Monitoring

Query serverless function monitoring for AWS Lambda, Azure Functions, and Google Cloud Functions to track invocations, duration, errors, and cold starts.

## What is Serverless Monitoring?

Serverless Monitoring provides complete visibility into:
- **Invocations** - Function execution counts and rates
- **Performance** - Execution duration and latency
- **Errors** - Failed invocations and error rates
- **Cold Starts** - Initialization time and frequency
- **Costs** - Estimated AWS/Azure/GCP costs

**Official Documentation**: https://docs.datadoghq.com/serverless/

## Usage

```bash
# Query all Lambda function invocations (last hour)
dd serverless

# Query specific function
dd serverless --function my-api-handler

# Query execution duration
dd serverless --metric duration

# Query error counts
dd serverless --metric errors

# Query cold start durations
dd serverless --metric cold-starts

# Query Azure Functions
dd serverless --provider azure --function my-function

# Query Google Cloud Functions
dd serverless --provider gcp --function my-function

# Query specific region
dd serverless --region us-east-1

# Query last 24 hours
dd serverless --duration 24h

# Get JSON output
dd serverless --function my-api --json
```

## Supported Providers

**AWS Lambda** (default)
- Enhanced metrics via Datadog Lambda Extension
- Cold start tracking
- Timeout and OOM monitoring
- Billing estimates

**Azure Functions**
- Execution counts and timing
- HTTP server errors
- Function app monitoring

**Google Cloud Functions**
- Execution counts and duration
- Error tracking
- Resource monitoring

## Metrics

**invocations** - Function invocation counts
**duration** - Execution time in milliseconds
**errors** - Error and failure counts
**cold-starts** - Cold start initialization time

## Use Cases

### 1. Monitor Invocation Rates
```bash
dd serverless --metric invocations --duration 24h
```

Track function invocation patterns over time.

### 2. Identify Slow Functions
```bash
dd serverless --metric duration --duration 7d
```

Find functions with high execution times.

### 3. Track Error Rates
```bash
dd serverless --metric errors
```

Quickly identify failing functions.

### 4. Analyze Cold Starts
```bash
dd serverless --metric cold-starts
```

Measure cold start impact on performance.

### 5. Compare Multi-Region Performance
```bash
dd serverless --region us-east-1 --metric duration
dd serverless --region eu-west-1 --metric duration
```

Compare function performance across regions.

### 6. Azure Functions Monitoring
```bash
dd serverless --provider azure --function my-func
```

Monitor Azure Functions execution.

### 7. Google Cloud Functions
```bash
dd serverless --provider gcp --function my-func
```

Track Google Cloud Function metrics.

## Why Use the CLI?

- **Fast queries** - Check function health in seconds
- **Multi-cloud** - AWS, Azure, and GCP support
- **Metric tracking** - Invocations, duration, errors, cold starts
- **Performance analysis** - Identify bottlenecks quickly
- **Cost monitoring** - Track function usage
- **Automation** - Script serverless health checks

## Example Prompts

> "Show me Lambda invocations for the last hour"
> "What's the error rate for my-api-handler?"
> "Check cold start times for my functions"
> "Compare function duration across regions"
> "Show Azure Functions execution counts"

## AWS Lambda Enhanced Metrics

Datadog provides enhanced Lambda metrics with second-granularity:
- `aws.lambda.enhanced.invocations`
- `aws.lambda.enhanced.duration`
- `aws.lambda.enhanced.errors`
- `aws.lambda.enhanced.init_duration` (cold starts)
- `aws.lambda.enhanced.timeout`
- `aws.lambda.enhanced.out_of_memory`
- `aws.lambda.enhanced.estimated_cost`

## Setup Requirements

**AWS Lambda:**
1. Install Datadog Lambda Extension
2. Instrument functions with Datadog library
3. Enable enhanced metrics

**Azure Functions:**
1. Configure Azure integration
2. Install Datadog extension

**Google Cloud Functions:**
1. Configure GCP integration
2. Enable Datadog tracing

## Troubleshooting

**No data appears:**
1. Verify serverless integration is configured
2. Check that functions have been invoked recently
3. Ensure enhanced metrics are enabled (AWS)
4. Verify function names match exactly

**Partial data:**
- Some functions may not have Datadog instrumentation
- Check Lambda Extension is deployed to all functions
- Verify region configuration

## Integration

Serverless Monitoring CLI integrates with:
- **APM** - Distributed tracing for serverless
- **Logs** - Function log aggregation
- **Metrics** - Custom serverless metrics
- **Cost** - FinOps and cost tracking

## Common Patterns

**Daily Health Check:**
```bash
dd serverless --duration 24h --metric errors
```

**Performance Baseline:**
```bash
dd serverless --metric duration --duration 7d
```

**Cold Start Analysis:**
```bash
dd serverless --metric cold-starts --duration 24h
```

**Regional Comparison:**
```bash
for region in us-east-1 eu-west-1 ap-southeast-1; do
  echo "=== $region ==="
  dd serverless --region $region --metric invocations
done
```

## Learn More

- [Serverless Monitoring Product Page](https://www.datadoghq.com/product/serverless-monitoring/)
- [AWS Lambda Monitoring](https://docs.datadoghq.com/serverless/aws_lambda/)
- [Azure Functions Monitoring](https://docs.datadoghq.com/serverless/azure_functions/)
- [Google Cloud Functions](https://docs.datadoghq.com/serverless/google_cloud_run/)

## Related Commands

- `dd apm` - Trace serverless function requests
- `dd logs` - Query Lambda/function logs
- `dd metrics` - Custom serverless metrics
- `dd cost` - Serverless cost analysis
