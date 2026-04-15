---
name: searxng
description: Use the local SearXNG instance when WebSearch is rate-limited, geo-blocked, or returns no results.
---

# searxng — Local Search Fallback

Use the local SearXNG instance when WebSearch is rate-limited, geo-blocked,
or returns no results.

## Instance

```
http://10.0.3.202:8080/
```

**Network note:** LAN-only. Reachable when connected to the studio network.
Not accessible from cloud/remote contexts. If ECONNREFUSED, you are off-network
— fall back to WebSearch.

## Usage

### JSON API (preferred for parsing)

```
WebFetch url="http://10.0.3.202:8080/search?q=<query>&format=json"
prompt="extract: titles, urls, snippets from results array"
```

### URL-encode the query

Spaces → `+` or `%20`. Special chars → percent-encoded.

Examples:
```
http://10.0.3.202:8080/search?q=rust+genai+crate+2026&format=json
http://10.0.3.202:8080/search?q=kunal732+MLX+Swift&format=json
http://10.0.3.202:8080/search?q=fastembed+rs+qdrant&format=json
```

### Filter by category

```
http://10.0.3.202:8080/search?q=<query>&categories=general&format=json
http://10.0.3.202:8080/search?q=<query>&categories=it&format=json
http://10.0.3.202:8080/search?q=<query>&categories=science&format=json
```

### JSON response structure

```json
{
  "query": "...",
  "results": [
    {
      "title": "...",
      "url": "...",
      "content": "...",
      "engine": "...",
      "score": 0.0
    }
  ],
  "answers": [],
  "suggestions": []
}
```

## When to use

- WebSearch returns 429 (rate limit)
- WebSearch returns no results for technical queries
- Need fresh results without Google/Bing geo-filtering
- Researching local network resources

## When NOT to use

- WebSearch is working fine — prefer it (returns structured search results natively)
- Query requires real-time data (SearXNG caches may lag)
