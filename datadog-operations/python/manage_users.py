#!/usr/bin/env python3
"""
Manage Datadog Users
List, invite, and manage user accounts
"""

import sys
import json
import argparse
from pathlib import Path
from typing import Dict, Any

sys.path.insert(0, str(Path(__file__).parent / "lib"))

from datadog_client import create_client


def list_users(client) -> Dict[str, Any]:
    """List all users"""
    data = client.get('/api/v2/users')
    return {
        'status': 'success',
        'total_users': len(data.get('data', [])),
        'users': [
            {
                'id': user['id'],
                'email': user['attributes']['email'],
                'name': user['attributes']['name'],
                'status': user['attributes']['status'],
                'verified': user['attributes'].get('verified', False),
                'disabled': user['attributes'].get('disabled', False)
            }
            for user in data.get('data', [])
        ]
    }


def get_user(client, user_id: str) -> Dict[str, Any]:
    """Get user details"""
    data = client.get(f'/api/v2/users/{user_id}')
    user = data['data']
    return {
        'status': 'success',
        'user': {
            'id': user['id'],
            'email': user['attributes']['email'],
            'name': user['attributes']['name'],
            'status': user['attributes']['status'],
            'roles': [r['id'] for r in user.get('relationships', {}).get('roles', {}).get('data', [])]
        }
    }


def main():
    parser = argparse.ArgumentParser(description="Manage Datadog Users")
    parser.add_argument('--action', choices=['list', 'get'], default='list')
    parser.add_argument('--user-id', help="User ID")

    args = parser.parse_args()

    try:
        client = create_client()

        if args.action == 'list':
            result = list_users(client)
        elif args.action == 'get':
            if not args.user_id:
                print('{"status":"error","message":"--user-id required"}', file=sys.stderr)
                sys.exit(1)
            result = get_user(client, args.user_id)

        print(json.dumps(result, indent=2))

    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
