---
description: "Query Database Monitoring for database performance analysis, slow queries, and connection metrics"
argument-hint: "[DATABASE] [--from TIMERANGE] [--query-type TYPE]"
---

# Datadog Database Monitoring

Query Database Monitoring to analyze database performance, identify slow queries, track connections, and optimize database operations.

## What is Database Monitoring?

Datadog Database Monitoring provides deep visibility into database performance:
- **Query-level metrics** - Execution time, wait events, explain plans
- **Real-time monitoring** - Active queries, connections, locks
- **Historical analysis** - Query trends, performance regression
- **Optimization insights** - Recommendations for indexes, schema changes

**Official Documentation**: https://www.datadoghq.com/product/database-monitoring/

## Usage

```bash
# Query all database activity
dd database

# Query specific database
dd database postgres-prod

# Filter by time range
dd database mysql-db --from 6h

# Filter by query type
dd database --query-type SELECT
```

## Supported Databases

- **PostgreSQL** - All versions
- **MySQL** - MySQL 5.6+, MariaDB
- **SQL Server** - SQL Server 2012+
- **Oracle** - Oracle 11g+
- **MongoDB** - MongoDB 4.0+

## Key Metrics

**Query Performance**:
- Execution time (avg, p95, p99)
- Rows examined vs returned
- Query frequency
- Lock wait time

**Connection Metrics**:
- Active connections
- Connection pool usage
- Connection errors
- Max connections reached

**Resource Usage**:
- CPU utilization
- Memory usage
- Disk I/O
- Network traffic

## Use Cases

### 1. Identify Slow Queries
```bash
dd database --from 1h
```

Shows queries with highest execution time, most frequent, or most resource-intensive.

### 2. Monitor Connection Health
```bash
dd database postgres-prod --from 15m
```

Tracks connection pool status, errors, and capacity issues.

### 3. Optimize Query Performance
```bash
dd database --query-type SELECT --from 24h
```

Analyzes SELECT query patterns for optimization opportunities.

## Why Use the CLI?

- **Faster than UI** - 3ms startup vs dashboard load time
- **Scriptable** - Automate performance checks in CI/CD
- **Git context** - Auto-detects database from app config
- **Multi-signal** - Correlate with APM and logs

## Example Prompts

> "Show me slow database queries from the last hour"
> "What's the database connection pool status?"
> "Find queries taking longer than 1 second"
> "Monitor PostgreSQL performance"

## Learn More

- [Database Monitoring Docs](https://www.datadoghq.com/product/database-monitoring/)
- [Query Performance Best Practices](https://docs.datadoghq.com/database_monitoring/)