#!/usr/bin/env python3
"""
Tundra Dome Context Graph Builder

Specialized graph builder for Tundra Dome that includes:
- Live K8s state (polecats, topics, deployments)
- HP PA-RISC Apollo metaphor mappings
- Lane-based routing configuration
- LLM model assignments
"""

import json
import os
import subprocess
import sys
from pathlib import Path
from dataclasses import dataclass, asdict
from typing import Dict, List, Optional, Set
from collections import defaultdict


@dataclass
class TundraNode:
    id: str
    type: str
    name: str
    apollo_type: Optional[str] = None  # Dome, Town, Polecat, Bead, Lane
    path: Optional[str] = None
    k8s_status: Optional[str] = None
    metadata: Optional[Dict] = None

    def to_dict(self):
        return {k: v for k, v in asdict(self).items() if v is not None}


@dataclass
class TundraEdge:
    source: str
    target: str
    type: str
    lane: Optional[str] = None  # experimental, standard, critical
    metadata: Optional[Dict] = None

    def to_dict(self):
        d = {"from": self.source, "to": self.target, "type": self.type}
        if self.lane:
            d["lane"] = self.lane
        if self.metadata:
            d["metadata"] = self.metadata
        return d


class TundraDomeGraph:
    """Tundra Dome context graph with Apollo metaphor mappings."""

    # Apollo metaphor mappings
    APOLLO_MAPPINGS = {
        "Dome": "K8s Cluster - The entire computational domain",
        "Town": "K8s Namespace - A logical grouping of workloads",
        "Polecat": "K8s Pod - A worker unit that processes beads",
        "Bead": "Work Item - A unit of work (issue, PR, task)",
        "Lane": "Priority Queue - Routing tier (experimental/standard/critical)",
        "Mayor": "Controller - Oversees town operations",
        "Sling": "Kafka - Message transport between components",
    }

    # Lane to model mapping
    LANE_MODELS = {
        "experimental": {"provider": "openrouter", "model": "deepseek/deepseek-r1-0528:free"},
        "standard": {"provider": "openrouter", "model": "meta-llama/llama-3.3-70b-instruct:free"},
        "critical": {"provider": "anthropic", "model": "claude-opus-4-5-20251101"},
    }

    def __init__(self):
        self.nodes: Dict[str, TundraNode] = {}
        self.edges: List[TundraEdge] = []
        self.adjacency: Dict[str, Set[str]] = defaultdict(set)
        self.reverse_adjacency: Dict[str, Set[str]] = defaultdict(set)

    def add_node(self, node: TundraNode):
        self.nodes[node.id] = node

    def add_edge(self, edge: TundraEdge):
        self.edges.append(edge)
        self.adjacency[edge.source].add(edge.target)
        self.reverse_adjacency[edge.target].add(edge.source)

    def to_dict(self) -> Dict:
        return {
            "apollo_mappings": self.APOLLO_MAPPINGS,
            "lane_models": self.LANE_MODELS,
            "nodes": {k: v.to_dict() for k, v in self.nodes.items()},
            "edges": [e.to_dict() for e in self.edges]
        }

    def get_by_apollo_type(self, apollo_type: str) -> List[TundraNode]:
        return [n for n in self.nodes.values() if n.apollo_type == apollo_type]

    def get_by_lane(self, lane: str) -> List[TundraEdge]:
        return [e for e in self.edges if e.lane == lane]

    def data_flow_summary(self) -> str:
        """Generate a data flow summary."""
        lines = ["# Tundra Dome Data Flow", ""]

        # Bead lifecycle
        lines.append("## Bead Lifecycle")
        lines.append("```")
        lines.append("GitHub Issue/PR")
        lines.append("    │")
        lines.append("    ▼ (Airflow DAG: github_sync)")
        lines.append("tundra-work-intake (Kafka)")
        lines.append("    │")
        lines.append("    ├─────────────────────────────────────┐")
        lines.append("    ▼                                     ▼")
        lines.append("experimental lane                   standard lane")
        lines.append("(deepseek-r1:free)                 (llama-3.3-70b:free)")
        lines.append("    │                                     │")
        lines.append("    └─────────────┬─────────────────────┘")
        lines.append("                  ▼")
        lines.append("           Polecat Worker")
        lines.append("                  │")
        lines.append("                  ▼")
        lines.append("         OpenRouter LLM Call")
        lines.append("         (ddtrace instrumented)")
        lines.append("                  │")
        lines.append("    ┌─────────────┴─────────────┐")
        lines.append("    ▼                           ▼")
        lines.append("tundra-beads-completed    tundra-beads-failed")
        lines.append("```")
        lines.append("")

        # Lane routing
        lines.append("## Lane Routing")
        for lane, config in self.LANE_MODELS.items():
            lines.append(f"- **{lane}**: {config['provider']}:{config['model']}")

        return "\n".join(lines)

    def architecture_summary(self) -> str:
        """Generate architecture summary."""
        lines = ["# Tundra Dome Architecture (Apollo Metaphors)", ""]

        for apollo, desc in self.APOLLO_MAPPINGS.items():
            lines.append(f"## {apollo}")
            lines.append(f"_{desc}_")
            lines.append("")

            nodes = self.get_by_apollo_type(apollo)
            if nodes:
                for node in nodes:
                    status = f" [{node.k8s_status}]" if node.k8s_status else ""
                    lines.append(f"- {node.name}{status}")
            else:
                lines.append("_No live instances_")
            lines.append("")

        return "\n".join(lines)


