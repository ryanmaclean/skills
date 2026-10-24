#!/usr/bin/env python3
"""
Query Datadog Network Performance Monitoring
Retrieves network flow data, connections, and latency metrics
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

def build_query(source: str, dest: str, port: str, tags: str) -> str:
    """Build query filter"""
    filters = []

    if source:
        filters.append(f'source_service:{source}')
    if dest:
        filters.append(f'dest_service:{dest}')
    if port:
        filters.append(f'dest_port:{port}')
    if tags:
        filters.append(tags)

    return ','.join(filters) if filters else '*'

def query_network(client: DatadogClient, duration: str, source: str,
                 dest: str, port: str, tags: str) -> Dict[str, Any]:
    """Query network metrics"""

    from_ts, to_ts = parse_duration(duration)
    query_filter = build_query(source, dest, port, tags)

    # Query bytes sent
    sent_query = f'sum:network.bytes_sent{{{query_filter}}} by {{source_service,dest_service}}'
    sent_data = client.query_metrics(sent_query, from_ts, to_ts)

    # Query bytes received
    rcvd_query = f'sum:network.bytes_rcvd{{{query_filter}}} by {{source_service,dest_service}}'
    rcvd_data = client.query_metrics(rcvd_query, from_ts, to_ts)

    # Query TCP latency
    lat_query = f'avg:network.tcp.rtt{{{query_filter}}} by {{source_service,dest_service}}'
    lat_data = client.query_metrics(lat_query, from_ts, to_ts)

    # Query connections
    conn_query = f'sum:network.tcp.connections{{{query_filter}}} by {{source_service,dest_service}}'
    conn_data = client.query_metrics(conn_query, from_ts, to_ts)

    # Query retransmits
    retrans_query = f'sum:network.tcp.retransmits{{{query_filter}}} by {{source_service,dest_service}}'
    retrans_data = client.query_metrics(retrans_query, from_ts, to_ts)

    # Calculate totals
    sent_series = sent_data.get('series', [])
    rcvd_series = rcvd_data.get('series', [])
    lat_series = lat_data.get('series', [])
    conn_series = conn_data.get('series', [])
    retrans_series = retrans_data.get('series', [])

    def sum_values(series):
        if not series:
            return 0
        total = 0
        for s in series:
            total += sum([p[1] for p in s.get('pointlist', [])])
        return total

    def avg_values(series):
        if not series:
            return 0
        all_values = []
        for s in series:
            all_values.extend([p[1] for p in s.get('pointlist', [])])
        return sum(all_values) / len(all_values) if all_values else 0

    return {
        'status': 'success',
        'query': query_filter,
        'duration': duration,
        'network': {
            'bytes_sent': {
                'metric': 'network.bytes_sent',
                'series': sent_series,
                'total': sum_values(sent_series)
            },
            'bytes_received': {
                'metric': 'network.bytes_rcvd',
                'series': rcvd_series,
                'total': sum_values(rcvd_series)
            },
            'tcp_latency': {
                'metric': 'network.tcp.rtt',
                'series': lat_series,
                'avg_ms': avg_values(lat_series)
            },
            'connections': {
                'metric': 'network.tcp.connections',
                'series': conn_series,
                'total': sum_values(conn_series)
            },
            'retransmits': {
                'metric': 'network.tcp.retransmits',
                'series': retrans_series,
                'total': sum_values(retrans_series)
            }
        },
        'summary': {
            'total_bytes_sent': sum_values(sent_series),
            'total_bytes_received': sum_values(rcvd_series),
            'avg_latency_ms': avg_values(lat_series),
            'total_connections': sum_values(conn_series),
            'total_retransmits': sum_values(retrans_series),
            'flow_count': len(sent_series)
        }
    }

def main():
    parser = argparse.ArgumentParser(description='Query network performance monitoring')
    parser.add_argument('--duration', default='1h', help='Time duration (e.g., 1h, 30m, 7d)')
    parser.add_argument('--source', help='Source service')
    parser.add_argument('--dest', '--destination', help='Destination service')
    parser.add_argument('--port', help='Destination port')
    parser.add_argument('--tags', help='Additional tag filters')

    args = parser.parse_args()

    try:
        client = DatadogClient()
        result = query_network(
            client, args.duration, args.source,
            args.dest or args.destination, args.port, args.tags
        )
        print(json.dumps(result, indent=2))

    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
