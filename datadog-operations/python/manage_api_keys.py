#!/usr/bin/env python3
"""Manage Datadog API Keys"""
import sys, json, argparse
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent / "lib"))
from datadog_client import create_client

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--action', choices=['list'], default='list')
    args = parser.parse_args()
    try:
        client = create_client()
        data = client.get('/api/v2/api_keys')
        result = {'status': 'success', 'api_keys': [{'id': k['id'], 'name': k['attributes']['name']} for k in data.get('data', [])]}
        print(json.dumps(result, indent=2))
    except Exception as e:
        print(json.dumps({'status': 'error', 'message': str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__': main()
