#!/usr/bin/env python3
"""
Context Graph Builder and Query Tool

Builds a knowledge graph of codebase relationships for faster understanding.
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path
from typing import Dict, List, Optional, Set, Tuple
from dataclasses import dataclass, asdict
from collections import defaultdict
try:
    import yaml
    HAS_YAML = True
except ImportError:
    HAS_YAML = False
    # Fallback: simple YAML parser for basic K8s docs
    class yaml:
        @staticmethod
        def safe_load(f):
            import re
            content = f.read() if hasattr(f, 'read') else f
            # Very basic YAML to dict for simple K8s manifests
            result = {}
            for line in content.split('\n'):
                if ':' in line and not line.strip().startswith('#'):
                    key, _, val = line.partition(':')
                    key = key.strip()
                    val = val.strip().strip('"').strip("'")
                    if key and val:
                        result[key] = val
            return result

        @staticmethod
        def safe_load_all(f):
            content = f.read() if hasattr(f, 'read') else f
            docs = content.split('---')
            return [yaml.safe_load(d) for d in docs if d.strip()]


@dataclass
class Node:
    id: str
    type: str
    name: str
    path: Optional[str] = None
    metadata: Optional[Dict] = None

    def to_dict(self):
        return {k: v for k, v in asdict(self).items() if v is not None}


@dataclass
class Edge:
    source: str
    target: str
    type: str
    metadata: Optional[Dict] = None

    def to_dict(self):
        d = {"from": self.source, "to": self.target, "type": self.type}
        if self.metadata:
            d["metadata"] = self.metadata
        return d


class ContextGraph:
    def __init__(self):
        self.nodes: Dict[str, Node] = {}
        self.edges: List[Edge] = []
        self.adjacency: Dict[str, Set[str]] = defaultdict(set)
        self.reverse_adjacency: Dict[str, Set[str]] = defaultdict(set)

    def add_node(self, node: Node):
        self.nodes[node.id] = node

    def add_edge(self, edge: Edge):
        self.edges.append(edge)
        self.adjacency[edge.source].add(edge.target)
        self.reverse_adjacency[edge.target].add(edge.source)

    def get_dependencies(self, node_id: str) -> List[str]:
        """What does this node depend on?"""
        return list(self.adjacency.get(node_id, set()))

    def get_dependents(self, node_id: str) -> List[str]:
        """What depends on this node?"""
        return list(self.reverse_adjacency.get(node_id, set()))

    def find_path(self, source: str, target: str) -> Optional[List[str]]:
        """Find path from source to target using BFS."""
        if source not in self.nodes or target not in self.nodes:
            return None

        visited = {source}
        queue = [(source, [source])]

        while queue:
            current, path = queue.pop(0)
            if current == target:
                return path

            for neighbor in self.adjacency.get(current, set()):
                if neighbor not in visited:
                    visited.add(neighbor)
                    queue.append((neighbor, path + [neighbor]))

        return None

    def get_nodes_by_type(self, node_type: str) -> List[Node]:
        """Get all nodes of a specific type."""
        return [n for n in self.nodes.values() if n.type == node_type]

    def to_dict(self) -> Dict:
        return {
            "nodes": {k: v.to_dict() for k, v in self.nodes.items()},
            "edges": [e.to_dict() for e in self.edges]
        }

    @classmethod
    def from_dict(cls, data: Dict) -> "ContextGraph":
        graph = cls()
        for node_id, node_data in data.get("nodes", {}).items():
            graph.add_node(Node(
                id=node_id,
                type=node_data.get("type", "unknown"),
                name=node_data.get("name", node_id),
                path=node_data.get("path"),
                metadata=node_data.get("metadata")
            ))
        for edge_data in data.get("edges", []):
            graph.add_edge(Edge(
                source=edge_data["from"],
                target=edge_data["to"],
                type=edge_data.get("type", "depends"),
                metadata=edge_data.get("metadata")
            ))
        return graph

    def to_mermaid(self) -> str:
        """Export as Mermaid flowchart."""
        lines = ["flowchart TD"]

        # Group nodes by type
        type_styles = {
            "kafka-topic": "([%s])",
            "k8s-deployment": "[%s]",
            "airflow-dag": "{{%s}}",
            "k8s-configmap": "[(%s)]",
            "k8s-crd": "[[%s]]",
        }

        for node in self.nodes.values():
            style = type_styles.get(node.type, "[%s]")
            safe_name = node.name.replace("-", "_").replace(".", "_")
            label = style % node.name
            lines.append(f"    {safe_name}{label}")

        for edge in self.edges:
            src = edge.source.replace("-", "_").replace(".", "_")
            tgt = edge.target.replace("-", "_").replace(".", "_")
            arrow = "-->" if edge.type == "depends" else "-.->|%s|" % edge.type
            if edge.type in ["produces", "consumes"]:
                arrow = f"-->|{edge.type}|"
            lines.append(f"    {src} {arrow} {tgt}")

        return "\n".join(lines)

    def to_dot(self) -> str:
        """Export as DOT for Graphviz."""
        lines = ["digraph context {", "    rankdir=LR;"]

        type_shapes = {
            "kafka-topic": "ellipse",
            "k8s-deployment": "box",
            "airflow-dag": "hexagon",
            "k8s-configmap": "cylinder",
            "k8s-crd": "doubleoctagon",
        }

        for node in self.nodes.values():
            shape = type_shapes.get(node.type, "box")
            lines.append(f'    "{node.id}" [label="{node.name}", shape={shape}];')

        for edge in self.edges:
            style = "solid" if edge.type == "depends" else "dashed"
            lines.append(f'    "{edge.source}" -> "{edge.target}" [style={style}, label="{edge.type}"];')

        lines.append("}")
        return "\n".join(lines)

    def summary(self) -> str:
        """Generate text summary of the graph."""
        lines = ["# Context Graph Summary", ""]

        # Count by type
        type_counts = defaultdict(int)
        for node in self.nodes.values():
            type_counts[node.type] += 1

        lines.append("## Node Types")
        for t, count in sorted(type_counts.items()):
            lines.append(f"- {t}: {count}")

        lines.append("")
        lines.append(f"## Edges: {len(self.edges)}")

        edge_types = defaultdict(int)
        for edge in self.edges:
            edge_types[edge.type] += 1
        for t, count in sorted(edge_types.items()):
            lines.append(f"- {t}: {count}")

        # Key nodes (most connections)
        lines.append("")
        lines.append("## Key Nodes (most connections)")
        connection_counts = []
        for node_id in self.nodes:
            count = len(self.adjacency.get(node_id, set())) + len(self.reverse_adjacency.get(node_id, set()))
            connection_counts.append((node_id, count))

        for node_id, count in sorted(connection_counts, key=lambda x: -x[1])[:10]:
            node = self.nodes[node_id]
            lines.append(f"- {node.name} ({node.type}): {count} connections")

        return "\n".join(lines)


class GraphBuilder:
    """Builds context graph by scanning codebase."""

    def __init__(self, root_path: str):
        self.root = Path(root_path)
        self.graph = ContextGraph()
        self.kafka_topics: Set[str] = set()

    def build(self) -> ContextGraph:
        """Build the complete graph."""
        print(f"Scanning {self.root}...", file=sys.stderr)

        self._scan_kubernetes()
        self._scan_airflow_dags()
        self._scan_go_modules()
        self._scan_node_modules()
        self._scan_helm_charts()
        self._scan_dockerfiles()
        self._extract_kafka_topics()
        self._infer_relationships()

        print(f"Built graph with {len(self.graph.nodes)} nodes and {len(self.graph.edges)} edges", file=sys.stderr)
        return self.graph

    def _scan_kubernetes(self):
        """Scan for Kubernetes manifests."""
        for ext in ["yaml", "yml"]:
            for path in self.root.rglob(f"*.{ext}"):
                if self._should_skip(path):
                    continue
                try:
                    with open(path) as f:
                        docs = list(yaml.safe_load_all(f))
                    for doc in docs:
                        if not isinstance(doc, dict):
                            continue
                        self._process_k8s_doc(doc, path)
                except Exception:
                    pass

    def _process_k8s_doc(self, doc: Dict, path: Path):
        """Process a Kubernetes document."""
        kind = doc.get("kind", "")
        metadata = doc.get("metadata", {})
        name = metadata.get("name", "")

        if not kind or not name:
            return

        type_map = {
            "Deployment": "k8s-deployment",
            "Service": "k8s-service",
            "ConfigMap": "k8s-configmap",
            "Secret": "k8s-secret",
            "StatefulSet": "k8s-statefulset",
            "DaemonSet": "k8s-daemonset",
            "Job": "k8s-job",
            "CronJob": "k8s-cronjob",
        }

        node_type = type_map.get(kind)
        if not node_type:
            # Check for CRDs
            api_version = doc.get("apiVersion", "")
            if "/" in api_version and api_version.split("/")[0] not in ["apps", "v1", "batch", "networking.k8s.io"]:
                node_type = "k8s-crd"

        if node_type:
            node_id = f"{kind.lower()}-{name}"
            self.graph.add_node(Node(
                id=node_id,
                type=node_type,
                name=name,
                path=str(path.relative_to(self.root)),
                metadata={"kind": kind, "apiVersion": doc.get("apiVersion")}
            ))

            # Extract references
            self._extract_k8s_references(doc, node_id)

    def _extract_k8s_references(self, doc: Dict, source_id: str):
        """Extract references from K8s document."""
        spec = doc.get("spec", {})

        # ConfigMap references
        for vol in spec.get("template", {}).get("spec", {}).get("volumes", []):
            if "configMap" in vol:
                cm_name = vol["configMap"].get("name")
                if cm_name:
                    target_id = f"configmap-{cm_name}"
                    self.graph.add_edge(Edge(source_id, target_id, "mounts"))

        # Service references in env
        for container in spec.get("template", {}).get("spec", {}).get("containers", []):
            for env in container.get("env", []):
                val = env.get("value", "")
                # Look for service references like "service-name:port"
                if re.match(r"^[\w-]+-service:\d+$", val):
                    svc_name = val.split(":")[0]
                    target_id = f"service-{svc_name}"
                    self.graph.add_edge(Edge(source_id, target_id, "connects"))

    def _scan_airflow_dags(self):
        """Scan for Airflow DAG files."""
        for path in self.root.rglob("*.py"):
            if self._should_skip(path):
                continue
            try:
                content = path.read_text()
                if "DAG(" in content or "with DAG" in content:
                    # Extract DAG ID
                    match = re.search(r'dag_id\s*=\s*["\']([^"\']+)["\']', content)
                    if not match:
                        match = re.search(r'DAG\(\s*["\']([^"\']+)["\']', content)

                    if match:
                        dag_id = match.group(1)
                        self.graph.add_node(Node(
                            id=f"dag-{dag_id}",
                            type="airflow-dag",
                            name=dag_id,
                            path=str(path.relative_to(self.root))
                        ))

                        # Extract Kafka topics
                        for topic_match in re.finditer(r'topic\s*=\s*["\']([^"\']+)["\']', content):
                            topic = topic_match.group(1)
                            self.kafka_topics.add(topic)
                            self.graph.add_edge(Edge(f"dag-{dag_id}", f"topic-{topic}", "produces"))
            except Exception:
                pass

    def _scan_go_modules(self):
        """Scan for Go modules."""
        for path in self.root.rglob("go.mod"):
            if self._should_skip(path):
                continue
            try:
                content = path.read_text()
                match = re.search(r'^module\s+(\S+)', content, re.MULTILINE)
                if match:
                    module_name = match.group(1)
                    self.graph.add_node(Node(
                        id=f"go-{module_name.replace('/', '-')}",
                        type="go-module",
                        name=module_name,
                        path=str(path.parent.relative_to(self.root))
                    ))
            except Exception:
                pass

    def _scan_node_modules(self):
        """Scan for Node.js packages."""
        for path in self.root.rglob("package.json"):
            if self._should_skip(path) or "node_modules" in str(path):
                continue
            try:
                with open(path) as f:
                    pkg = json.load(f)
                name = pkg.get("name", path.parent.name)
                self.graph.add_node(Node(
                    id=f"node-{name}",
                    type="node-module",
                    name=name,
                    path=str(path.parent.relative_to(self.root)),
                    metadata={"version": pkg.get("version")}
                ))
            except Exception:
                pass

    def _scan_helm_charts(self):
        """Scan for Helm charts."""
        for path in self.root.rglob("Chart.yaml"):
            if self._should_skip(path):
                continue
            try:
                with open(path) as f:
                    chart = yaml.safe_load(f)
                name = chart.get("name", path.parent.name)
                self.graph.add_node(Node(
                    id=f"helm-{name}",
                    type="helm-chart",
                    name=name,
                    path=str(path.parent.relative_to(self.root)),
                    metadata={"version": chart.get("version")}
                ))
            except Exception:
                pass

    def _scan_dockerfiles(self):
        """Scan for Dockerfiles."""
        for path in self.root.rglob("Dockerfile*"):
            if self._should_skip(path):
                continue
            name = path.parent.name
            self.graph.add_node(Node(
                id=f"docker-{name}",
                type="dockerfile",
                name=name,
                path=str(path.relative_to(self.root))
            ))

    def _extract_kafka_topics(self):
        """Extract Kafka topics from codebase."""
        topic_patterns = [
            r'topic\s*[:=]\s*["\']([^"\']+)["\']',
            r'KAFKA_TOPIC[S]?\s*[:=]\s*["\']([^"\']+)["\']',
            r'topics?\s*:\s*["\']([^"\']+)["\']',
            r'\.subscribe\(\s*\{\s*topic:\s*["\']([^"\']+)["\']',
            r'\.send\(\s*\{\s*topic:\s*["\']([^"\']+)["\']',
        ]

        for ext in ["js", "ts", "py", "go", "yaml", "yml"]:
            for path in self.root.rglob(f"*.{ext}"):
                if self._should_skip(path):
                    continue
                try:
                    content = path.read_text()
                    for pattern in topic_patterns:
                        for match in re.finditer(pattern, content, re.IGNORECASE):
                            topic = match.group(1)
                            if not topic.startswith("$") and not topic.startswith("{"):
                                self.kafka_topics.add(topic)
                except Exception:
                    pass

        # Add topic nodes
        for topic in self.kafka_topics:
            self.graph.add_node(Node(
                id=f"topic-{topic}",
                type="kafka-topic",
                name=topic
            ))

    def _infer_relationships(self):
        """Infer relationships between components."""
        # Infer Kafka producer/consumer relationships
        for ext in ["js", "ts", "py", "go"]:
            for path in self.root.rglob(f"*.{ext}"):
                if self._should_skip(path):
                    continue
                try:
                    content = path.read_text()

                    # Find which component this file belongs to
                    component_id = self._find_component_for_file(path)
                    if not component_id:
                        continue

                    # Check for Kafka consumption
                    if re.search(r'consumer|subscribe|KAFKA_TOPICS_CONSUME', content, re.IGNORECASE):
                        for topic in self.kafka_topics:
                            if topic in content:
                                self.graph.add_edge(Edge(f"topic-{topic}", component_id, "consumed-by"))

                    # Check for Kafka production
                    if re.search(r'producer|\.send\(|KAFKA_TOPICS_PRODUCE', content, re.IGNORECASE):
                        for topic in self.kafka_topics:
                            if topic in content:
                                self.graph.add_edge(Edge(component_id, f"topic-{topic}", "produces"))
                except Exception:
                    pass

    def _find_component_for_file(self, path: Path) -> Optional[str]:
        """Find which component a file belongs to."""
        # Check for package.json in ancestors
        for parent in path.parents:
            if parent == self.root:
                break
            pkg_json = parent / "package.json"
            if pkg_json.exists():
                try:
                    with open(pkg_json) as f:
                        pkg = json.load(f)
                    return f"node-{pkg.get('name', parent.name)}"
                except Exception:
                    pass

            # Check for go.mod
            go_mod = parent / "go.mod"
            if go_mod.exists():
                return f"go-{parent.name}"

        return None

    def _should_skip(self, path: Path) -> bool:
        """Check if path should be skipped."""
        skip_dirs = {"node_modules", ".git", "vendor", "__pycache__", ".venv", "venv", "dist", "build"}
        return any(d in path.parts for d in skip_dirs)


def save_graph(graph: ContextGraph, path: Path):
    """Save graph to JSON file."""
    with open(path, "w") as f:
        json.dump(graph.to_dict(), f, indent=2)
    print(f"Saved graph to {path}", file=sys.stderr)


def load_graph(path: Path) -> Optional[ContextGraph]:
    """Load graph from JSON file."""
    if not path.exists():
        return None
    try:
        with open(path) as f:
            data = json.load(f)
        return ContextGraph.from_dict(data)
    except Exception as e:
        print(f"Error loading graph: {e}", file=sys.stderr)
        return None


def main():
    parser = argparse.ArgumentParser(description="Context Graph Builder and Query Tool")
    subparsers = parser.add_subparsers(dest="command", help="Commands")

    # Build command
    build_parser = subparsers.add_parser("build", help="Build context graph")
    build_parser.add_argument("--path", default=".", help="Path to codebase root")
    build_parser.add_argument("--output", help="Output file (default: .claude/context-graph.json)")

    # Query command
    query_parser = subparsers.add_parser("query", help="Query context graph")
    query_parser.add_argument("--graph", help="Path to graph file")
    query_parser.add_argument("--depends-on", help="Find what depends on a node")
    query_parser.add_argument("--dependencies", help="Find dependencies of a node")
    query_parser.add_argument("--flow", nargs=2, metavar=("FROM", "TO"), help="Find path between nodes")
    query_parser.add_argument("--type", help="List nodes of a type")
    query_parser.add_argument("--show", help="Show node details")
    query_parser.add_argument("--search", help="Search nodes by name")

    # Export command
    export_parser = subparsers.add_parser("export", help="Export graph")
    export_parser.add_argument("--graph", help="Path to graph file")
    export_parser.add_argument("--format", choices=["dot", "mermaid", "summary", "json"], default="summary")

    args = parser.parse_args()

    if args.command == "build":
        root = Path(args.path).resolve()
        builder = GraphBuilder(str(root))
        graph = builder.build()

        output_path = args.output
        if not output_path:
            output_path = root / ".claude" / "context-graph.json"
            output_path.parent.mkdir(parents=True, exist_ok=True)
        else:
            output_path = Path(output_path)

        save_graph(graph, output_path)
        print(graph.summary())

    elif args.command == "query":
        graph_path = Path(args.graph) if args.graph else Path(".claude/context-graph.json")
        graph = load_graph(graph_path)

        if not graph:
            print(f"No graph found at {graph_path}. Run 'build' first.", file=sys.stderr)
            sys.exit(1)

        if args.depends_on:
            # Find partial matches
            matches = [n for n in graph.nodes if args.depends_on.lower() in n.lower()]
            for node_id in matches:
                dependents = graph.get_dependents(node_id)
                print(f"\n{node_id}:")
                for dep in dependents:
                    node = graph.nodes.get(dep)
                    if node:
                        print(f"  <- {node.name} ({node.type})")

        elif args.dependencies:
            matches = [n for n in graph.nodes if args.dependencies.lower() in n.lower()]
            for node_id in matches:
                deps = graph.get_dependencies(node_id)
                print(f"\n{node_id}:")
                for dep in deps:
                    node = graph.nodes.get(dep)
                    if node:
                        print(f"  -> {node.name} ({node.type})")

        elif args.flow:
            src_matches = [n for n in graph.nodes if args.flow[0].lower() in n.lower()]
            tgt_matches = [n for n in graph.nodes if args.flow[1].lower() in n.lower()]

            for src in src_matches:
                for tgt in tgt_matches:
                    path = graph.find_path(src, tgt)
                    if path:
                        print(f"\nPath from {src} to {tgt}:")
                        print(" -> ".join(path))

        elif args.type:
            nodes = graph.get_nodes_by_type(args.type)
            print(f"\nNodes of type '{args.type}':")
            for node in nodes:
                print(f"  - {node.name}" + (f" ({node.path})" if node.path else ""))

        elif args.show:
            matches = [n for n in graph.nodes if args.show.lower() in n.lower()]
            for node_id in matches:
                node = graph.nodes[node_id]
                print(f"\n{node.name} ({node.type})")
                if node.path:
                    print(f"  Path: {node.path}")
                if node.metadata:
                    print(f"  Metadata: {json.dumps(node.metadata)}")

                deps = graph.get_dependencies(node_id)
                if deps:
                    print(f"  Dependencies ({len(deps)}):")
                    for d in deps[:5]:
                        print(f"    -> {d}")
                    if len(deps) > 5:
                        print(f"    ... and {len(deps) - 5} more")

                dependents = graph.get_dependents(node_id)
                if dependents:
                    print(f"  Dependents ({len(dependents)}):")
                    for d in dependents[:5]:
                        print(f"    <- {d}")
                    if len(dependents) > 5:
                        print(f"    ... and {len(dependents) - 5} more")

        elif args.search:
            matches = [n for n in graph.nodes.values() if args.search.lower() in n.name.lower()]
            print(f"\nSearch results for '{args.search}':")
            for node in matches:
                print(f"  - {node.name} ({node.type})")

    elif args.command == "export":
        graph_path = Path(args.graph) if args.graph else Path(".claude/context-graph.json")
        graph = load_graph(graph_path)

        if not graph:
            print(f"No graph found at {graph_path}. Run 'build' first.", file=sys.stderr)
            sys.exit(1)

        if args.format == "dot":
            print(graph.to_dot())
        elif args.format == "mermaid":
            print(graph.to_mermaid())
        elif args.format == "summary":
            print(graph.summary())
        elif args.format == "json":
            print(json.dumps(graph.to_dict(), indent=2))

    else:
        parser.print_help()


if __name__ == "__main__":
    main()
