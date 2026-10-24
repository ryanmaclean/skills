#!/usr/bin/env python3
"""Manage Datadog Roles"""
import sys, json, argparse
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent / "lib"))
from datadog_client import create_client

def main():
    parser = argparse.ArgumentParser(description="Manage Datadog Roles")
    parser.add_argument('--action', choices=['list', 'get'], default='list')
    parser.add_argument('--role-id', help="Role ID")
    args = parser.parse_args()

    try:
        client = create_client()
        if args.action == 'list':
            data = client.get('/api/v2/roles')
            result = {'status': 'success', 'roles': [{'id': r['id'], 'name': r['attributes']['name']} for r in data.get('data', [])]}
        else:
            data = client.get(f'/api/v2/roles/{args.role_id}')
            result = {'status': 'success', 'role': data['data']}
        print(json.dumps(result, indent=2))
    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__': main()
