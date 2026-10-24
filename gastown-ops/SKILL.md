# Gas Town Operations Skill

Manage Gas Town agent orchestration, bead lifecycle, Kafka messaging, Airflow DAGs, and multi-rig coordination.

## Vocabulary

Gas Town extends the Tundra Dome frontier settlement metaphor for AI agent orchestration:

| Term | Equivalent | Description |
|------|------------|-------------|
| **Gas Town** | Agent Orchestration Layer | The coordination system for AI agents |
| **Bead** | Work Unit (Issue/PR/Task) | Atomic unit of work flowing through the system |
| **Lane** | Priority Queue | Critical/Standard/Experimental routing |
| **Rig** | Machine/Environment | Compute resource running agents |
| **Polecat** | Worker Agent | Individual AI agent instance |
| **Crew** | Agent Group | Set of polecats working together |
| **Nudge** | Agent Notification | Prompt or reminder to an agent |
| **Whisper** | Internal Message | Private inter-agent communication |
| **Mail** | Formal Message | Structured agent-to-agent communication |
| **Playbook** | Airflow DAG | Automated workflow sequence |
| **Errand** | Airflow Task | Single step in a playbook |

### Agent Roles

| Role | Responsibility | Kafka Command Topic |
|------|----------------|---------------------|
| **Mayor** | Orchestration, scheduling, high-level commands | `tundra-mayor-commands` |
| **Deacon** | CI/CD, builds, deployments, health checks | `tundra-deacon-commands` |
| **Polecat** | General worker, executes tasks | `tundra-polecat-commands` |
| **Reaper** | Cleanup, garbage collection, pruning | `tundra-reaper-commands` |
| **Witness** | Observation, logging, auditing | `tundra-witness-commands` |
| **Overseer** | Validation, quality checks, enforcement | `tundra-overseer-commands` |

## Bead Lifecycle

```
created --> hooked --> assigned --> working --> completed
                                        |
                                        +--> escalated --> (re-assigned)
                                        |
                                        +--> failed --> (dead-letter)
```

### Bead States

| State | Description |
|-------|-------------|
| `pending` | Bead created, awaiting assignment |
| `synced` | Synchronized across rigs |
| `conflict` | Version conflict between rigs |
| `failed` | Processing failed |
| `in-progress` | Being worked on |
| `completed` | Successfully finished |
| `escalated` | Escalated for human review |

## Commands

### gt CLI - Agent Management

```bash
# Check Gas Town status
gt status

# List all rigs
gt rig list

# Add a new rig
gt rig add <rig-name> --git-url <repo-url>

# Start/stop Mayor daemon
gt mayor start
gt mayor stop

# Start/stop Deacon daemon
gt deacon start
gt deacon stop

# Start/stop Witness daemon
gt witness start
gt witness stop

# Spawn a polecat on a rig
gt polecat spawn --rig <rig-name> --type vibecode

# List active polecats
gt polecat list

# Kill a polecat
gt polecat kill <polecat-id>
```

### Bead Operations

```bash
# List beads
bd list
bd list --status open
bd list --status in_progress

# Get bead details
bd show <bead-id>

# Create a bead
bd create --title "Fix login timeout" --priority 1 --type issue

# Count beads with filters
bd count --created-after "2026-02-01" --by-priority
bd count --status blocked --updated-after "2026-02-01"

# Update bead status
bd update <bead-id> --status in_progress
bd update <bead-id> --status completed

# Assign bead to agent
bd assign <bead-id> --to <agent-id>
```

### Kubernetes Bead Operations

```bash
# List all beads in Tundra Dome
kubectl get beads -n tundra-dome

# Get bead details
kubectl describe bead <bead-name> -n tundra-dome

# Create a bead
kubectl apply -f - <<EOF
apiVersion: tundra.dome/v1
kind: Bead
metadata:
  name: fix-bug-123
  namespace: tundra-dome
spec:
  source: github
  sourceId: "123"
  title: "Fix login timeout"
  lane: standard
  assignedRole: polecat
EOF

# Delete a bead
kubectl delete bead <bead-name> -n tundra-dome

# Watch bead status changes
kubectl get beads -n tundra-dome -w
```