def kubectl_get_json(resource: str, namespace: str = "tundra-dome") -> Optional[Dict]:
    """Get K8s resource as JSON."""
    try:
        result = subprocess.run(
            ["kubectl", "get", resource, "-n", namespace, "-o", "json"],
            capture_output=True, text=True, timeout=10
        )
        if result.returncode == 0:
            return json.loads(result.stdout)
    except Exception:
        pass
    return None


def build_live_graph() -> TundraDomeGraph:
    """Build graph from live K8s state."""
    graph = TundraDomeGraph()

    # Add the Dome (cluster)
    graph.add_node(TundraNode(
        id="dome-tundra",
        type="cluster",
        name="Tundra Dome",
        apollo_type="Dome",
        metadata={"namespace": "tundra-dome"}
    ))

    # Add the Town (namespace)
    graph.add_node(TundraNode(
        id="town-tundra-dome",
        type="namespace",
        name="tundra-dome",
        apollo_type="Town"
    ))
    graph.add_edge(TundraEdge("dome-tundra", "town-tundra-dome", "contains"))

    # Get Polecats (from CRD)
    polecats = kubectl_get_json("polecats")
    if polecats and "items" in polecats:
        for pc in polecats["items"]:
            name = pc["metadata"]["name"]
            spec = pc.get("spec", {})
            status = pc.get("status", {})

            graph.add_node(TundraNode(
                id=f"polecat-{name}",
                type="polecat-crd",
                name=name,
                apollo_type="Polecat",
                k8s_status=status.get("phase", "Unknown"),
                metadata={
                    "role": spec.get("role", "polecat"),
                    "skills": spec.get("skills", []),
                    "lanes": spec.get("lanes", []),
                    "replicas": spec.get("replicas", 1),
                    "kafka_consume": spec.get("kafkaTopics", {}).get("consume", []),
                    "kafka_produce": spec.get("kafkaTopics", {}).get("produce", []),
                }
            ))
            graph.add_edge(TundraEdge("town-tundra-dome", f"polecat-{name}", "contains"))

            # Add Kafka topic relationships
            for topic in spec.get("kafkaTopics", {}).get("consume", []):
                topic_id = f"topic-{topic}"
                if topic_id not in graph.nodes:
                    graph.add_node(TundraNode(
                        id=topic_id, type="kafka-topic", name=topic, apollo_type="Sling"
                    ))
                graph.add_edge(TundraEdge(topic_id, f"polecat-{name}", "consumed-by"))

            for topic in spec.get("kafkaTopics", {}).get("produce", []):
                topic_id = f"topic-{topic}"
                if topic_id not in graph.nodes:
                    graph.add_node(TundraNode(
                        id=topic_id, type="kafka-topic", name=topic, apollo_type="Sling"
                    ))
                graph.add_edge(TundraEdge(f"polecat-{name}", topic_id, "produces"))

    # Get Pods for status
    pods = kubectl_get_json("pods")
    if pods and "items" in pods:
        for pod in pods["items"]:
            name = pod["metadata"]["name"]
            labels = pod["metadata"].get("labels", {})
            phase = pod["status"]["phase"]

            # Match to polecat
            polecat_name = labels.get("tundra.dome/polecat")
            if polecat_name and f"polecat-{polecat_name}" in graph.nodes:
                graph.nodes[f"polecat-{polecat_name}"].k8s_status = phase

    # Add Lanes
    for lane in ["experimental", "standard", "critical"]:
        model_config = TundraDomeGraph.LANE_MODELS[lane]
        graph.add_node(TundraNode(
            id=f"lane-{lane}",
            type="lane",
            name=lane,
            apollo_type="Lane",
            metadata=model_config
        ))

    # Get Airflow DAGs (from configmap)
    dags_cm = kubectl_get_json("configmap/airflow-dags")
    if dags_cm and "data" in dags_cm:
        for dag_file in dags_cm["data"].keys():
            if dag_file.endswith(".py"):
                dag_name = dag_file.replace(".py", "")
                graph.add_node(TundraNode(
                    id=f"dag-{dag_name}",
                    type="airflow-dag",
                    name=dag_name,
                    apollo_type="Mayor"
                ))

    return graph


