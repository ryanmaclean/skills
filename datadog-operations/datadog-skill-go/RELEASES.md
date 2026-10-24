# Binary Distribution via GitHub Releases

This document explains how to distribute the Go CLI binaries via GitHub Releases instead of committing them to the repository.

## Why GitHub Releases?

- **Smaller Repository:** Reduces repo size from ~150 MB to ~20 MB
- **Faster Clones:** 7x faster for contributors
- **Standard Practice:** Industry standard for binary distribution
- **Bandwidth Efficient:** Only download binaries you need
- **Release Notes:** Attach release notes and changelogs

## Current Binary Sizes

```
datadog-cli-darwin-amd64:       12 MB (macOS Intel)
datadog-cli-darwin-arm64:       11 MB (macOS Apple Silicon)
datadog-cli-linux-amd64:        12 MB (Linux x64)
datadog-cli-linux-arm64:        11 MB (Linux ARM)
datadog-cli-windows-amd64.exe:  12 MB (Windows x64)
datadog-cli-windows-arm64.exe:  11 MB (Windows ARM)
Total:                          69 MB
```

## How to Create a Release

### Step 1: Build Binaries

```bash
cd dd-skill-test-go

# Build all platform binaries
make build-all

# Or build individually
GOOS=darwin GOARCH=amd64 go build -o bin/datadog-cli-darwin-amd64 cmd/main.go
GOOS=darwin GOARCH=arm64 go build -o bin/datadog-cli-darwin-arm64 cmd/main.go
GOOS=linux GOARCH=amd64 go build -o bin/datadog-cli-linux-amd64 cmd/main.go
GOOS=linux GOARCH=arm64 go build -o bin/datadog-cli-linux-arm64 cmd/main.go
GOOS=windows GOARCH=amd64 go build -o bin/datadog-cli-windows-amd64.exe cmd/main.go
GOOS=windows GOARCH=arm64 go build -o bin/datadog-cli-windows-arm64.exe cmd/main.go
```

### Step 2: Create Git Tag

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0: Complete Datadog automation toolkit"

# Push tag to GitHub
git push origin v1.0.0
```

### Step 3: Create GitHub Release

**Via GitHub UI:**

1. Go to https://github.com/ryanmaclean/dd-skill-test/releases/new
2. Choose tag: `v1.0.0`
3. Title: `v1.0.0 - Complete Datadog Automation Toolkit`
4. Description: Copy from CHANGELOG.md
5. Attach binaries:
   - `datadog-cli-darwin-amd64`
   - `datadog-cli-darwin-arm64`
   - `datadog-cli-linux-amd64`
   - `datadog-cli-linux-arm64`
   - `datadog-cli-windows-amd64.exe`
   - `datadog-cli-windows-arm64.exe`
6. Click "Publish release"

**Via GitHub CLI:**

```bash
# Create release
gh release create v1.0.0 \
  --title "v1.0.0 - Complete Datadog Automation Toolkit" \
  --notes-file CHANGELOG.md \
  bin/datadog-cli-darwin-amd64 \
  bin/datadog-cli-darwin-arm64 \
  bin/datadog-cli-linux-amd64 \
  bin/datadog-cli-linux-arm64 \
  bin/datadog-cli-windows-amd64.exe \
  bin/datadog-cli-windows-arm64.exe
```

### Step 4: Update Documentation

Update installation instructions in README.md:

```markdown
## Installation

### Download Pre-built Binary

Download the latest release for your platform:

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/ryanmaclean/dd-skill-test/releases/latest/download/datadog-cli-darwin-arm64 -o /usr/local/bin/datadog-cli
chmod +x /usr/local/bin/datadog-cli
```

**macOS (Intel):**
```bash
curl -L https://github.com/ryanmaclean/dd-skill-test/releases/latest/download/datadog-cli-darwin-amd64 -o /usr/local/bin/datadog-cli
chmod +x /usr/local/bin/datadog-cli
```

**Linux (x64):**
```bash
curl -L https://github.com/ryanmaclean/dd-skill-test/releases/latest/download/datadog-cli-linux-amd64 -o /usr/local/bin/datadog-cli
chmod +x /usr/local/bin/datadog-cli
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://github.com/ryanmaclean/dd-skill-test/releases/latest/download/datadog-cli-windows-amd64.exe" -OutFile "datadog-cli.exe"
```

Or download directly from: https://github.com/ryanmaclean/dd-skill-test/releases/latest
```

## Automated Release with GitHub Actions

See `.github/workflows/release.yml` for automated binary building and release creation on tag push.

## Benefits

1. **Repository Size:** ~70 MB → ~20 MB (77% reduction)
2. **Clone Speed:** 150 MB → 20 MB (7x faster)
3. **Selective Download:** Users only download binaries for their platform
4. **Version Control:** Easy to track which binaries belong to which version
5. **Release Notes:** Attach comprehensive release notes
6. **Checksums:** Generate SHA256 checksums for verification

## Removing Binaries from Git History (Optional)

To completely remove binaries from git history (requires force-push):

```bash
# Using BFG Repo-Cleaner (recommended)
bfg --delete-files 'datadog-cli-*'
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# Or using git filter-branch
git filter-branch --force --index-filter \
  "git rm --cached --ignore-unmatch dd-skill-test-go/bin/datadog-cli-*" \
  --prune-empty --tag-name-filter cat -- --all
```

**Warning:** Force-push required. Coordinate with all contributors.

## FAQ

**Q: Can users still build from source?**
A: Yes! Full build instructions in dd-skill-test-go/README.md

**Q: How do we update binaries?**
A: Create a new tag and release. GitHub keeps all versions available.

**Q: What about auto-updates?**
A: Consider implementing self-update feature in Go CLI to check for new releases.

**Q: How do we verify binary integrity?**
A: Generate and publish SHA256 checksums with each release.

## See Also

- [GitHub Releases Documentation](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