### Kafka Operations

```bash
# Port forward to Kafka
kubectl port-forward svc/kafka 9092:9092 -n tundra-dome

# List topics (using kafka-topics.sh or kafkacat)
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Produce to a command topic
echo '{"action": "nudge", "target": "polecat-123"}' | \
  kafka-console-producer.sh --broker-list localhost:9092 \
  --topic tundra-polecat-commands

# Consume from bead lifecycle topic
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic tundra-beads-created --from-beginning

# Using td tools (Rust TUI)
cd td/tools/kafka_tui && cargo run

# Check queue status
td/scripts/kafka-queue-status.sh
```

### Airflow DAG Management

```bash
# Port forward to Airflow UI
kubectl port-forward svc/airflow-api-service 8080:8080 -n tundra-dome
# Visit http://localhost:8080 (credentials: tundra/admin)

# List playbooks
kubectl get playbooks -n tundra-dome

# Trigger a playbook
kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags trigger <dag_id>

# Trigger specific playbooks
kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags trigger tundra_lane_router

kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags trigger tundra_superdome_policies

# Pause/unpause a DAG
kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags pause <dag_id>

kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags unpause <dag_id>

# List DAG runs
kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags list-runs -d <dag_id>
```

### Multi-Rig Coordination

```bash
# Sync beads across rigs
gt sync --rig <rig-name>

# Check sync status
gt sync status

# Force sync resolution
gt sync resolve <bead-id> --strategy local
gt sync resolve <bead-id> --strategy remote
gt sync resolve <bead-id> --strategy merge

# View rig configuration
cat mayor/rigs.json

# View daemon patrol config
cat mayor/daemon.json

# Check deacon health state
cat deacon/health-check-state.json
cat deacon/heartbeat.json
```

### Metrics and Observability

```bash
# Send Gas Town metrics to Datadog
python scripts/send_gastown_metrics.py

# Dry run to see metrics
python scripts/send_gastown_metrics.py --dry-run

# Send metrics with lookback window
python scripts/send_gastown_metrics.py --since 1h

# Batch send for dashboard population
python scripts/send_gastown_metrics.py --batch

# Check KPI snapshot
cat ~/gt/logs/kpi_snapshot.json

# Update Datadog dashboard
python scripts/update_datadog_dashboard.py
```

## Kafka Topics

### Command Topics (Control Plane)

- `tundra-mayor-commands` - Mayor orchestration commands
- `tundra-deacon-commands` - Deacon CI/CD commands
- `tundra-polecat-commands` - Polecat work commands
- `tundra-reaper-commands` - Reaper cleanup commands
- `tundra-witness-commands` - Witness observation commands
- `tundra-overseer-commands` - Overseer validation commands

### Bead Lifecycle Topics

- `tundra-beads-created` - New beads
- `tundra-beads-in-progress` - Beads being worked
- `tundra-beads-completed` - Finished beads
- `tundra-beads-escalated` - Escalated beads
- `tundra-beads-failed` - Failed beads

### Lane Topics (Fabric)

- `tundra-lane-critical-beads` - P0/P1 urgent work
- `tundra-lane-standard-beads` - Normal priority work
- `tundra-lane-experimental-beads` - Low-priority/experimental

### Communication Topics

- `tundra-nudges` - Agent notifications/reminders
- `tundra-whispers` - Private inter-agent messages
- `tundra-mail-outbox` - Outbound formal messages
- `tundra-mail-inbox` - Inbound formal messages

### Operations Topics

- `tundra-work-intake` - New work entering system
- `tundra-audit-actions` - Compliance/audit log
- `tundra-metrics-kpi` - KPI snapshots
- `tundra-schema-dlq` - Dead letter queue for schema errors

### Errand Lifecycle Topics

- `tundra-errands-queued` - Task queued
- `tundra-errands-running` - Task started
- `tundra-errands-success` - Task succeeded
- `tundra-errands-failed` - Task failed
- `tundra-errands-retry` - Task retrying
- `tundra-errands-skipped` - Task skipped

## Airflow Playbooks (DAGs)

### Control Loop DAGs

