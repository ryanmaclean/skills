#!/usr/bin/env python3
"""
Manage Datadog Monitor Downtimes
Handles listing, creating, and canceling downtimes for scheduled maintenance
"""

import sys
import os
import json
import argparse
from datetime import datetime
from typing import Dict, Any, Optional

# Add lib directory to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'lib'))

from datadog_client import DatadogClient

def list_downtimes(client: DatadogClient, current_only: bool = False) -> Dict[str, Any]:
    """List all downtimes"""
    params = {}
    if current_only:
        params['current_only'] = 'true'

    data = client.get('/api/v2/downtime', params=params)

    downtimes = []
    for item in data.get('data', []):
        attrs = item.get('attributes', {})
        downtimes.append({
            'id': item.get('id'),
            'message': attrs.get('message'),
            'scope': attrs.get('scope'),
            'status': attrs.get('status'),
            'start': attrs.get('schedule', {}).get('start'),
            'end': attrs.get('schedule', {}).get('end'),
            'recurring': attrs.get('schedule', {}).get('recurrences') is not None,
            'timezone': attrs.get('display_timezone')
        })

    return {
        'status': 'success',
        'total_downtimes': data.get('meta', {}).get('page', {}).get('total_count', 0),
        'active_count': len([d for d in downtimes if d['status'] == 'active']),
        'scheduled_count': len([d for d in downtimes if d['status'] == 'scheduled']),
        'downtimes': downtimes
    }

def create_downtime(client: DatadogClient, message: str, scope: Optional[str],
                   start: str, end: Optional[str], duration: Optional[str],
                   rrule: Optional[str], timezone: str, monitor_id: Optional[int],
                   monitor_tags: Optional[str]) -> Dict[str, Any]:
    """Create a new downtime"""

    # Build attributes
    attributes = {
        'display_timezone': timezone
    }

    if message:
        attributes['message'] = message

    if scope:
        attributes['scope'] = scope

    # Build monitor identifier
    if monitor_id:
        attributes['monitor_identifier'] = {'monitor_id': monitor_id}
    elif monitor_tags:
        tags = [t.strip() for t in monitor_tags.split(',')]
        attributes['monitor_identifier'] = {'monitor_tags': tags}

    # Build schedule
    schedule = {}
    if rrule:
        # Recurring downtime
        recurrence = {'rrule': rrule}
        if start:
            recurrence['start'] = start
        if duration:
            recurrence['duration'] = duration
        schedule['recurrences'] = [recurrence]
        schedule['timezone'] = timezone
    else:
        # One-time downtime
        schedule['start'] = start
        if end:
            schedule['end'] = end

    attributes['schedule'] = schedule

    payload = {
        'data': {
            'type': 'downtime',
            'attributes': attributes
        }
    }

    data = client.post('/api/v2/downtime', json=payload)

    dt_data = data.get('data', {})
    dt_attrs = dt_data.get('attributes', {})

    return {
        'status': 'success',
        'downtime': {
            'id': dt_data.get('id'),
            'message': dt_attrs.get('message'),
            'scope': dt_attrs.get('scope'),
            'status': dt_attrs.get('status'),
            'start': dt_attrs.get('schedule', {}).get('start'),
            'end': dt_attrs.get('schedule', {}).get('end'),
            'recurring': dt_attrs.get('schedule', {}).get('recurrences') is not None,
            'timezone': dt_attrs.get('display_timezone')
        }
    }

def cancel_downtime(client: DatadogClient, downtime_id: str) -> Dict[str, Any]:
    """Cancel a downtime"""
    client.delete(f'/api/v2/downtime/{downtime_id}')

    return {
        'status': 'success',
        'message': f'Downtime {downtime_id} canceled successfully'
    }

def main():
    parser = argparse.ArgumentParser(description='Manage Datadog monitor downtimes')
    parser.add_argument('--action', default='list', choices=['list', 'create', 'cancel'],
                       help='Action to perform')
    parser.add_argument('--downtime-id', help='Downtime ID (for cancel)')
    parser.add_argument('--message', help='Downtime message')
    parser.add_argument('--scope', help='Scope filter (e.g., env:prod)')
    parser.add_argument('--monitor-id', type=int, help='Monitor ID to mute')
    parser.add_argument('--monitor-tags', help='Monitor tags (comma-separated)')
    parser.add_argument('--start', help='Start time (RFC3339)')
    parser.add_argument('--end', help='End time (RFC3339)')
    parser.add_argument('--duration', help='Duration (for recurring, e.g., 2h)')
    parser.add_argument('--rrule', help='Recurrence rule')
    parser.add_argument('--timezone', default='UTC', help='Timezone')
    parser.add_argument('--current-only', action='store_true', help='Show only active downtimes')

    args = parser.parse_args()

    try:
        client = DatadogClient()

        if args.action == 'list':
            result = list_downtimes(client, args.current_only)
        elif args.action == 'create':
            if not args.start:
                raise ValueError('--start is required for creating downtime')
            result = create_downtime(
                client, args.message, args.scope, args.start, args.end,
                args.duration, args.rrule, args.timezone, args.monitor_id, args.monitor_tags
            )
        elif args.action == 'cancel':
            if not args.downtime_id:
                raise ValueError('--downtime-id is required for cancel action')
            result = cancel_downtime(client, args.downtime_id)

        print(json.dumps(result, indent=2))

    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
