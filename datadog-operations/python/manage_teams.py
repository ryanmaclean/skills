#!/usr/bin/env python3
"""
Manage Datadog Teams
Create, update, and manage team organization
"""

import sys
import json
import argparse
from pathlib import Path
from typing import Optional, Dict, Any

sys.path.insert(0, str(Path(__file__).parent / "lib"))

from datadog_client import create_client


def list_teams(client) -> Dict[str, Any]:
    """List all teams"""
    data = client.get('/api/v2/team')
    return {
        'status': 'success',
        'total_teams': len(data.get('data', [])),
        'teams': [
            {
                'id': team['id'],
                'name': team['attributes']['name'],
                'handle': team['attributes']['handle'],
                'description': team['attributes'].get('description'),
                'link_count': team['attributes'].get('link_count', 0),
                'user_count': team['attributes'].get('user_count', 0)
            }
            for team in data.get('data', [])
        ]
    }


def get_team(client, team_id: str) -> Dict[str, Any]:
    """Get team details"""
    data = client.get(f'/api/v2/team/{team_id}')
    team = data['data']
    return {
        'status': 'success',
        'team': {
            'id': team['id'],
            'name': team['attributes']['name'],
            'handle': team['attributes']['handle'],
            'description': team['attributes'].get('description'),
            'links': team['attributes'].get('links', []),
            'members': team.get('relationships', {}).get('team_links', {}).get('data', [])
        }
    }


def create_team(client, name: str, handle: str, description: str = "") -> Dict[str, Any]:
    """Create new team"""
    payload = {
        'data': {
            'type': 'team',
            'attributes': {
                'name': name,
                'handle': handle,
                'description': description
            }
        }
    }
    data = client.post('/api/v2/team', payload)
    return {
        'status': 'success',
        'team': {
            'id': data['data']['id'],
            'name': data['data']['attributes']['name'],
            'handle': data['data']['attributes']['handle']
        }
    }


def main():
    parser = argparse.ArgumentParser(description="Manage Datadog Teams")
    parser.add_argument('--action', choices=['list', 'get', 'create'], default='list')
    parser.add_argument('--team-id', help="Team ID")
    parser.add_argument('--name', help="Team name")
    parser.add_argument('--handle', help="Team handle")
    parser.add_argument('--description', help="Team description", default="")

    args = parser.parse_args()

    try:
        client = create_client()

        if args.action == 'list':
            result = list_teams(client)
        elif args.action == 'get':
            if not args.team_id:
                print('{"status":"error","message":"--team-id required"}', file=sys.stderr)
                sys.exit(1)
            result = get_team(client, args.team_id)
        elif args.action == 'create':
            if not args.name or not args.handle:
                print('{"status":"error","message":"--name and --handle required"}', file=sys.stderr)
                sys.exit(1)
            result = create_team(client, args.name, args.handle, args.description)

        print(json.dumps(result, indent=2))

    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
