---
name: agentbridge-mcp
description: AgentBridge iPhone MCP server. Use when working with the live iOS app, calling device tools on Ryan's iPhone 12 mini, or testing the on-device agent loop.
---

# AgentBridge MCP

iPhone 12 mini running the AgentBridge iOS app at `http://Ryans-iPhone.local:8080`.
396 tools active. Use `curl` (not WebFetch — WebFetch upgrades to HTTPS, phone is HTTP-only).

## Endpoints

| Path | Method | Purpose |
|------|--------|---------|
| `/` | GET | HTML dashboard — lists all tools, shows status |
| `/health` | GET | `{"status":"ok"}` liveness probe |
| `/mcp` | POST | MCP JSON-RPC 2.0 — tool calls |
| `/mcp/tools/list` | GET | JSON list of all tool schemas |
| `/debug/chat` | POST | One-shot agent loop: `{"text":"..."}` → `{"assistant":"...","turns":N}` |
| `/debug/chat_trace` | POST | Same but returns full per-turn message history |

## Tool Call Pattern

```bash
curl -s http://Ryans-iPhone.local:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"TOOL_NAME","arguments":{}}}' \
  | python3 -m json.tool
```

Response shape:
```json
{"id":1,"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"...JSON..."}],"isError":false}}
```

## Chat / Agent Loop

```bash
# One-shot — resets history, runs agent, returns final assistant message
curl -s http://Ryans-iPhone.local:8080/debug/chat \
  -H 'Content-Type: application/json' \
  -d '{"text":"what is my battery level?"}' \
  | python3 -m json.tool

# Full trace — returns all messages (system, user, tool results, assistant)
curl -s http://Ryans-iPhone.local:8080/debug/chat_trace \
  -H 'Content-Type: application/json' \
  -d '{"text":"where am I?"}' \
  | python3 -m json.tool
```

## Key Diagnostic Tools

| Tool | Args | Returns |
|------|------|---------|
| `get_battery` | `{}` | `{"level":95,"state":"charging","low_power_mode":false}` |
| `get_time` | `{}` | `{"local":"Thu, May 28...","timezone_abbr":"PDT",...}` |
| `get_location` | `{}` | `{"latitude":49.27,"longitude":-123.10,"city":"Vancouver",...}` |
| `get_network_info` | `{}` | `{"primary_ip":"192.168.14.198","interfaces":[...]}` |
| `get_device_info` | `{}` | `{"model":"iPhone","system_version":"26.5","storage_free_bytes":...}` |
| `get_screen_info` | `{}` | `{"width_pixels":1125,"height_pixels":2436,"scale":3.0,"orientation":"portrait"}` |
| `get_barometer` | `{}` | `{"pressure_kpa":100.20,"relative_altitude_m":0.0}` |
| `get_compass` | `{}` | `{"magnetic_heading":155.9,"true_heading":171.4,"accuracy_degrees":47.2,"available":true}` |
| `get_audio_route` | `{}` | `{"outputs":[{"name":"Speaker","type":"built_in_speaker"}]}` |
| `get_carrier` | `{}` | `{"carrier":"Rogers","connection_type":"LTE"}` or `"--"` if no SIM |
| `get_locale_info` | `{}` | `{"locale":"en_CA","currency":"CAD","metric":true,"timezone_abbreviation":"PDT"}` |
| `capture_photo` | `{}` | JPEG base64 from front/back camera |
| `speak_text` | `{"text":"..."}` | Plays TTS on speaker |
| `notify_user` | `{"title":"...","body":"..."}` | Local push notification |
| `get_steps` | `{}` | Step count from CMPedometer |
| `get_motion` | `{}` | Accelerometer / gyro snapshot |

## Active Model: Qwen3-1.7B-Q4_K_M.gguf

