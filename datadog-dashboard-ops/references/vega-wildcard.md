# Vega/Vega-Lite Wildcard Widget (Datadog)

Use `wildcard` widgets with a `specification` block:

```json
{
  "definition": {
    "type": "wildcard",
    "title": "Example",
    "requests": [
      {
        "response_format": "scalar",
        "queries": [
          {"data_source": "metrics", "name": "query1", "query": "avg:metric.name{*}"}
        ]
      }
    ],
    "specification": {
      "type": "vega-lite",
      "contents": {
        "$schema": "https://vega.github.io/schema/vega-lite/v5.json",
        "data": {"name": "table1"},
        "mark": "bar",
        "encoding": {
          "x": {"field": "query1", "type": "quantitative"}
        }
      }
    }
  }
}
```

Notes:
- Use `specification` (not `spec`).
- `specification.type` must be `vega` or `vega-lite`.
- `specification.contents` contains the actual Vega/Vega-Lite spec.
