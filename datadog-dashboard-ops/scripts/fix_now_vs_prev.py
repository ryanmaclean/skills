#!/usr/bin/env python3
import json
import sys
from pathlib import Path

def load_map(path: Path):
    mapping = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        title, metric, include_ai = [part.strip() for part in line.split('|')]
        mapping[title] = (metric, include_ai.lower() == 'true')
    return mapping

if len(sys.argv) < 3:
    print("usage: fix_now_vs_prev.py <dashboard.json> <mapping.txt>")
    sys.exit(1)

path = Path(sys.argv[1])
map_path = Path(sys.argv[2])

obj = json.loads(path.read_text())
map_metrics = load_map(map_path)

updated = 0
for w in obj.get('widgets', []):
    d = w.get('definition', {})
    title = d.get('title')
    if title in map_metrics and d.get('type') == 'timeseries':
        metric, include_ai = map_metrics[title]
        queries = [
            {"data_source": "metrics", "name": "query1", "query": f"sum:{metric}{{*}}"}
        ]
        if include_ai:
            queries.append({"data_source": "metrics", "name": "query2", "query": f"sum:ai_agent.{metric}{{*}}"})
        formulas = [
            {"formula": "query1" + (" + query2" if include_ai else ""), "alias": "now"},
            {"formula": "timeshift(query1" + (" + query2" if include_ai else "") + ", '1h')", "alias": "prev_1h"},
            {"formula": "timeshift(query1" + (" + query2" if include_ai else "") + ", '$compare_window')", "alias": "prev_window"},
        ]
        d['requests'][0]['queries'] = queries
        d['requests'][0]['formulas'] = formulas
        updated += 1

path.write_text(json.dumps(obj, indent=2))
print(updated)