Default model as of 2026-06-04. Served from Mac HTTP server at initial run
(10.0.3.4:8765), cached in Application Support/llama/ thereafter.
UserDefaults: `llm.gguf.basename = "Qwen3-1.7B-Q4_K_M.gguf"`,
`llm.gguf.url = "http://10.0.3.4:8765/Qwen3-1.7B-Q4_K_M.gguf"`
(or leave both empty to load from bundle copy).

Key fixes applied 2026-06-04:
- **Bundle resource fix**: `forResource: "Qwen3-1.7B-Q4_K_M"` (was hardcoded to 0.6B)
- **Stale-file guard**: files < 100 MB at ggufPath are deleted before load (catches corrupt partial downloads)
- **/no_think mode**: all user-role messages (including tool results) get `/no_think` prefix
  for Qwen3 → disables 200-400s think blocks, reduces turn time to 25-30s on A14
- **96-token cap**: `cappedMaxTokens = min(maxTokens, 96)` — enough for tool call (~20 tok)
  + verbalization (~25 tok), limits worst-case generation to ~32s at 3 tok/s
- **ctx=2048 first in ladder**: `(2048, .q8_0)` is now the first ladder step (was 4096)
  → 115 MB KV vs 231 MB; endSession rebuild ~40s vs ~80s
- **No-rebuild endSession**: `endSession()` calls `resetKVCache()` (KV clear, ~1 ms) not
  `rebuildClient()` (LlamaClient init, 40-200s) — halves 5-battery response time
- **Sampler reset in KV-clear**: `llama_sampler_reset(sampling)` added to `Context.clear()`
  in vendored LocalLLMClient — prevents EOS bias / wrong token generation after KV wipe
- **HTTP handler cancellation**: NWConnection `stateUpdateHandler` now cancels the handler
  Task on disconnect, propagating cancellation to AgentLoop via `withTaskCancellationHandler`
- **defer isRunning=false**: `send()` uses `defer` to guarantee `isRunning=false` even
  when the outer Task is cancelled (curl --max-time disconnects)
- **130s force-reset threshold**: lowered from 180s to match 96-token/ctx-2048 max loop time

