# Context Graph Skill

Build and query a knowledge graph of codebase relationships to accelerate understanding.

## What This Skill Does

- **Scan**: Analyze codebase structure, imports, configs, and data flows
- **Build**: Create a relationship graph stored as JSON
- **Query**: Answer questions about dependencies, data flows, and component relationships
- **Update**: Incrementally update graph as codebase changes

## Usage

### Build Context Graph
```bash
python3 scripts/context_graph.py build [--path /path/to/codebase]
```

### Query the Graph
```bash
# What depends on a component?
python3 scripts/context_graph.py query --depends-on "kafka-service"

# What does a component depend on?
python3 scripts/context_graph.py query --dependencies "polecat-operator"

# Show data flow from source to sink
python3 scripts/context_graph.py query --flow "github-issues" "llm-response"

# List all components of a type
python3 scripts/context_graph.py query --type "kafka-topic"

# Show component details
python3 scripts/context_graph.py query --show "polecat-github-syncer"
```

### Export Graph
```bash
# Export as DOT for visualization
python3 scripts/context_graph.py export --format dot > graph.dot

# Export as Mermaid
python3 scripts/context_graph.py export --format mermaid

# Export summary
python3 scripts/context_graph.py export --format summary
```

## Graph Schema

```json
{
  "nodes": {
    "node-id": {
      "type": "service|topic|dag|file|config|secret",
      "name": "human readable name",
      "path": "/path/to/definition",
      "metadata": {}
    }
  },
  "edges": [
    {
      "from": "source-node-id",
      "to": "target-node-id",
      "type": "depends|produces|consumes|imports|configures",
      "metadata": {}
    }
  ]
}
```

## Supported Component Types

| Type | Description | Detection |
|------|-------------|-----------|
| `k8s-deployment` | Kubernetes Deployment | YAML with kind: Deployment |
| `k8s-service` | Kubernetes Service | YAML with kind: Service |
| `k8s-configmap` | Kubernetes ConfigMap | YAML with kind: ConfigMap |
| `k8s-crd` | Custom Resource | YAML with custom apiVersion |
| `kafka-topic` | Kafka Topic | Referenced in configs/code |
| `airflow-dag` | Airflow DAG | Python files with DAG() |
| `go-package` | Go Package | go.mod, *.go files |
| `node-module` | Node.js Module | package.json |
| `python-module` | Python Module | setup.py, pyproject.toml |
| `helm-chart` | Helm Chart | Chart.yaml |
| `dockerfile` | Container Image | Dockerfile |

## Integration with Claude Code

When this skill is loaded, Claude can:
1. Quickly answer "what talks to X?" questions
2. Understand data flows without reading every file
3. Identify impact of changes
4. Navigate complex microservice architectures

## Graph Storage

Graph is stored at `.claude/context-graph.json` in the project root.
