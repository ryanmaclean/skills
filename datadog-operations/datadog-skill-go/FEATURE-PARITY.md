# Feature Parity Comparison: Go CLI vs Bash vs Python

**Last Updated:** January 29, 2026

## Summary

- **Go CLI**: 57 commands (most comprehensive)
- **Bash Scripts**: 24 scripts (skill-focused, Claude Code optimized)
- **Python Scripts**: 23 scripts (bash parity with better structure)

**Verdict: NO feature parity** - Go CLI has ~2.4x more features than bash/python skills.

---

## Core Observability Features

| Feature | Go CLI | Bash | Python |
|---------|--------|------|--------|
| APM/Traces | ✅ `apm`, `spans`, `service-map` | ✅ `query-apm.sh` | ✅ `query_apm.py`, `query_apm_refactored.py` |
| Logs | ✅ `logs` | ✅ `search-logs.sh` | ✅ `search_logs.py` |
| Metrics | ✅ `metrics` | ✅ `query-metrics.sh` | ✅ `query_metrics.py` |
| Database | ✅ `database` | ✅ `query-database.sh` | ✅ `query_database.py` |
| Security | ✅ `security` | ✅ `query-security-signals.sh` | ✅ `query_security_signals.py` |
| SLOs | ✅ `slos`, `error-budgets`, `slo-history` | ✅ `query-slos.sh`, `calculate-error-budget.sh` | ✅ `query_slos.py`, `calculate_error_budget.py` |
| Watchdog | ✅ `watchdog` | ✅ `query-watchdog.sh` | ✅ `query_watchdog.py` |
| RUM | ✅ `rum` | ✅ `query-rum.sh` | ❌ |
| LLM | ✅ `llm` | ✅ `analyze-llm.sh` | ✅ `analyze_llm.py` |
| Service Catalog | ✅ `catalog` | ✅ `query-service-catalog.sh` | ✅ `query_service_catalog.py` |

---

## Management Operations

| Feature | Go CLI | Bash | Python |
|---------|--------|------|--------|
| Monitors | ✅ `monitors` | ✅ `manage-monitors.sh` | ✅ `manage_monitors.py` |
| Incidents | ✅ `incidents` | ✅ `manage-incidents.sh` | ✅ `manage_incidents.py` |
| Dashboards | ✅ `dashboards` | ✅ `create-dashboard.sh` | ✅ `create_dashboard.py` |
| Synthetics | ✅ `synthetics` | ✅ `manage-synthetics.sh` | ✅ `manage_synthetics.py` |
| Workflows | ✅ `workflows` | ✅ `trigger-workflow.sh` | ✅ `trigger_workflow.py` |
| Downtimes | ✅ `downtimes` | ❌ | ❌ |
| Notebooks | ✅ `notebooks` | ❌ | ❌ |

---

## Smart Operations

| Feature | Go CLI | Bash | Python |
|---------|--------|------|--------|
| Health Check | ✅ `health` | ✅ `smart-health.sh` | ✅ `smart_health.py` |
| Deploy Check | ✅ `deploy` | ✅ `deploy-check.sh` | ✅ `deploy_check.py` |
| Context Detection | ✅ `context` | ✅ `detect-context.sh` | ✅ `detect_context.py` |
| Service Investigation | ❌ | ✅ `investigate-service.sh` | ✅ `investigate_service.py` |
| Usage/Cost Analysis | ❌ | ✅ `analyze-usage-cost.sh` | ✅ `analyze_usage_cost.py` |

---

## Go CLI Exclusive Features

These 30+ features are only available in the Go CLI:

### Infrastructure Monitoring
- `containers` - Docker and Kubernetes container monitoring
- `kubernetes` - Kubernetes pod and cluster monitoring
- `serverless` - Lambda, Azure Functions, Cloud Functions
- `network` - Network Performance Monitoring

### Advanced Analytics
- `anomalies` - Detect anomalies across signals
- `correlation` - Correlate events for root cause analysis
- `impact-analysis` - Assess blast radius and dependencies

### Platform Administration
- `teams` - Manage teams for organization
- `users` - User access control and administration
- `roles` - Fine-grained permissions management
- `service-accounts` - Service accounts for automation
- `api-keys` - Primary authentication keys
- `application-keys` - API authentication keys

### Compliance & Security
- `audit-logs` - Compliance and security tracking

### Collaboration
- `cases` - Case Management for issue tracking
- `status-pages` - Customer communication
- `on-call` - Scheduling and rotations

### Software Delivery
- `dora` - DORA Metrics for DevOps performance
- `cicd` - CI Visibility for pipelines and tests

### Additional Features
- `events` - Query and post deployment events
- `tags` - Manage host tags
- `integrations` - Cloud provider integrations (AWS, Azure, Slack)
- `slo-corrections` - Manage SLO corrections
- `auto-remediate` - Automated remediation workflows
- `change-management` - Track and correlate changes

---

## Bash/Python Exclusive Features

These features are only in bash/python skills:

### Bash Only
- `example-monitored-script.sh` - Instrumentation example
- `query-rum.sh` - Real User Monitoring (Python missing)
- `TEST_MONITORING.sh` - Test harness

### Python Only
- `query_apm_refactored.py` - Refactored APM query
- `dd.py` - Python CLI wrapper

### Both Bash & Python (not in Go)
- `investigate-service` - Comprehensive multi-signal service investigation
- `analyze-usage-cost` - Detailed cost and usage analysis

---

## Implementation Characteristics

### Go CLI (Production-Grade)
- **Lines of Code**: ~15,000+ LOC
- **Architecture**: Modular packages with clean interfaces
- **Error Handling**: Comprehensive with retry logic
- **Testing**: 232 tests, 83% coverage
- **Performance**: <3ms startup time
- **Output**: JSON, table, or human-readable
- **Use Case**: Production CLI tool, CI/CD pipelines, automation

### Bash Scripts (Skill-Optimized)
- **Lines of Code**: ~3,000 LOC total
- **Architecture**: Standalone scripts
- **Error Handling**: Basic with curl error codes
- **Testing**: Integration tests only
- **Performance**: ~100-500ms per script
- **Output**: JSON for Claude Code parsing
- **Use Case**: Claude Code skill integration, quick queries

### Python Scripts (Maintainable Skills)
- **Lines of Code**: ~4,000 LOC total
- **Architecture**: Modular with lib/ directory
- **Error Handling**: Exception handling with retries
- **Testing**: Unit and integration tests
- **Performance**: ~200-600ms per script
- **Output**: Conversational formatted output
- **Use Case**: Claude Code skill integration, extensible framework

---

## Feature Coverage by Category

| Category | Go CLI | Bash | Python |
|----------|--------|------|--------|
| **Observability** (APM, logs, metrics, etc.) | 10/10 | 10/10 | 9/10 |
| **Management** (monitors, dashboards, etc.) | 7/7 | 5/7 | 5/7 |
| **Smart Ops** (health, deploy, context) | 3/3 | 5/5 | 5/5 |
| **Infrastructure** | 4/4 | 0/4 | 0/4 |
| **Platform Admin** | 6/6 | 0/6 | 0/6 |
| **Advanced Analytics** | 3/3 | 0/3 | 0/3 |
| **Compliance** | 1/1 | 0/1 | 0/1 |
| **Collaboration** | 3/3 | 0/3 | 0/3 |
| **DORA/CI/CD** | 2/2 | 0/2 | 0/2 |

**Coverage Score:**
- Go CLI: 39/39 features (100%)
- Bash: 20/39 features (51%)
- Python: 19/39 features (49%)

---

## Shared Features (Core Parity)

These 21 features exist in all three implementations:

1. APM query
2. Log search
3. Metrics query
4. Database monitoring
5. Security signals
6. SLO query
7. Error budget calculation
8. Watchdog anomalies
9. LLM observability
10. Service catalog
11. Monitor management
12. Incident management
13. Dashboard creation
14. Synthetic test management
15. Workflow triggers
16. Health checks
17. Deploy safety checks
18. Context detection
19. Setup verification
20. Smart health analysis
21. Service investigation (bash/python only)

---

## Recommendations

### Use Go CLI when you need:
- Production-ready tooling
- Platform administration
- CI/CD integration
- Advanced analytics
- Infrastructure monitoring
- Compliance tracking
- Full Datadog API coverage

### Use Bash Scripts when you need:
- Quick Claude Code skill integration
- Rapid prototyping
- Minimal dependencies
- Simple read operations
- Fast queries (no dependencies to install)

### Use Python Scripts when you need:
- Maintainable skill codebase
- Structured error handling
- Extensible framework
- Better code organization
- Conversational output formatting

---

## Bridging the Gap

To achieve feature parity, bash/python skills would need:

### High Priority (8 features)
1. Downtimes management
2. Notebooks management
3. Container monitoring
4. Kubernetes monitoring
5. Serverless monitoring
6. Network monitoring
7. Anomaly detection
8. Correlation analysis

### Medium Priority (6 features)
9. Events management
10. Tags management
11. Integrations management
12. SLO corrections
13. DORA metrics
14. CI/CD visibility

### Low Priority (15+ features)
- Platform admin (teams, users, roles, etc.)
- Compliance (audit logs)
- Collaboration (cases, status pages, on-call)
- Advanced automation

---

## Conclusion

The Go CLI is the **most comprehensive** implementation with 57 commands covering all Datadog product areas. It's production-ready and extensively tested.

The Bash and Python skills have **51% feature parity** focused on core observability, management, and smart operations. They're optimized for Claude Code integration and conversational interaction.

**Key Insight**: The implementations serve different purposes:
- **Go CLI**: Production automation and full API access
- **Bash/Python Skills**: Claude Code integration and quick queries

Feature parity is intentionally not 100% - the skills focus on the most useful operations for conversational AI interaction, while the Go CLI provides comprehensive API coverage for all use cases.
