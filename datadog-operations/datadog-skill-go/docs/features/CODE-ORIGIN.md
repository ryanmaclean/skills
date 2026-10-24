# Code Origin for APM Spans

Link your APM traces directly to source code for faster debugging and investigation.

---

## What is Code Origin?

**Code Origin** automatically captures the precise locations in your codebase where APM spans originate. Each service entry span records:
- **File path** - Source file generating the span
- **Line number** - Exact line in the file
- **Function name** - Method or function name

This feature helps teams quickly identify and debug performance issues by showing exactly which code generated each trace span.

---

## Benefits

### 1. Faster Debugging
- Jump directly from trace to source code
- No more searching for span origins
- Immediate context for performance issues

### 2. Better Collaboration
- Developers see exact code locations
- Easier handoffs between team members
- Clear ownership of performance bottlenecks

### 3. Improved Incident Response
- Quickly identify problematic code during outages
- Reduce mean time to resolution (MTTR)
- Focus investigation efforts precisely

### 4. Performance Analysis
- Understand which code paths are slow
- Track performance across releases
- Identify optimization opportunities

---

## How It Works

### Architecture

```
Your Application
    ↓
Datadog Tracer (instrumented)
    ↓
Captures: file path, line number, function
    ↓
Sends to Datadog APM
    ↓
View in CLI: dd apm --code-origin
    ↓
Links to GitHub/GitLab source
```

### What Gets Captured

**Service Entry Spans Only**:
- Web request handlers
- gRPC endpoints
- Kafka consumers
- Background job handlers

**Information Recorded**:
```json
{
  "span_id": "12345",
  "service": "web-api",
  "operation": "http.request",
  "code_origin": {
    "file": "src/handlers/api.go",
    "line": 42,
    "function": "HandleUserRequest"
  }
}
```

---

## Prerequisites

### 1. Supported Languages

**Fully Supported**:
- ✅ **Java** (v1.47.0+) - Spring Boot, Spring Data, gRPC, Micronaut 4, Kafka
- ✅ **Python** (v2.15.0+) - Django, Flask, Starlette
- ✅ **Node.js** (v4.49.0+) - Fastify, Express
- ✅ **.NET** (v3.15.0+) - ASP.NET Core
- ✅ **Go** (v1.70.0+) - net/http, gRPC, gorilla/mux
- ✅ **Ruby** (v2.6.0+) - Rails, Sinatra

**Coming Soon**:
- PHP
- Rust

### 2. Source Code Integration

Enable **Source Code Integration** in Datadog:
1. Go to **Organization Settings** → **Source Code Integration**
2. Connect your GitHub, GitLab, or Bitbucket account
3. Enable code preview

This allows Datadog to show code snippets in the UI.

### 3. Tracer Version

Ensure your Datadog tracer is up-to-date:
```bash
# Check versions
dd-trace-java --version     # Java
python -m ddtrace.version   # Python
node -e "console.log(require('dd-trace/package.json').version)"  # Node.js
```

---

## Enabling Code Origin

### Environment Variable (All Languages)

The simplest way to enable Code Origin:

```bash
export DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
```

### Language-Specific Setup

#### Java (Spring Boot)

**application.properties**:
```properties
dd.code.origin.for.spans.enabled=true
```

**Or programmatically**:
```java
import datadog.trace.api.Config;

Config.get().setCodeOriginForSpansEnabled(true);
```

**Run application**:
```bash
java -javaagent:dd-java-agent.jar \
  -Ddd.code.origin.for.spans.enabled=true \
  -jar myapp.jar
```

#### Python (Django/Flask)

**Environment variable**:
```python
import os
os.environ['DD_CODE_ORIGIN_FOR_SPANS_ENABLED'] = 'true'

# Initialize tracer
from ddtrace import tracer
```

**Or in code**:
```python
from ddtrace import config

config.code_origin_for_spans_enabled = True
```

#### Node.js (Express/Fastify)

**Environment variable**:
```javascript
// .env file
DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
```

**For TypeScript with source maps**:
```bash
# Ensure source maps are enabled
node --enable-source-maps -r dd-trace/init app.js
```

**Tracer version requirement**:
```json
{
  "dependencies": {
    "dd-trace": "^5.59.0"
  }
}
```

#### .NET (ASP.NET Core)

**appsettings.json**:
```json
{
  "DD_CODE_ORIGIN_FOR_SPANS_ENABLED": "true"
}
```

**Or environment variable**:
```bash
export DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
dotnet run
```

#### Go

**Environment variable**:
```bash
export DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
```

**In code**:
```go
import "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

tracer.Start(
    tracer.WithCodeOriginForSpans(true),
)
defer tracer.Stop()
```

**Handler example**:
```go
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // This span will capture: file=handlers.go, line=42, func=HandleRequest
    span, ctx := tracer.StartSpanFromContext(r.Context(), "http.request")
    defer span.Finish()

    // Your handler logic...
}
```

#### Ruby (Rails)