Test battery results (2026-06-04 final, Qwen3-1.7B, no_think, 96-tok, ctx=2048):
- Test 1: time → `get_time` → "Thursday June 4, 2026 at 9:25 PM PDT" ✅ (28s)
- Test 2: math (247×83) → `math_eval("247*83")` → 20,501 ✅ (28s)
- Test 3: percent (15% of 340) → `math_eval("0.15*340")` → 51 ✅ (28s)
  (Fixed by adding percent rule to ## When to use in math-and-numbers/SKILL.md)
- Test 4: battery → `get_battery` → 100% fully charged ✅ (28s)
- Test 5: reminder → `add_reminder` → "Done! I've added that reminder" ✅ (24s)
  (Fixed by adding add_reminder example to system prompt to prevent tool_describe)

**Reminder performance (2026-06-04):** 24s with `tool_describe` fix.
Before fix: 4 turns × 27s = 110s (turn 1 = tool_describe request, turn 3 = loop guard).
After fix: 2 turns × 12s = 24s (model uses parameter hints directly, no loop guard).
Fix: system prompt adds explicit `add_reminder` example + "do NOT request schema first".

Build constants to know:
- `LlamaEngine.defaultGGUFBasename = "Qwen3-1.7B-Q4_K_M.gguf"`
- `LlamaEngine.cappedMaxTokens = min(maxTokens, 96)` — tight cap prevents 200s think blocks
- `AgentLoop.maxTokens = 2048` per generate call (capped to 96 by engine)
- Context ladder starts at `(2048, .q8_0)` → 115 MB KV cache (halved from 4096/231 MB)
- `AgentLoop.send(maxTurns: 6)` in debug endpoints — allows error-retry for math queries

## Snapshot Path

When `llm.snapshot.enabled=true` OR a GGUF with "agentbridge" in the filename is
loaded, the agent injects live device state into the system prompt instead of
dispatching tools (for STATE queries only).

**Hybrid routing (2026-05-28):** when a tool-dispatching skill is selected for the
query (e.g., `calendar-and-reminders` for a reminder request), the snapshot path
is bypassed and tool dispatch is used — even if snapshot is enabled. This means:
- State queries (time, battery, location) → snapshot if enabled, or `get_time` etc.
- Action queries (add reminder, create event) → always tool dispatch via skill router

Toggle snapshot via debug API:
```bash
# Enable
curl -s -X POST http://Ryans-iPhone.local:8080/debug/snapshot \
  -H 'Content-Type: application/json' -d '{"enabled": true}'
# Disable
curl -s -X POST http://Ryans-iPhone.local:8080/debug/snapshot \
  -H 'Content-Type: application/json' -d '{"enabled": false}'
# Check
curl -s -X POST http://Ryans-iPhone.local:8080/debug/snapshot
```

Sample snapshot:
```
## Current device state
- battery: 95% (charging, low_power off)
- time: Thursday, May 28, 2026 at 3:10:17 PM PDT
- location: Vancouver BC (49.27, -123.10), accuracy ~10 m
- network: Wi-Fi, IP 192.168.14.198
- device: iPhone, iOS 26.5, 67 GB used / 119 GB total
- locale: en_CA, currency CAD, metric system, PDT time zone
- screen: 1125 × 2436 pixels at 3× scale, portrait
- compass: heading 156° (SSE), accuracy 47°
- audio output: Speaker
- barometer: 100.20 kPa
```

## Datadog LLM Observability (LLMO)

Every agent loop turn is forwarded from the iPhone to the Mac bridge at
`http://192.168.14.105:9090/llmobs/agent_turn` and emitted as a Datadog
LLMObs span hierarchy:

```
agent.turn  (root — full input/output for the conversation turn)
  llm.infer   (LLM inference child — model_name, model_provider, messages)
  tool.<name> (one child per tool call — arguments + result)
```

ML app in Datadog: **`agentbridge`** — view at:
`https://app.datadoghq.com/llm/traces`

### Mac Bridge Setup

The Mac bridge (`bridge/agentbridge_mac.py`) must be running on port **9090**.
Currently started manually; to persist across reboots:

```bash
# Install as launchd service (run once, requires interactive shell)
cp /Users/studio/ios/bridge/local.agentbridge.mac.plist ~/Library/LaunchAgents/
launchctl unload ~/Library/LaunchAgents/local.agentbridge.mac.plist 2>/dev/null
launchctl load -w ~/Library/LaunchAgents/local.agentbridge.mac.plist

# Or start manually (foreground — Ctrl+C to stop):
python3 /Users/studio/ios/bridge/agentbridge_mac.py --port 9090
```

Health check: `curl http://localhost:9090/health` — look for `"llmobs":true`.

### iOS Proxy URL

Set once after app install (survives until factory reset):

```bash
curl -s http://Ryans-iPhone.local:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"user_defaults","arguments":{"action":"set","key":"llmobs_proxy_url","value":"http://192.168.14.105:9090/llmobs/agent_turn"}}}'
```

**Note:** use the Mac's LAN IP (`192.168.14.105`) not mDNS — the phone resolves
`mbp-m1.local` inconsistently when the bridge isn't advertising via Bonjour.

### Datadog Evaluations

Managed evaluations (Hallucination, Topic Relevancy, Tool Selection, etc.) run
automatically once spans reach the `agentbridge` ML app. To create evaluations:

1. Go to **LLM Observability → Evaluations** in Datadog
2. Click **Create Evaluation**
3. Select span-level evaluations — choose `agentbridge` as the Application
4. Use built-in templates: Hallucination, Tool Selection, Tool Argument
   Correctness, Topic Relevancy, Prompt Injection are the most useful for
   AgentBridge

### Dataset Eval → Datadog

To submit offline eval runs (LoRA adapter grading) to Datadog:

```bash
cd /Users/studio/ios
python3 dataset/eval.py --submit-dd
# With a stable run ID (for comparison across adapter versions):
python3 dataset/eval.py --submit-dd --eval-run-id qwen3-adapter-v2
```

Key metrics emitted per row: `dispatch_correct`, `name_correct`, `args_valid`.
Summary span includes: `dispatch_recall`, `tool_name_precision`, `direct_accuracy`.

### Env / Credentials

- `~/.datadog-llmobs.env` — DD_API_KEY, DD_SITE, DD_LLMOBS_ML_APP=agentbridge
- The Mac bridge loads this file automatically at startup
- `~/.zshrc` also has `export DD_API_KEY=...` as the primary source

## Model Config (UserDefaults keys)

| Key | Default | Notes |
|-----|---------|-------|
| `llm.snapshot.enabled` | `""` | Set to `"true"` to enable snapshot path on ANY model |
| `llm.gguf.basename` | `""` | Model filename. Also triggers snapshot path if it contains "agentbridge" |
| `llm.gguf.url` | device models | Custom model URL |
| `llm.context.size` | 4096 | KV cache context length |

**Important:** do NOT set `llm.gguf.basename` to a fake value just to test snapshot path —
the app treats it as a model filename and tries to download it. Use `llm.snapshot.enabled` instead.

**Important:** to CLEAR `llm.gguf.url` and `llm.gguf.basename` overrides, use the `/debug/set_model`
endpoint with an empty body (NOT `user_defaults` set with `value:""`). The `user_defaults` action
stores an empty string, which LlamaEngine treats as "key present → use override" and sets
`ggufPath` to the directory instead of a file → `"Failed to load model from file"` error.

```bash
# Clear model overrides (send NO body — empty JSON {} parses as dict → 400 error)
curl -s -X POST http://Ryans-iPhone.local:8080/debug/set_model
# (omit -d entirely; no Content-Type header; JSON parse fails → removeObject both keys)

# Set a custom model (serves GGUF from Mac over LAN)
curl -s -X POST http://Ryans-iPhone.local:8080/debug/set_model \
  -H 'Content-Type: application/json' \
  -d '{"url":"http://192.168.14.105:8765/ModelName.gguf","basename":"ModelName.gguf"}'
```

### Serving a GGUF from Mac for download

```bash
cd /Users/studio/ios/AgentBridge/Resources/Models  # or dataset/out/
python3 -m http.server 8765
# Verify: curl -I http://192.168.14.105:8765/Qwen3-1.7B-Q4_K_M.gguf
```

Then set the URL override via `/debug/set_model`, reinstall+relaunch the app. The GGUF is saved
to `Application Support/llama/<basename>` and loaded on subsequent launches without the server.

```bash
# Enable snapshot on current model (any model)
curl -s http://Ryans-iPhone.local:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"user_defaults","arguments":{"action":"set","key":"llm.snapshot.enabled","value":"true"}}}'

# Disable snapshot
curl -s http://Ryans-iPhone.local:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"user_defaults","arguments":{"action":"set","key":"llm.snapshot.enabled","value":"false"}}}'

# Check any key
curl -s http://Ryans-iPhone.local:8080/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"user_defaults","arguments":{"action":"get","key":"llm.gguf.basename"}}}'
```

## Build & Deploy

```bash
cd /Users/studio/ios
xcodegen generate
xcodebuild build \
  -project AgentBridge.xcodeproj \
  -scheme AgentBridge \
  -destination 'id=33ED2801-D654-5822-9A72-3A5BD81F5438' \
  -configuration Debug \
  2>&1 | grep -E "BUILD SUCCEEDED|BUILD FAILED|error:" | head -20
# Note: IDESkipMacroFingerprintValidation must be true in Xcode defaults for CLI builds
# defaults write com.apple.dt.Xcode IDESkipMacroFingerprintValidation -bool YES

APP=$(find ~/Library/Developer/Xcode/DerivedData/AgentBridge-*/Build/Products/Debug-iphoneos \
  -name "AgentBridge.app" -maxdepth 1 | head -1)
xcrun devicectl device install app \
  --device 33ED2801-D654-5822-9A72-3A5BD81F5438 "$APP"
xcrun devicectl device process launch \
  --device 33ED2801-D654-5822-9A72-3A5BD81F5438 \
  local.agentbridge.AgentBridge
```

## Device Info

- **Device:** iPhone 12 mini (iPhone13,1), Ryan's iPhone
- **UDID:** `33ED2801-D654-5822-9A72-3A5BD81F5438`
- **Bundle:** `local.agentbridge.AgentBridge`
- **IP:** `192.168.14.198` (en0, Wi-Fi) — mDNS `Ryans-iPhone.local` is more stable
- **iOS:** 26.5
- **Project:** `/Users/studio/ios`

## Gotchas

- WebFetch upgrades HTTP→HTTPS and will fail. Always use `curl` via Bash.
- `/debug/chat` and `/debug/chat_trace` reset message history before each call — stateless.
- The macro `LocalLLMClientMacrosPlugin` requires one-time GUI approval in Xcode
  after any package update; bypass CLI with:
  `defaults write com.apple.dt.Xcode IDESkipMacroFingerprintValidation -bool YES`
- GPS can take up to 3 s indoors; snapshot omits location slot rather than blocking.
- `get_carrier` returns `"--"` (not "null") when eSIM slot is active but
  no carrier is provisioned — treat as nil.
- `get_locale_info` uses key `"timezone_abbreviation"`, not `"timezone_abbr"`.
- Don't send multiple chat_trace requests in parallel — the engine is single-threaded.
  Sequential requests with a small gap are fine. Parallel requests leave the engine stuck
  in "generating" state and subsequent requests return `{"turns":[]}`.
- `LlamaEngine` caps generation at 1024 tokens regardless of `maxTokens` arg.
  The agent loop passes 2048. 1024 is enough for a full Qwen3 think block (~250 tokens)
  + tool call + verbalization. Context window is 4096.
- Think-block extraction: the 1.7B model sometimes places tool call JSON inside the
  `<think>` block rather than outputting it as visible content. `AgentLoop.extractLastToolCallJSON`
  recovers these by scanning the reasoning for the last valid `{"name":..,"arguments":..}` JSON.
- Reminder double-dispatch fix: after a successful tool result, the agent injects
  "The tool completed successfully. Now respond in plain natural language. Do NOT call more tools."
  into the tool result message. Also a loop guard: if the model re-calls the same tool after
  a success result, the loop guard injects a hint and continues instead of dispatching again.
- **SmolLM3-3B is NOT safe on iPhone 12 mini**: 1.9 GB weights + KV cache exceeds the 2 GB jetsam
  ceiling (A14 default, no `increased-memory-limit` entitlement). At ctx=8192 it jetsam-kills;
  at ctx=4096 it's marginal. The fine-tuned SmolLM3-3B-agentbridge-v3-Q4_K_M.gguf
  (at `/Users/studio/ios/dataset/out/`) scores 100%/96.7% on the eval set but can't deploy safely.
  To deploy it would require the `com.apple.developer.kernel.increased-memory-limit` entitlement
  which requires Apple approval. The architecture: 36 layers, 16 heads, 4 KV heads, ctx=65K native.
- **SmolLM2-360M fine-tuned adapter**: The `SmolLM2-360M-agentbridge-v4-Q4_K_M.gguf` (258 MB)
  at `/Users/studio/ios/dataset/out/` is a fine-tuned dispatch model. It fits comfortably in memory
  (258 MB + overhead ≪ 2 GB) but is less capable than Qwen3-1.7B for reasoning. Can be used
  as a low-memory fallback or tested via the debug/set_model endpoint.
- **Qwen3-1.7B is the production model**: 1.0 GB weights, ctx=4096, q8_0 KV cache (~144 MB),
  total ~1.35 GB. Safe on iPhone 12 mini. Stored at Application Support/llama/ after first load.
  Bundle resource: `Models/Qwen3-1.7B-Q4_K_M.gguf` in app bundle.
