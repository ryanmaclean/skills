# Command Category Alignment with Datadog Website

**Date:** January 22, 2026 (Iteration 26)
**Purpose:** Align CLI command structure with Datadog's official product categories and identify expansion opportunities

---

## Research Sources

- [Datadog Product Page](https://www.datadoghq.com/product/)
- [Datadog API Reference](https://docs.datadoghq.com/api/latest/)
- [DASH 2025 Announcements](https://www.datadoghq.com/blog/dash-2025-new-feature-roundup-keynote/)
- [Datadog CI Tool (Apache 2.0)](https://github.com/DataDog/datadog-ci)
- [Datadog Sync CLI (BSD-3-Clause)](https://github.com/DataDog/datadog-sync-cli)

---

## Datadog's Official Product Categories (2026)

### 1. Observability
**Infrastructure:**
- Infrastructure Monitoring ✅ (our: network, cost)
- Metrics ✅ (our: metrics)
- Network Monitoring ✅ (our: network)
- Container Monitoring ❌ (MISSING)
- Kubernetes Autoscaling ❌ (MISSING)
- Serverless ❌ (MISSING)
- Cloud Cost Management ✅ (our: cost)
- Cloudcraft ❌ (MISSING - visualization tool)
- Storage Management ❌ (MISSING)

**Applications:**
- APM ✅ (our: apm)
- Universal Service Monitoring ❌ (MISSING)
- Continuous Profiler ❌ (MISSING)
- Dynamic Instrumentation ❌ (MISSING)
- LLM Observability ✅ (our: llm)

**Data:**
- Database Monitoring ✅ (our: database)
- Data Streams Monitoring ❌ (MISSING)
- Quality Monitoring ❌ (MISSING)
- Jobs Monitoring ❌ (MISSING)

**Logs:**
- Log Management ✅ (our: logs)
- Sensitive Data Scanner ❌ (MISSING)
- Audit Trail ❌ (MISSING)
- Observability Pipelines ❌ (MISSING)
- Error Tracking ❌ (MISSING)
- CloudPrem ❌ (MISSING)

### 2. Security
**Code Security:**
- Code Security ❌ (MISSING)
- Software Composition Analysis ❌ (MISSING)
- SAST ❌ (MISSING)
- IAST ❌ (MISSING)
- IaC Security ❌ (MISSING)
- Secret Scanning ❌ (MISSING)

**Cloud Security:**
- Cloud Security ✅ (our: security - partial)
- CSPM ❌ (MISSING)
- CIEM ❌ (MISSING)
- Vulnerability Management ❌ (MISSING)
- Compliance ❌ (MISSING)

**Threat Management:**
- Cloud SIEM ❌ (MISSING)
- Workload Protection ❌ (MISSING)
- App and API Protection ❌ (MISSING)
- Sensitive Data Scanner ❌ (MISSING)

### 3. Digital Experience
- Browser Real User Monitoring ✅ (our: rum)
- Mobile RUM ✅ (our: rum - partial)
- Product Analytics ❌ (MISSING - NEW in 2025!)
- Session Replay ❌ (MISSING)
- Synthetic Monitoring ✅ (our: synthetics)
- Mobile App Testing ❌ (MISSING)
- Error Tracking ❌ (MISSING)

### 4. Software Delivery
- Internal Developer Portal ❌ (MISSING - NEW in 2025!)
- CI Visibility ✅ (our: cicd)
- Test Optimization ❌ (MISSING)
- Continuous Testing ✅ (our: synthetics - partial)
- IDE Plugins ❌ (MISSING - integration)
- Feature Flags ❌ (MISSING)
- Code Coverage ❌ (MISSING)

### 5. Service Management
- Incident Response ✅ (our: incidents)
- Software Catalog ✅ (our: catalog)
- SLOs ✅ (our: slos)
- Case Management ❌ (MISSING - NEW in 2025!)
- Workflow Automation ✅ (our: workflows)
- App Builder ❌ (MISSING - NEW in 2025!)
- Bits AI SRE ❌ (MISSING - NEW in 2025!)
- Watchdog ✅ (our: watchdog)
- Event Management ❌ (MISSING)

### 6. AI
- LLM Observability ✅ (our: llm)
- AI Integrations ❌ (MISSING)
- Bits AI Agents ❌ (MISSING - NEW in 2025!)
- Bits AI SRE ❌ (MISSING - NEW in 2025!)
- Watchdog ✅ (our: watchdog)
- Event Management ❌ (MISSING)

### 7. Platform Capabilities
- Alerts ❌ (MISSING - via monitors but not standalone)
- Dashboards ✅ (our: dashboards)
- Notebooks ❌ (MISSING)
- Mobile App ❌ (N/A for CLI)
- Fleet Automation ❌ (MISSING)
- Access Control ❌ (MISSING)
- Integrations ❌ (MISSING - management)
- API ✅ (our: entire CLI uses API)
- Marketplace ❌ (MISSING - integration marketplace)
- DORA Metrics ❌ (MISSING - NEW!)

---

## Current vs. Proposed Command Categories

### Current Structure (Our CLI)

**Core Commands (3):**
- health, deploy, context

**Observability (9):**
- logs, metrics, rum, network, database, apm, llm, security, watchdog

**Incident & Alert Management (3):**
- incidents, monitors, slos

**Advanced Features (3):**
- synthetics, cicd, workflows

**Infrastructure (3):**
- catalog, dashboards, cost

**Total: 21 commands**

---

### Proposed Structure (Aligned with Datadog)

**1. Observability (15-20 commands)**

**Infrastructure Monitoring:**
- `dd infra` - Infrastructure metrics and monitoring
- `dd network` - Network Performance Monitoring ✅
- `dd containers` - Container monitoring (NEW)
- `dd kubernetes` - Kubernetes monitoring and autoscaling (NEW)
- `dd serverless` - Serverless monitoring (NEW)
- `dd storage` - Storage management (NEW)
- `dd cost` - Cloud cost management ✅

**Application Monitoring:**
- `dd apm` - Application Performance Monitoring ✅
- `dd profiler` - Continuous profiling (NEW)
- `dd llm` - LLM Observability ✅
- `dd instrumentation` - Dynamic instrumentation (NEW)
- `dd service-map` - Universal Service Monitoring (NEW)

**Data Monitoring:**
- `dd database` - Database monitoring ✅
- `dd data-streams` - Data streams monitoring (NEW)
- `dd data-quality` - Data quality monitoring (NEW)
- `dd jobs` - Jobs monitoring (NEW)

**Logs & Events:**
- `dd logs` - Log management ✅
- `dd events` - Event management (NEW)
- `dd pipelines` - Observability pipelines (NEW)
- `dd audit` - Audit trail (NEW)

**Digital Experience:**
- `dd rum` - Real User Monitoring ✅
- `dd synthetics` - Synthetic monitoring ✅
- `dd session-replay` - Session replay (NEW)
- `dd analytics` - Product analytics (NEW)
- `dd errors` - Error tracking (NEW)

**2. Security (10-15 commands - NEW CATEGORY)**

**Code Security:**
- `dd code-scan` - Code security scanning
- `dd sca` - Software Composition Analysis
- `dd sast` - Static Application Security Testing
- `dd iast` - Interactive Application Security Testing
- `dd iac-scan` - Infrastructure as Code security
- `dd secrets` - Secret scanning

**Cloud Security:**
- `dd security` - Cloud security overview (EXISTING - expand)
- `dd cspm` - Cloud Security Posture Management
- `dd ciem` - Cloud Infrastructure Entitlement Management
- `dd vulnerabilities` - Vulnerability management
- `dd compliance` - Compliance monitoring

**Threat Management:**
- `dd siem` - Cloud SIEM
- `dd workload-protection` - Workload protection
- `dd app-protection` - Application & API protection

**3. Software Delivery (8-10 commands - NEW CATEGORY)**

- `dd cicd` - CI/CD visibility ✅
- `dd tests` - Test optimization
- `dd coverage` - Code coverage
- `dd gates` - Deployment gates
- `dd dora` - DORA metrics
- `dd feature-flags` - Feature flags management
- `dd developer-portal` - Internal Developer Portal (IDP)

**4. Service Management (10-12 commands)**

- `dd incidents` - Incident management ✅
- `dd cases` - Case management (NEW)
- `dd monitors` - Monitor management ✅
- `dd slos` - Service Level Objectives ✅
- `dd downtimes` - Downtime scheduling (NEW)
- `dd on-call` - On-call scheduling (NEW)
- `dd status-pages` - Status pages (NEW)
- `dd workflows` - Workflow automation ✅
- `dd app-builder` - App builder (NEW)
- `dd catalog` - Service catalog ✅
- `dd watchdog` - AI anomaly detection ✅

**5. AI & Automation (5-8 commands - NEW CATEGORY)**

- `dd ai-agents` - Bits AI Agents
- `dd ai-sre` - Bits AI SRE
- `dd ai-security` - Bits AI Security Analyst
- `dd llm` - LLM Observability ✅ (also in Observability)
- `dd experiments` - LLM Experiments
- `dd watchdog` - Watchdog ✅ (also in Service Management)

**6. Platform & Administration (8-10 commands - NEW CATEGORY)**

- `dd dashboards` - Dashboard management ✅
- `dd notebooks` - Notebooks management
- `dd teams` - Team management
- `dd users` - User management
- `dd roles` - Role management
- `dd integrations` - Integration management
- `dd api-keys` - API key management
- `dd org` - Organization management
- `dd fleet` - Fleet automation

**7. Core Utilities (3-5 commands)**

- `dd health` - Multi-signal service health ✅
- `dd deploy` - Deployment readiness check ✅
- `dd context` - Service context detection ✅
- `dd metrics` - Timeseries metrics (MOVE from Observability)
- `dd version` - CLI version

---

## Gap Analysis

### Currently Implemented: 21/22 commands (95%)

### High-Priority Additions (Based on 2025 Announcements):

**1. AI & New Features (2025 DASH Announcements):**
- Product Analytics (GA in 2025)
- Internal Developer Portal (major launch)
- Case Management (service management)
- App Builder (workflow automation)
- Bits AI suite (AI SRE, AI Agents, AI Security)
- DORA Metrics (DevOps metrics)
- Status Pages (incident communication)
- GPU Monitoring (AI workload monitoring)

**2. Security Commands (Large Gap):**
- Full security category missing (15+ potential commands)
- Only basic `security` command exists
- CSPM, CIEM, SIEM, secret scanning all missing

**3. Software Delivery (Partial Coverage):**
- Have CI/CD but missing test optimization
- Missing code coverage, feature flags
- Missing deployment gates, DORA metrics

**4. Container & Kubernetes:**
- No container monitoring commands
- No Kubernetes-specific commands
- Major gap for modern cloud-native apps

**5. Advanced Observability:**
- Missing continuous profiler
- Missing dynamic instrumentation
- Missing data streams monitoring
- Missing jobs monitoring
- Missing session replay

---

## Recommended Expansion Plan

### Phase 1: High-Value Quick Wins (Next 2-3 Iterations)

**Priority 1 - Align with 2025 Announcements:**
1. `dd analytics` - Product Analytics (GA in 2025)
2. `dd dora` - DORA Metrics (hot topic)
3. `dd cases` - Case Management (new service)
4. `dd status-pages` - Status Pages (incident comms)
5. `dd on-call` - On-Call Scheduling (service management)

**Priority 2 - Container & Cloud Native:**
6. `dd containers` - Container monitoring
7. `dd kubernetes` - Kubernetes monitoring
8. `dd serverless` - Serverless monitoring

**Priority 3 - Security Basics:**
9. `dd secrets` - Secret scanning (2025 release)
10. `dd cspm` - Cloud Security Posture Management
11. `dd vulnerabilities` - Vulnerability management

**Expected Impact:** +11 commands (21 → 32 commands, ~145% growth)

### Phase 2: Security & Compliance (Iterations 4-6)

**Code Security:**
12. `dd code-scan` - Code security
13. `dd sca` - Software Composition Analysis
14. `dd sast` - Static analysis
15. `dd iast` - Interactive analysis
16. `dd iac-scan` - IaC security

**Threat Management:**
17. `dd siem` - Cloud SIEM
18. `dd workload-protection` - Workload security
19. `dd app-protection` - App/API protection
20. `dd compliance` - Compliance monitoring

**Expected Impact:** +9 commands (32 → 41 commands)

### Phase 3: Advanced Observability (Iterations 7-10)

**Application Insights:**
21. `dd profiler` - Continuous profiling
22. `dd instrumentation` - Dynamic instrumentation
23. `dd service-map` - Service mapping
24. `dd errors` - Error tracking
25. `dd session-replay` - Session replay

**Data & Pipelines:**
26. `dd data-streams` - Data streams monitoring
27. `dd data-quality` - Data quality
28. `dd jobs` - Jobs monitoring
29. `dd pipelines` - Observability pipelines
30. `dd audit` - Audit trail

**Expected Impact:** +10 commands (41 → 51 commands)

### Phase 4: AI & Automation (Iterations 11-13)

31. `dd ai-agents` - Bits AI Agents
32. `dd ai-sre` - Bits AI SRE
33. `dd ai-security` - Bits AI Security Analyst
34. `dd experiments` - LLM Experiments
35. `dd app-builder` - App Builder

**Software Delivery:**
36. `dd tests` - Test optimization
37. `dd coverage` - Code coverage
38. `dd gates` - Deployment gates
39. `dd feature-flags` - Feature flags

**Expected Impact:** +9 commands (51 → 60 commands)

### Phase 5: Platform & Administration (Iterations 14-16)

40. `dd notebooks` - Notebooks
41. `dd teams` - Team management
42. `dd users` - User management
43. `dd roles` - Role management
44. `dd integrations` - Integration management
45. `dd api-keys` - API key management
46. `dd org` - Organization management
47. `dd fleet` - Fleet automation
48. `dd downtimes` - Downtime scheduling
49. `dd events` - Event management
50. `dd storage` - Storage management

**Expected Impact:** +11 commands (60 → 71 commands)

---

## Proposed Final Command Structure

**Total Commands: ~70-75**

1. **Observability** (20 commands)
   - Infrastructure, Applications, Data, Logs, Digital Experience

2. **Security** (15 commands)
   - Code Security, Cloud Security, Threat Management

3. **Software Delivery** (10 commands)
   - CI/CD, Testing, Deployment, DevOps Metrics

4. **Service Management** (12 commands)
   - Incidents, Monitoring, SLOs, Automation, Catalog

5. **AI & Automation** (8 commands)
   - AI Agents, AI SRE, LLM Observability, Experiments

6. **Platform & Administration** (10 commands)
   - Dashboards, Teams, Users, Integrations, Organization

7. **Core Utilities** (5 commands)
   - Health, Deploy, Context, Metrics, Version

---

## Implementation Considerations

### API Availability
All proposed commands map to documented Datadog API v2 endpoints:
- Authentication
- Infrastructure & Cloud Integrations
- Observability (APM, Logs, RUM, Synthetics)
- Security (CSM, Application Security, SIEM)
- Service Management (Incidents, Case Management, On-Call)
- Software Delivery (CI Visibility, DORA Metrics)
- Administration (Users, Teams, Roles, API Keys)

### Licensing & Prior Art
- **datadog-ci** (Apache 2.0): CI/CD commands, DORA, coverage, SARIF
- **datadog-sync-cli** (BSD-3-Clause): Resource sync patterns
- Both provide excellent reference implementations

### Command Naming Conventions
Follow Datadog's product naming:
- Use official product names (e.g., `cspm` not `security-posture`)
- Hyphenate multi-word commands (e.g., `service-map` not `servicemap`)
- Keep commands concise (e.g., `rum` not `real-user-monitoring`)

### Backward Compatibility
- Maintain all 21 existing commands
- Only add new commands, never break existing ones
- Consider aliases for renamed categories

---

## Success Metrics

**Current State:**
- Commands: 21
- Categories: 5
- API Coverage: ~25%
- Product Alignment: ~30%

**Target State (End of Phase 5):**
- Commands: 70+
- Categories: 7 (aligned with Datadog)
- API Coverage: ~80%
- Product Alignment: ~90%

**Key Milestones:**
- Phase 1: +50% commands (32 total)
- Phase 2: +95% commands (41 total)
- Phase 3: +143% commands (51 total)
- Phase 4: +186% commands (60 total)
- Phase 5: +238% commands (71 total)

---

## Next Steps (Iteration 27)

1. Review and approve expansion plan
2. Prioritize Phase 1 commands (11 new commands)
3. Create API client methods for new endpoints
4. Implement first 3-5 Phase 1 commands
5. Update documentation and plugin skills
6. Test new commands with real data

---

**Created:** January 22, 2026
**Author:** Ralph Loop Iteration 26
**Status:** Proposal for Review
**Impact:** 238% command growth over 16 iterations
