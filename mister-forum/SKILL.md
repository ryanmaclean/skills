---
name: mister-forum
description: Browse and search MiSTer FPGA forum (misterfpga.org) efficiently via SearXNG cache
trigger: mister forum, misterfpga, fpga forum, core status, mister news, mister development
---

# MiSTer FPGA Forum Browser

Search and browse misterfpga.org threads without hammering the site.

## Usage

- "search mister forum for CPS3 core status"
- "check misterfpga.org for SNES save state updates"
- "find mister forum threads about translation overlay"

## How It Works

1. **SearXNG first** — search via SearXNG to find relevant threads without crawling:
   ```
   WebFetch http://10.0.3.35:8080/search?q=site:misterfpga.org+{query}&format=json
   ```
   Prompt: "Extract all search results: title, URL, and snippet for each result."

2. **Parse results** — extract thread URLs, titles, dates, and snippet text from the JSON response.

3. **Direct fetch only when needed** — if the user asks for full thread content, fetch the specific URL:
   ```
   WebFetch https://misterfpga.org/viewtopic.php?t=THREAD_ID
   ```
   Prompt: "Extract forum posts: author, date, and post content. Summarize the thread."

4. **Rate limit** — max 1 direct page fetch per 10 seconds to misterfpga.org. Never crawl index pages.

## Commands

### search <query>
Search forum via SearXNG. URL-encode the query and use `+` for spaces:
```
WebFetch http://10.0.3.35:8080/search?q=site:misterfpga.org+{url_encoded_query}&format=json
```
Prompt: "List all search results with title, URL, and snippet text."

If SearXNG is unreachable (QNAS macvlan connectivity — confirmed timeout from Mac as of 2026-04-04), fall back to:
```
WebSearch site:misterfpga.org {query}   (with allowed_domains: ["misterfpga.org"])
```
Note: WebSearch fallback is currently the reliable path from Mac. SearXNG will work from any /23 host that can reach 10.0.3.35 (or 10.0.2.35:32768 via host port forward).

### thread <url>
Fetch and parse a specific phpBB thread page. Extract:
- Thread title
- Each post: author, date, content (trimmed)
- Pagination links if multi-page

Use prompt: "Parse this phpBB forum thread. For each post extract: author username, post date, and post content. List them in chronological order. Note if there are additional pages."

### latest <section>
Check latest posts in a forum section. Search SearXNG with a recent date filter:
```
WebFetch http://10.0.3.35:8080/search?q=site:misterfpga.org+{section}&format=json&time_range=month
```

## Cache Strategy

- WebFetch has a built-in 15-minute cache — repeated fetches of the same URL are free.
- For longer persistence, summarize findings in the conversation and reference them rather than re-fetching.
- Never fetch forum index/listing pages — always use SearXNG to find specific threads.

## phpBB Structure Notes

misterfpga.org runs phpBB. Key URL patterns:
- Thread: `https://misterfpga.org/viewtopic.php?t=ID`
- Forum section: `https://misterfpga.org/viewforum.php?f=ID`
- User profile: `https://misterfpga.org/memberlist.php?mode=viewprofile&u=ID`
- Search: `https://misterfpga.org/search.php?keywords=QUERY`

Key forum sections (by topic, not stable IDs):
- Core Development
- General Discussion
- Cores (Arcade, Console, Computer)
- Hardware / Cases / Accessories

## Etiquette

- This is a community forum. Do not scrape, crawl, or mass-fetch pages.
- SearXNG search results are sufficient for most queries.
- Only fetch individual thread pages when the user specifically needs full post content.
- Space direct fetches at least 10 seconds apart.
