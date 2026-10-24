# Datadog Claude Code Plugin - Troubleshooting Guide

Common issues and solutions when using the Datadog Claude Code plugin.

## Table of Contents

1. [Authentication Issues](#authentication-issues)
2. [Service Detection Issues](#service-detection-issues)
3. [No Data Returned](#no-data-returned)
4. [CLI Installation Issues](#cli-installation-issues)
5. [Plugin Not Working](#plugin-not-working)
6. [Performance Issues](#performance-issues)
7. [API Rate Limits](#api-rate-limits)
8. [Credential Security](#credential-security)

---

## Authentication Issues

### Problem: "Authentication failed" or "Invalid API key"

**Cause:** Missing or incorrect Datadog credentials.

**Solution 1:** Check credentials are exported

```bash
echo $DD_API_KEY
echo $DD_APP_KEY
echo $DD_SITE
```

If any are empty, export them:

```bash
export DD_API_KEY="your-api-key"
export DD_APP_KEY="your-app-key"
export DD_SITE="datadoghq.com"
```

**Solution 2:** Verify credentials are correct

1. Log in to Datadog: https://app.datadoghq.com
2. Go to Organization Settings → API Keys
3. Verify your API key is correct and active
4. Go to Organization Settings → Application Keys
5. Verify your Application key is correct and active

**Solution 3:** Check Datadog site

Different Datadog sites use different domains:
- US1: `datadoghq.com`
- US3: `us3.datadoghq.com`
- US5: `us5.datadoghq.com`
- EU: `datadoghq.eu`
- GOV: `ddog-gov.com`

Make sure `DD_SITE` matches your account's region.

**Solution 4:** Make credentials permanent

Add to shell profile (~/.zshrc or ~/.bashrc):

```bash
echo 'export DD_API_KEY="your-key"' >> ~/.zshrc
echo 'export DD_APP_KEY="your-key"' >> ~/.zshrc
echo 'export DD_SITE="datadoghq.com"' >> ~/.zshrc
source ~/.zshrc
```

### Problem: "Unauthorized" or "Permission denied"

**Cause:** Application key lacks necessary permissions.

**Solution:**

1. Go to Organization Settings → Application Keys
2. Delete the current Application key
3. Create a new one with **full permissions**
4. Update `DD_APP_KEY` with new key

---

## Service Detection Issues

### Problem: "Service not found"

**Cause:** Service name is incorrect or doesn't exist.

**Solution 1:** Check exact service name (case-sensitive)

```bash
# List all services in catalog
dd catalog

# Or ask Claude:
"List all services in the Datadog catalog"
```

Service names are **case-sensitive**:
- ✅ `api-service`
- ❌ `API-Service`
- ❌ `Api-Service`

**Solution 2:** Use full service name

Some services have prefixes or suffixes:
- `prod-api-service`
- `api-service-v2`
- `my-company-api-service`

Check in Datadog UI: APM → Services

**Solution 3:** Specify service explicitly

If auto-detection fails:

```
"Check health for api-service"          # Instead of just "Check health"
"Show logs for payment-service"         # Instead of just "Show logs"
```

Or use CLI directly:

```bash
dd health --service api-service
dd logs --service payment-service
```

### Problem: Context detection not working

**Cause:** No git repository or service mapping not configured.

**Solution 1:** Check context detection

```bash
dd context
```

**Solution 2:** Set context manually

```bash
dd context --set api-service
```

**Solution 3:** Use DD_SERVICE environment variable

```bash
export DD_SERVICE="api-service"
```

**Solution 4:** Always specify service

In queries to Claude, include the service name:

```
"Check health for api-service"
"Show logs for payment-service"
```

---

## No Data Returned

### Problem: "No data found" or empty results

**Cause:** Service has no data in the specified time range.

**Solution 1:** Try longer time range

```
"Show traces from the last 24 hours"    # instead of "last hour"
"Show logs from the last week"          # instead of default 15 minutes
```

CLI flags:
```bash
dd apm api-service --from 24h
dd logs --from 7d
dd metrics "..." --from 30d
```

**Solution 2:** Verify service is sending data

Check in Datadog web UI:
1. APM → Services → [your-service]
2. Verify traces are appearing
3. Note the time of last activity

**Solution 3:** Check service instrumentation

Ensure your service is properly instrumented:
- APM tracer is installed and configured
- Logs are being forwarded to Datadog
- Metrics are being collected

**Solution 4:** Verify environment/tags

Data might be filtered by environment:

```bash
# Check all environments
dd apm api-service  # shows all envs

# Or specify environment
dd apm api-service --env production
```

### Problem: Partial data only

**Cause:** API result limits or sampling.

**Solution 1:** Increase limit

```bash
dd logs --limit 500              # default is 100
dd apm --limit 1000              # get more traces
```

**Solution 2:** Use narrower time range

Shorter time ranges return more complete data:

```bash
dd logs --from 1h    # More complete than --from 7d
```

**Solution 3:** Apply specific filters

Instead of querying all data, filter:

```bash
dd logs --service api-service --status error
dd apm api-service --status error --resource "POST /api/orders"
```

---

## CLI Installation Issues

### Problem: "Command not found: dd"

**Cause:** CLI not installed or not in PATH.

**Solution 1:** Check if CLI is installed

```bash
which dd
```

If not found, install the Datadog CLI.

**Solution 2:** Use full path

If installed but not in PATH:

```bash
/full/path/to/dd health
```

**Solution 3:** Add to PATH

Find where CLI is installed:
```bash
find / -name "dd" -type f 2>/dev/null
```

Add to PATH:
```bash
export PATH="/path/to/cli:$PATH"
```

Make permanent by adding to ~/.zshrc or ~/.bashrc.

### Problem: "Permission denied" when running dd

**Cause:** CLI binary doesn't have execute permissions.

**Solution:**

```bash
chmod +x /path/to/dd
```

### Problem: CLI installed but commands fail

**Cause:** Wrong binary or old version.

**Solution 1:** Check version

```bash
dd version
# Should show version 1.0.0 or later
```

**Solution 2:** Reinstall CLI

Download latest version and replace old binary.

---

## Plugin Not Working

### Problem: Claude doesn't recognize Datadog commands

**Cause:** Plugin not installed or not loaded.

**Solution 1:** Verify plugin is installed

```bash
ls -la ~/.claude/plugins/user/datadog-cli/
```

Should show:
```
.claude-plugin/plugin.json
commands/ (22 .md files)
```

**Solution 2:** Reinstall plugin

```bash
cp -r /path/to/claude-plugin ~/.claude/plugins/user/datadog-cli/
```

**Solution 3:** Restart Claude Code

Exit and restart Claude Code to reload plugins.

**Solution 4:** Check plugin.json

Verify `~/.claude/plugins/user/datadog-cli/.claude-plugin/plugin.json` exists and is valid JSON.

### Problem: Skills not triggering

**Cause:** Query doesn't match skill description.

**Solution:** Use more specific language

Instead of vague queries:
❌ "Check stuff"
❌ "Show me things"

Use specific terms:
✅ "Check health"
✅ "Show logs"
✅ "List incidents"
✅ "Check deploy safety"

### Problem: Wrong skill triggered

**Cause:** Ambiguous query.

**Solution:** Be more specific

Instead of:
❌ "Show me data" → Could trigger any skill

Use:
✅ "Show me APM traces"
✅ "Show me error logs"
✅ "Show me metrics for CPU usage"

---

## Performance Issues

### Problem: Commands are slow

**Cause:** Network latency or API response time.

**Solution 1:** Check network connectivity

```bash
ping app.datadoghq.com
```

**Solution 2:** Use shorter time ranges

Querying less data is faster:

```bash
dd logs --from 1h     # Fast
dd logs --from 30d    # Slower
```

**Solution 3:** Add filters

Filtered queries are faster:

```bash
dd logs --service api-service --status error  # Fast
dd logs                                        # Slower (all logs)
```

**Solution 4:** Check Datadog API status

Visit: https://status.datadoghq.com

### Problem: Timeout errors

**Cause:** Query too large or API too slow.

**Solution 1:** Reduce time range

```bash
dd apm api-service --from 15m  # Instead of --from 7d
```

**Solution 2:** Add more filters

```bash
dd logs --service api-service --status error --from 1h
# Instead of: dd logs --from 1d
```

**Solution 3:** Increase timeout (if CLI supports it)

```bash
dd logs --timeout 60s
```

---

## API Rate Limits

### Problem: "Rate limit exceeded" or "429 Too Many Requests"

**Cause:** Too many API calls in short time period.

**Solution 1:** Wait before retrying

Datadog API has rate limits. Wait 1-5 minutes between bulk queries.

**Solution 2:** Reduce query frequency

Don't run queries in tight loops:

```bash
# ❌ Bad: Too many requests
while true; do dd health; sleep 1; done

# ✅ Good: Reasonable frequency
while true; do dd health; sleep 60; done
```

**Solution 3:** Use longer time ranges

Instead of multiple short queries, use one longer query:

```bash
# ❌ Bad: Multiple queries
dd logs --from 1h
dd logs --from 2h
dd logs --from 3h

# ✅ Good: One query
dd logs --from 3h
```

**Solution 4:** Cache results

Store query results locally instead of re-querying:

```bash
dd health > health_status.json
# Use cached results for a few minutes
```

### Problem: "Quota exceeded"

**Cause:** Datadog account API quota limits reached.

**Solution:**

1. Check your Datadog plan's API limits
2. Contact Datadog support to increase quota
3. Reduce API call frequency
4. Use more specific queries to reduce data volume

---

## Credential Security

### Problem: Worried about credential security

**Best Practices:**

1. **Never commit credentials to git:**

Add to `.gitignore`:
```
.env
.credentials
*.key
```

2. **Use environment variables (not files):**

```bash
# ✅ Good: Environment variables
export DD_API_KEY="..."

# ❌ Bad: Plain text files
echo "DD_API_KEY=..." > credentials.txt
```

3. **Use credential managers:**

macOS Keychain:
```bash
security add-generic-password -a $(whoami) -s datadog-api-key -w
security find-generic-password -a $(whoami) -s datadog-api-key -w
```

4. **Rotate keys regularly:**

Create new API/Application keys monthly and delete old ones.

5. **Use restricted keys:**

Create Application keys with minimal required permissions, not full access.

6. **Never share credentials:**

Each team member should have their own API/Application keys.

---

## Getting Help

### Check Documentation

1. **Plugin docs:** `PLUGIN-COMPLETE.md`, `SKILLS-STATUS.md`
2. **Skill docs:** `~/.claude/plugins/user/datadog-cli/commands/*.md`
3. **Testing guide:** `TESTING-GUIDE.md`
4. **Quickstart:** `CLAUDE-CODE-QUICKSTART.md`

### Command Help

```bash
dd --help
dd health --help
dd apm --help
```

### Ask Claude

In Claude Code:

```
"How do I check service health?"
"What Datadog commands are available?"
"Show me examples of log queries"
"Help me troubleshoot authentication errors"
```

### Check Logs

Enable verbose logging:

```bash
dd health --verbose
dd logs --debug
```

### Verify Setup

Run diagnostic checks:

```bash
# Check credentials
echo $DD_API_KEY
echo $DD_APP_KEY
echo $DD_SITE

# Check CLI
which dd
dd version

# Check context
dd context

# Test API connectivity
dd catalog
```

---

## Common Error Messages

### "Error: context deadline exceeded"

**Meaning:** Request timed out.

**Solution:** Use shorter time range or add more filters.

### "Error: service not instrumented"

**Meaning:** Service exists but has no APM instrumentation.

**Solution:** Install and configure Datadog APM tracer in your service.

### "Error: no metrics found"

**Meaning:** Metric name doesn't exist or has no data.

**Solution:** Check metric name in Datadog Metrics Explorer.

### "Error: invalid query syntax"

**Meaning:** Log query or metric query has syntax error.

**Solution:** Check Datadog query syntax documentation.

### "Error: resource not found"

**Meaning:** Trying to access incident/monitor/dashboard that doesn't exist.

**Solution:** Verify resource ID is correct.

---

## Still Having Issues?

### Create Detailed Bug Report

Include:

1. **Error message:** Exact error text
2. **Command:** Full command executed
3. **Environment:**
   - OS version
   - CLI version (`dd version`)
   - Datadog site (`echo $DD_SITE`)
4. **Expected behavior:** What should happen
5. **Actual behavior:** What actually happened
6. **Verbose output:** Run with `--verbose` or `--debug`

### Test with Simple Query

Isolate the issue:

```bash
# Start with simplest query
dd context

# Then try basic commands
dd catalog
dd health

# Then try your failing command
dd [your-command]
```

### Check Similar Issues

Search existing issues or discussions for similar problems.

---

**Most Common Fixes:**

1. ✅ Check `DD_API_KEY` and `DD_APP_KEY` are exported
2. ✅ Verify service names are case-sensitive and correct
3. ✅ Use longer time ranges if no data returned
4. ✅ Restart Claude Code after installing plugin
5. ✅ Check `dd version` shows 1.0.0+
