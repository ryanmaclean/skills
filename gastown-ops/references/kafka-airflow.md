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

