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