**config/initializers/datadog.rb**:
```ruby
Datadog.configure do |c|
  c.code_origin_for_spans_enabled = true
end
```

---

## Using Code Origin with Datadog CLI

### View Traces with Code Origin

```bash
# View recent traces with code origin info
dd apm --from 1h --code-origin

# Filter by service
dd apm --service web-api --code-origin

# View specific trace
dd apm --trace-id abc123 --code-origin
```

### Expected Output

```
Trace ID: abc123def456
Service: web-api
Operation: http.request
Duration: 45ms

Spans:
┌─────────────────────────────────────────────────────────┐
│ http.request                                     45ms   │
│   ├─ File: src/handlers/api.go:42                      │
│   │  Function: HandleUserRequest                       │
│   │  GitHub: https://github.com/org/repo/blob/.../42   │
│   └─ db.query                                    23ms   │
│        ├─ File: src/db/users.go:115                    │
│        └─ Function: FindUserByID                       │
└─────────────────────────────────────────────────────────┘
```

### Query Examples

```bash
# Find slow spans by source file
dd apm --from 6h --query "file:*handlers/api.go duration:>500ms"

# Find all traces from specific function
dd apm --from 1d --query "function:HandleUserRequest"

# Find errors from specific file
dd apm --from 12h --query "file:*auth.go error:true"

# Group by file location
dd apm --from 24h --group-by file
```

---

## Best Practices

### 1. Enable in All Environments

**Development**:
```bash
DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
DD_ENV=development
```

**Staging**:
```bash
DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
DD_ENV=staging
```

**Production**:
```bash
DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
DD_ENV=production
```

### 2. Use with Source Code Integration

Connect your repository to Datadog:
- **GitHub**: Organization Settings → Integrations → GitHub
- **GitLab**: Organization Settings → Integrations → GitLab
- **Bitbucket**: Organization Settings → Integrations → Bitbucket

This enables:
- Code snippets in trace view
- Direct links from CLI to source
- Inline annotations in PRs

### 3. Combine with Git Commit SHA

Track which commit generated the span:

```bash
# Set during deployment
export DD_VERSION=$(git rev-parse HEAD)
export DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
```

Then query:
```bash
dd apm --version abc123def --code-origin
```

### 4. Filter by Code Location

Create monitors for specific files:

```bash
# Alert on slow requests from auth handlers
dd monitors create \
  --query "trace.http.request{file:*auth.go}:p95 > 200ms" \
  --message "Auth handlers are slow"
```

### 5. Use in Incident Investigation

**Incident workflow**:
1. Identify slow trace: `dd apm --from 1h --sort duration`
2. View code origin: `dd apm --trace-id abc123 --code-origin`
3. Jump to GitHub: Click link in output
4. Review code and recent changes
5. Deploy fix with new DD_VERSION

---

## Performance Impact

### Overhead

**CPU**: < 1% overhead
- Capturing file/line is fast (compile-time info)
- No runtime reflection or stack walking

**Memory**: < 5MB per service
- Minimal metadata storage
- Only for entry spans (not all spans)

**Network**: < 1KB per trace
- File path, line number, function name
- Compressed in agent protocol

### When to Disable

**Never disable in production** - overhead is negligible.

Only disable if:
- Running very old tracer versions
- Extreme performance sensitivity (< 1% matters)
- Privacy concerns about code structure

---

## Troubleshooting

### Code Origin Not Showing

**Check tracer version**:
```bash
# Java
java -jar dd-java-agent.jar --version

# Python
python -c "import ddtrace; print(ddtrace.__version__)"

# Node.js
node -e "console.log(require('dd-trace/package').version)"
```

**Ensure env var is set**:
```bash
echo $DD_CODE_ORIGIN_FOR_SPANS_ENABLED
# Should print: true
```

**Check service entry spans**:
- Code Origin only works on **entry spans**
- Internal spans don't have code origin
- Verify you're looking at the top-level span

### Source Maps Not Working (TypeScript)

**Enable source maps**:
```bash
# tsconfig.json
{
  "compilerOptions": {
    "sourceMap": true,
    "inlineSourceMap": false
  }
}
```

**Run with source map support**:
```bash
node --enable-source-maps -r dd-trace/init dist/app.js
```

**Verify source maps exist**:
```bash
ls dist/*.js.map
```

### GitHub Links Not Working

**Check Source Code Integration**:
1. Organization Settings → Source Code Integration
2. Verify repository is connected
3. Check repository permissions

**Verify DD_VERSION matches**:
```bash
# Should match current git commit
echo $DD_VERSION
git rev-parse HEAD
```

### File Paths Are Absolute

**Problem**: Seeing `/home/user/app/src/file.go` instead of `src/file.go`

**Solution**: Set DD_SOURCE_ROOT
```bash
export DD_SOURCE_ROOT=/home/user/app
```

This strips the prefix, showing only `src/file.go`.

---

## Examples

### Example 1: Debug Slow Endpoint

