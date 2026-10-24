---
description: "Query Security Monitoring Signals for threats, vulnerabilities, and security events"
argument-hint: "[--from TIMERANGE] [--severity LEVEL] [--rule RULE]"
---

# Datadog Security Monitoring

Query Security Monitoring Signals to detect threats, investigate security events, and monitor for vulnerabilities.

## What is Security Monitoring?

Datadog Security Monitoring provides:
- **Threat detection** - Real-time security signal detection
- **Cloud SIEM** - Security information and event management
- **Compliance monitoring** - CIS benchmarks, PCI-DSS, HIPAA
- **Vulnerability tracking** - CVE detection and remediation

**Official Documentation**: https://www.datadoghq.com/product/cloud-siem/

## Usage

```bash
# Query all security signals
dd security

# Filter by time range
dd security --from 24h

# Filter by severity
dd security --severity high

# Filter by rule
dd security --rule "AWS access key exposed"
```

## Severity Levels

- `critical` - Immediate action required
- `high` - High priority investigation
- `medium` - Standard investigation
- `low` - Informational
- `info` - Context/audit trail

## Key Signal Types

**Threats**:
- Unauthorized access attempts
- Privilege escalation
- Data exfiltration
- Malware detection

**Vulnerabilities**:
- Exposed credentials
- Misconfigured security groups
- Unpatched systems
- Weak encryption

**Compliance**:
- CIS benchmark violations
- PCI-DSS failures
- HIPAA non-compliance
- GDPR issues

## Use Cases

### 1. Investigate Security Incidents
```bash
dd security --severity critical --from 1h
```

### 2. Monitor Compliance
```bash
dd security --rule "CIS" --from 7d
```

### 3. Track Vulnerabilities
```bash
dd security --from 24h
```

## Why Use the CLI?

- **Fast access** - 3ms startup for rapid incident response
- **Scriptable** - Automate security checks
- **Integration** - Combine with APM/logs for investigation
- **Offline** - Query cached data without network

## Example Prompts

> "Show me critical security signals from the last hour"
> "What security threats were detected today?"
> "Check for exposed AWS keys"
> "Monitor compliance violations"

## Learn More

- [Cloud SIEM](https://www.datadoghq.com/product/cloud-siem/)
- [Security Docs](https://docs.datadoghq.com/security/)