#!/usr/bin/env python3
"""
Query Datadog Kubernetes Monitoring
Retrieves K8s pod, deployment, and cluster metrics
"""

import sys
import os
import json
import argparse
from typing import Dict, Any
import time

# Add lib directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'lib'))

from datadog_client import DatadogClient

def parse_duration(duration: str) -> tuple:
    """Convert duration string to epoch timestamps"""
    now = int(time.time())
    seconds = 3600  # default 1 hour

    if duration.endswith('m'):
        seconds = int(duration[:-1]) * 60
    elif duration.endswith('h'):
        seconds = int(duration[:-1]) * 3600
    elif duration.endswith('d'):
        seconds = int(duration[:-1]) * 86400

    return now - seconds, now

def build_query(namespace: str, pod: str, deployment: str, cluster: str, tags: str) -> str:
    """Build query filter"""
    filters = []

    if namespace:
        filters.append(f'kube_namespace:{namespace}')
    if pod:
        filters.append(f'pod_name:{pod}')
    if deployment:
        filters.append(f'kube_deployment:{deployment}')
    if cluster:
        filters.append(f'kube_cluster_name:{cluster}')
    if tags:
        filters.append(tags)

    return ','.join(filters) if filters else '*'

def query_kubernetes(client: DatadogClient, duration: str, namespace: str,
                    pod: str, deployment: str, cluster: str, tags: str) -> Dict[str, Any]:
    """Query Kubernetes metrics"""

    from_ts, to_ts = parse_duration(duration)
    query_filter = build_query(namespace, pod, deployment, cluster, tags)

    # Query CPU usage
    cpu_query = f'avg:kubernetes.cpu.usage.total{{{query_filter}}} by {{pod_name,kube_namespace,kube_deployment}}'
    cpu_data = client.query_metrics(cpu_query, from_ts, to_ts)

    # Query memory usage
    mem_query = f'avg:kubernetes.memory.usage{{{query_filter}}} by {{pod_name,kube_namespace,kube_deployment}}'
    mem_data = client.query_metrics(mem_query, from_ts, to_ts)

    # Query pod count
    pod_query = f'sum:kubernetes.pods.running{{{query_filter}}} by {{kube_namespace,kube_deployment}}'
    pod_data = client.query_metrics(pod_query, from_ts, to_ts)

    # Query node status
    node_query = f'avg:kubernetes.kubelet.running_pods{{{query_filter}}} by {{host,kube_cluster_name}}'
    node_data = client.query_metrics(node_query, from_ts, to_ts)

    # Calculate averages
    cpu_series = cpu_data.get('series', [])
    mem_series = mem_data.get('series', [])
    pod_series = pod_data.get('series', [])
    node_series = node_data.get('series', [])

    def avg_values(series):
        if not series:
            return 0
        all_values = []
        for s in series:
            all_values.extend([p[1] for p in s.get('pointlist', [])])
        return sum(all_values) / len(all_values) if all_values else 0

    # Extract namespaces
    namespaces = set()
    for s in cpu_series:
        scope = s.get('scope', '')
        for part in scope.split(','):
            if part.startswith('kube_namespace:'):
                namespaces.add(part.split(':')[1])

    return {
        'status': 'success',
        'query': query_filter,
        'duration': duration,
        'kubernetes': {
            'pods': {
                'cpu_usage': {
                    'metric': 'kubernetes.cpu.usage.total',
                    'series': cpu_series,
                    'avg_nanocores': avg_values(cpu_series)
                },
                'memory_usage': {
                    'metric': 'kubernetes.memory.usage',
                    'series': mem_series,
                    'avg_bytes': avg_values(mem_series)
                },
                'running_count': {
                    'metric': 'kubernetes.pods.running',
                    'series': pod_series,
                    'current': pod_series[0]['pointlist'][-1][1] if pod_series and pod_series[0].get('pointlist') else 0
                }
            },
            'nodes': {
                'running_pods': {
                    'metric': 'kubernetes.kubelet.running_pods',
                    'series': node_series,
                    'total': avg_values(node_series)
                }
            }
        },
        'summary': {
            'pod_count': len(cpu_series),
            'avg_cpu_nanocores': avg_values(cpu_series),
            'avg_memory_mb': avg_values(mem_series) / 1048576 if avg_values(mem_series) > 0 else 0,
            'node_count': len(node_series),
            'namespaces': list(namespaces)
        }
    }

def main():
    parser = argparse.ArgumentParser(description='Query Kubernetes monitoring')
    parser.add_argument('--duration', default='1h', help='Time duration (e.g., 1h, 30m, 7d)')
    parser.add_argument('--namespace', help='Kubernetes namespace')
    parser.add_argument('--pod', help='Pod name')
    parser.add_argument('--deployment', help='Deployment name')
    parser.add_argument('--cluster', help='Cluster name')
    parser.add_argument('--tags', help='Additional tag filters')

    args = parser.parse_args()

    try:
        client = DatadogClient()
        result = query_kubernetes(
            client, args.duration, args.namespace, args.pod,
            args.deployment, args.cluster, args.tags
        )
        print(json.dumps(result, indent=2))

    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