| DAG ID | Schedule | Purpose |
|--------|----------|---------|
| `tundra_lane_router` | `*/1 * * * *` | Routes beads to correct lanes based on KPI |
| `tundra_lane_watchdog` | `*/2 * * * *` | Escalates on bead failure spikes |
| `tundra_role_allocator` | `*/2 * * * *` | Nudges when role activity drops |
| `tundra_auto_recovery` | `*/5 * * * *` | Reassigns work on agent churn |
| `tundra_escalation_gate` | `*/1 * * * *` | Escalates stale escalations |
| `tundra_ci_cd_rigor` | `*/5 * * * *` | Escalates CI failure spikes |
| `tundra_mail_watchdog` | `*/5 * * * *` | Nudges on mail backlog |
| `tundra_nudge_watchdog` | `*/5 * * * *` | Nudges on unanswered nudges |
| `tundra_convoy_watchdog` | `*/10 * * * *` | Escalates convoy stalls |
| `tundra_merge_watchdog` | `*/10 * * * *` | Nudges on merge backlog |

### Integration DAGs

| DAG ID | Purpose |
|--------|---------|
| `tundra_github_ollama` | Sync GitHub issues to beads |
| `tundra_dome_integrations` | Periodic integration health checks |
| `tundra_dome_kafka_emit` | Emit events to Kafka |
| `tundra_gitea_cicd` | Gitea CI/CD integration |
| `tundra_sling_ingest` | Data ingestion pipelines |
| `tundra_maintenance_drain` | Worker drain/maintenance |

## Operational Runbooks

### Starting Gas Town

```bash
# 1. Verify Tundra Dome cluster is running
kind get clusters
kubectl cluster-info --context kind-tundra-dome

# 2. Start core services
gt mayor start
gt deacon start
gt witness start

# 3. Verify services are running
gt status

# 4. Send initial metrics
python scripts/send_gastown_metrics.py --batch
```

### Handling Stuck Beads

```bash
# 1. Identify stuck beads
bd list --status in_progress
bd count --status blocked

# 2. Check bead details
bd show <bead-id>
kubectl describe bead <bead-name> -n tundra-dome

# 3. Force escalation
bd update <bead-id> --status escalated
# Or via Kafka:
echo '{"action": "escalate", "bead_id": "<bead-id>", "reason": "stuck"}' | \
  kafka-console-producer.sh --broker-list localhost:9092 \
  --topic tundra-mayor-commands

# 4. Reassign to different polecat
bd assign <bead-id> --to <new-agent-id>
```

### Recovering from Agent Failure

```bash
# 1. Check agent health
gt status
cat deacon/health-check-state.json

# 2. Restart failed agent
gt polecat kill <polecat-id>
gt polecat spawn --rig <rig-name> --type vibecode

# 3. Check beads assigned to failed agent
bd list --assignee <failed-agent>

# 4. Trigger auto-recovery DAG
kubectl exec -it deploy/airflow-scheduler -n tundra-dome -- \
  airflow dags trigger tundra_auto_recovery
```

### Scaling Polecats

```bash
# 1. Check current capacity
gt status
gt polecat list

# 2. Scale up polecats
for i in {1..3}; do
  gt polecat spawn --rig vibecode --type vibecode
done

# 3. Scale down (drain first)
gt polecat drain <polecat-id>
gt polecat kill <polecat-id>

# 4. In Kubernetes
kubectl scale deployment <crew-name> --replicas=5 -n tundra-dome
```

### Debugging Kafka Issues

```bash
# 1. Check Kafka connectivity
kubectl port-forward svc/kafka 9092:9092 -n tundra-dome &
kafka-topics.sh --bootstrap-server localhost:9092 --list

# 2. Check consumer lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --all-groups

# 3. Check dead letter queue
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic tundra-schema-dlq --from-beginning

# 4. Replay messages (if needed)
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic tundra-beads-failed --from-beginning | \
  kafka-console-producer.sh --broker-list localhost:9092 \
  --topic tundra-work-intake
```

### KPI Threshold Tuning

