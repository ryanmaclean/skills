# Testing Quickstart

**Get started testing the Datadog CLI and Skill in under 5 minutes.**

---

## Prerequisites

1. **Set Datadog API credentials:**
   ```bash
   export DD_API_KEY="your_api_key"
   export DD_APP_KEY="your_app_key"
   export DD_SITE="datadoghq.com"  # or datadoghq.eu
   ```

2. **Install jq** (for skill tests):
   ```bash
   # macOS
   brew install jq

   # Linux (Debian/Ubuntu)
   sudo apt-get install jq
   ```

---

## Quick Tests

### Option 1: Full Test Suite (Recommended)

Run all tests (Go CLI + Skill scripts):

```bash
./run-all-tests.sh
```

**What it tests:**
- ✅ Go unit tests (232 tests)
- ✅ CLI build
- ✅ Integration with live Datadog API
- ✅ Skill scripts
- ✅ Code coverage

**Time:** ~2-3 minutes

---

### Option 2: Quick Validation

Fast validation that everything is working:

```bash
./test-skill-quick.sh
```

**What it tests:**
- ✅ Skill directory structure
- ✅ Environment variables
- ✅ Datadog API connectivity
- ✅ Core scripts (APM, logs, metrics)

**Time:** ~30 seconds

---

### Option 3: Go CLI Only

Test just the Go CLI:

```bash
# Unit tests
go test ./... -v

# Build
go build -o dd cmd/main.go

# Try it
./dd health
./dd apm --duration 1h
```

**Time:** ~1 minute

---

### Option 4: Skill Scripts Only

Test skill bash scripts:

```bash
cd ~/dd-skill-test
./test-skills.sh
```

**Time:** ~2 minutes

---

## What Success Looks Like

### Full Test Suite Output

```
╔══════════════════════════════════════════════════════════════╗
║     Datadog CLI & Skill Comprehensive Test Suite            ║
╚══════════════════════════════════════════════════════════════╝

═══════════════════════════════════════════════════════════════
 Go CLI Unit Tests
═══════════════════════════════════════════════════════════════
...
ok  	github.com/datadog/skill/internal/client	3.045s
✓ Go CLI Unit Tests completed successfully

═══════════════════════════════════════════════════════════════
 Go CLI Build Test
═══════════════════════════════════════════════════════════════
  ✓ Binary created: dd
✓ Go CLI Build Test completed successfully

...

═══════════════════════════════════════════════════════════════
 Test Summary
═══════════════════════════════════════════════════════════════

Total test sections: 5
Passed: 5
Failed: 0
Skipped: 0

✅ All tests passed!
```

### Quick Validation Output

```
╔══════════════════════════════════════════════════════════════╗
║        Datadog Skill Quick Validation                        ║
╚══════════════════════════════════════════════════════════════╝

Checking skill directory... ✓
Checking setup.sh exists... ✓
Checking skill definition... ✓
Checking DD_API_KEY... ✓
Checking DD_APP_KEY... ✓
Checking jq installed... ✓ (jq-1.7.1)
Validating Datadog API... ✓
Running verify-setup.sh... ✓
Testing query-apm.sh... ✓
Testing search-logs.sh... ✓
Testing query-metrics.sh... ✓

═══════════════════════════════════════════════════════════════
Results: 11/11 passed

✅ Skill is working correctly!
```

---

## Troubleshooting

### "DD_API_KEY not set"

```bash
# Add to ~/.zshrc or ~/.bashrc
export DD_API_KEY="your_api_key"
export DD_APP_KEY="your_app_key"

# Reload
source ~/.zshrc
```

### "jq: command not found"

```bash
brew install jq  # macOS
sudo apt-get install jq  # Linux
```

### "go: command not found"

```bash
# macOS
brew install go

# Linux
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Tests fail with "403 Forbidden"

Your API key needs additional permissions. In Datadog:
1. Go to Organization Settings → Application Keys
2. Click your app key
3. Enable "Actions API Access"

---

## Next Steps

1. **Test in Claude Code:**
   ```bash
   claude-code ~/dd-skill-test
   ```
   Then try: `/datadog` or "Check APM for my-service"

2. **View detailed test results:**
   ```bash
   go tool cover -html=coverage.out
   ```

3. **Read full testing guide:**
   See [SKILL-TESTING.md](SKILL-TESTING.md)

---

## Questions?

- **Detailed testing:** See [SKILL-TESTING.md](SKILL-TESTING.md)
- **Known issues:** See [KNOWN-ISSUES.md](KNOWN-ISSUES.md)
- **Contributing:** See [CONTRIBUTING.md](CONTRIBUTING.md)