def main():
    import argparse
    parser = argparse.ArgumentParser(description="Tundra Dome Context Graph")
    parser.add_argument("command", choices=["build", "flow", "arch", "polecats", "topics", "lanes"])
    parser.add_argument("--output", help="Output file")
    args = parser.parse_args()

    graph = build_live_graph()

    if args.command == "build":
        output = args.output or ".claude/tundra-dome-graph.json"
        Path(output).parent.mkdir(parents=True, exist_ok=True)
        with open(output, "w") as f:
            json.dump(graph.to_dict(), f, indent=2)
        print(f"Saved Tundra Dome graph to {output}", file=sys.stderr)
        print(f"Nodes: {len(graph.nodes)}, Edges: {len(graph.edges)}")

    elif args.command == "flow":
        print(graph.data_flow_summary())

    elif args.command == "arch":
        print(graph.architecture_summary())

    elif args.command == "polecats":
        print("# Polecats (Workers)")
        for node in graph.get_by_apollo_type("Polecat"):
            print(f"\n## {node.name}")
            print(f"Status: {node.k8s_status or 'Unknown'}")
            if node.metadata:
                print(f"Role: {node.metadata.get('role')}")
                print(f"Skills: {', '.join(node.metadata.get('skills', []))}")
                print(f"Lanes: {', '.join(node.metadata.get('lanes', []))}")
                print(f"Consumes: {', '.join(node.metadata.get('kafka_consume', []))}")
                print(f"Produces: {', '.join(node.metadata.get('kafka_produce', []))}")

    elif args.command == "topics":
        print("# Kafka Topics (Slings)")
        for node in graph.get_by_apollo_type("Sling"):
            consumers = [e.target for e in graph.edges if e.source == node.id and e.type == "consumed-by"]
            producers = [e.source for e in graph.edges if e.target == node.id and e.type == "produces"]
            print(f"\n## {node.name}")
            if producers:
                print(f"  Producers: {', '.join(producers)}")
            if consumers:
                print(f"  Consumers: {', '.join(consumers)}")

    elif args.command == "lanes":
        print("# Lanes (Priority Routing)")
        for lane, config in TundraDomeGraph.LANE_MODELS.items():
            print(f"\n## {lane}")
            print(f"  Provider: {config['provider']}")
            print(f"  Model: {config['model']}")


if __name__ == "__main__":
    main()