```bash
# 1. Find slow traces
dd apm --from 1h --sort duration | head -5

# Output:
# Trace abc123: 1.2s - POST /api/users
# Trace def456: 890ms - GET /api/orders
# ...

# 2. View code origin
dd apm --trace-id abc123 --code-origin

# Output:
# File: src/handlers/users.go:78
# Function: CreateUser
# GitHub: https://github.com/org/repo/blob/main/src/handlers/users.go#L78

# 3. Open in GitHub, review code
# 4. Identify N+1 query problem
# 5. Fix and deploy
```

### Example 2: Track Performance by File

```bash
# Query traces from specific file
dd apm --from 24h --query "file:*checkout.go" \
  --format json | jq '.[] | {duration, file, line}'

# Output:
# {"duration": 450, "file": "checkout.go", "line": 42}
# {"duration": 320, "file": "checkout.go", "line": 78}
# {"duration": 890, "file": "checkout.go", "line": 42}

# Identify line 42 as consistently slow
# Review that specific function
```

### Example 3: Error Investigation

```bash
# Find errors from authentication module
dd apm --from 6h --query "file:*auth.go error:true"

# View details
dd apm --trace-id xyz789 --code-origin

# Output shows:
# File: auth.go:156
# Function: ValidateToken
# Error: "invalid signature"

# Quick fix: Jump to line 156, review token validation
```

---

## Integration with Other Tools

### CI/CD Pipeline

**Track deployments with code origin**:

```yaml
# .github/workflows/deploy.yml
- name: Deploy with Code Origin
  env:
    DD_VERSION: ${{ github.sha }}
    DD_CODE_ORIGIN_FOR_SPANS_ENABLED: true
  run: |
    kubectl set env deployment/web \
      DD_VERSION=$DD_VERSION \
      DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
```

### Monitoring Dashboards

**Create dashboard widgets**:

```bash
# Widget 1: Top 10 slowest functions
dd dashboards create --widget \
  "Top Slow Functions" \
  --query "trace.*:p95 by {function}"

# Widget 2: Error rate by file
dd dashboards create --widget \
  "Errors by File" \
  --query "trace.*:errors by {file}"
```

### Slack Notifications

**Alert on errors with code links**:

```bash
dd monitors create \
  --query "trace.*.errors{file:*payment.go} > 10" \
  --message "@slack-oncall Payment errors in {{file}}:{{line}} - {{github_link}}"
```

---

## Advanced Usage

### Custom Code Origin Tags

**Add additional context**:

```go
span.SetTag("code.namespace", "handlers.api")
span.SetTag("code.team", "platform")
span.SetTag("code.owner", "alice@example.com")
```

### Filter by Git Branch

```bash
# Track feature branch performance
export DD_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Query
dd apm --from 1h --tag "git.branch:feature/new-checkout"
```

### Combine with Profiling

**Link traces to profiles**:

```bash
# Enable both Code Origin and profiling
export DD_CODE_ORIGIN_FOR_SPANS_ENABLED=true
export DD_PROFILING_ENABLED=true

# View trace with profile link
dd apm --trace-id abc123 --code-origin --with-profile
```

---

## Resources

### Datadog Documentation
- [Code Origin Official Docs](https://docs.datadoghq.com/tracing/code_origin/)
- [Source Code Integration](https://docs.datadoghq.com/integrations/github/)
- [APM Setup Guides](https://docs.datadoghq.com/tracing/setup/)

### Tracer Releases
- [dd-trace-java Releases](https://github.com/DataDog/dd-trace-java/releases)
- [dd-trace-py Releases](https://github.com/DataDog/dd-trace-py/releases)
- [dd-trace-js Releases](https://github.com/DataDog/dd-trace-js/releases)
- [dd-trace-dotnet Releases](https://github.com/DataDog/dd-trace-dotnet/releases)
- [dd-trace-go Releases](https://github.com/DataDog/dd-trace-go/releases)

### Community
- [Datadog Community Forum](https://community.datadoghq.com/)
- [GitHub Discussions](https://github.com/DataDog/datadog-api-client-go/discussions)

---

## FAQ

### Q: Does Code Origin work for all spans?

**A**: No, only **service entry spans** (HTTP requests, gRPC calls, Kafka consumers, etc.). Internal spans don't capture code origin.

### Q: What's the performance impact?

**A**: < 1% CPU, < 5MB memory. Negligible in production.

### Q: Can I use this without Source Code Integration?

**A**: Yes! Code Origin works standalone. Source Code Integration adds GitHub links and code previews.

### Q: Does it work with minified code?

**A**: Yes, but use source maps for readable file names (TypeScript, JavaScript).

### Q: Can I disable it for specific services?

**A**: Yes, set `DD_CODE_ORIGIN_FOR_SPANS_ENABLED=false` for that service.

### Q: Does it capture environment variables or secrets?

**A**: No, only file path, line number, and function name.

---

**Created**: January 22, 2026 (Iteration 15)
**Status**: Production Ready
**Feature**: Code Origin for APM Spans
**Supported Languages**: Java, Python, Node.js, .NET, Go, Ruby