```bash
# 1. View current thresholds
cat ~/gt/settings/kpi-thresholds.json

# 2. Edit thresholds
# Default thresholds:
# {
#   "beads_failed_15m": 3,
#   "role_activity_min_15m": 1,
#   "agent_churn_15m": 1,
#   "escalations_stale_4h": 1,
#   "ci_failed_15m": 1,
#   "mail_backlog_15m": 5,
#   "nudges_unanswered_15m": 3,
#   "convoy_stall_4h": 1,
#   "merge_backlog_4h": 1
# }

# 3. Verify KPI snapshot
cat ~/gt/logs/kpi_snapshot.json
```

## Files

### Configuration Files

- `mayor/daemon.json` - Daemon patrol configuration
- `mayor/overseer.json` - Overseer identity config
- `mayor/rigs.json` - Multi-rig configuration
- `deacon/health-check-state.json` - Agent health state
- `deacon/heartbeat.json` - Last heartbeat timestamp
- `deacon/.beads` - Bead tracking file
- `deacon/dogs/` - Deacon watchdog configs

### Source Code

- `src/lib/gastown/client.ts` - Gas Town webhook client
- `src/lib/gastown/types.ts` - TypeScript type definitions
- `src/lib/beads/schema.ts` - Bead sync schema definitions
- `scripts/send_gastown_metrics.py` - Metrics emission script
- `scripts/lib/ai_agent_telemetry.py` - Telemetry/tracing library

### Airflow DAGs

- `airflow/dags/tundra_superdome_policies.py` - Policy control loops
- `airflow/dags/tundra_github_ollama.py` - GitHub integration
- `airflow/dags/tundra_dome_integrations.py` - Integration health
- `airflow/dags/tundra_dome_kafka.py` - Kafka event emission

### Infrastructure

- `infra/tundra-dome/deploy.sh` - Deployment script
- `infra/tundra-dome/METAPHOR.md` - Full metaphor specification
- `superdome-ops.md` - Operational model documentation

## Environment Variables

```bash
# DogStatsD Configuration
DD_AGENT_HOST=127.0.0.1
DD_DOGSTATSD_PORT=8125
DD_ENV=studio

# Gas Town Configuration
GASTOWN_WEBHOOK_URL=<webhook-endpoint>
GASTOWN_API_TOKEN=<api-token>
GASTOWN_RIG=vibecode
GASTOWN_DEFAULT_PRIORITY=P1

# Metrics Control
GASTOWN_SKIP_GT_STATUS=false
GASTOWN_SKIP_BD=false
GASTOWN_SKIP_GIT=false
GASTOWN_SKIP_GH=false
GASTOWN_EMIT_TRACE=true
GASTOWN_EMIT_EVENT=true

# Kafka
KAFKA_BROKERS=localhost:9092
```

## Telemetry Integration

### Python Usage

```python
from scripts.lib.ai_agent_telemetry import GasTownTracing, get_telemetry

tracing = GasTownTracing(service_name="gastown")
telemetry = get_telemetry()

# Track full bead lifecycle
with tracing.track_bead_lifecycle(
    bead_id="st-abc123",
    rig="vibecode",
    priority=1,
    title="Fix login bug"
) as ctx:
    # Hook work to agent
    with tracing.track_hook(ctx, agent="vibecode/polecats/agate"):
        pass

    # Crew assigns to polecat
    with tracing.track_crew_assign(ctx, crew_member="crew", target_polecat="polecat"):
        pass

    # Polecat executes work
    with tracing.track_polecat_work(ctx, polecat="agate", rig="vibecode") as work:
        # Track LLM call
        with telemetry.track_claude_request(model="claude-3-sonnet") as llm:
            llm.set_tokens(input=100, output=50)
        work.set_outcome("success")

    # Deacon review
    with tracing.track_deacon_review(ctx, deacon="deacon", review_type="qa") as review:
        review.set_outcome("approved")
```

### TypeScript Usage

```typescript
import { GastownClient, createGastownClientFromEnv } from '@/lib/gastown/client'

const client = createGastownClientFromEnv()

await client.reportWorkflowFailure(traceId, {
  runId: 12345,
  workflowName: 'CI',
  status: 'completed',
  conclusion: 'failure',
  // ... other fields
})
```
