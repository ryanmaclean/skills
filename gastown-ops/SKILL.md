---
name: gastown-ops
description: Manage Gas Town agent orchestration, bead lifecycle, Kafka messaging, Airflow DAGs, and multi-rig coordination. Use for gt operations, polecat management, stuck beads, Kafka topics, and Tundra Dome workflows.
---

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


## References

- `references/kafka-airflow.md` — Kafka topics + Airflow DAG playbooks
- `references/runbooks.md` — operational runbooks, files, env vars, telemetry integration
