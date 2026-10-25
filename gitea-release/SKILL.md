---
name: gitea-release
description: Create Gitea releases and upload build artifacts via the tea CLI and Gitea API. Use when publishing binaries or packages to a Gitea instance after a version bump.
trigger: gitea release, release artifact, upload binary, tea release, gitea api release
---

# Gitea Release Artifact Upload

As of 2026-04-06. See also `tea` for repo/issue/PR workflows.
See deployment-specific details below for fleet Gitea host coordinates.

## Create a release

```bash
# via tea CLI
tea release create \
  --repo studio/ii-agent \
  --tag v1.0.0 \
  --title "ii-agent v1.0.0" \
  --note "100 tests, production-ready"

# via raw API (fallback)
tea api -X POST /repos/studio/ii-agent/releases \
  -d '{"tag_name":"v1.0.0","name":"ii-agent v1.0.0","body":"100 tests, production-ready"}'
```

## Upload a single asset

```bash
tea release asset create \
  --repo studio/ii-agent \
  --tag v1.0.0 \
  --file ./packages/builds-1.0.0/aarch64-linux-musl/ii-agent
```

## Upload all targets after a build (nushell)

```nu
let targets = [
  "aarch64-linux-musl"
  "aarch64-linux-gnu"
  "aarch64-netbsd"
  "x86_64-linux-gnu"
  "aarch64-macos"
]

for t in $targets {
  let src = $"packages/builds-1.0.0/($t)/ii-agent"
  if ($src | path exists) {
    tea release asset create --repo studio/ii-agent --tag v1.0.0 --file $src
  }
}
```

## List releases

```bash
tea release list --repo studio/ii-agent
```

## Delete a release (careful)

```bash
tea api -X DELETE /repos/studio/ii-agent/releases/<ID>
```

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Tag doesn't exist yet | Push tag first: `git tag v1.0.0 && git push origin v1.0.0` |
| Asset name collision | Delete old asset before re-upload or use a suffix |
| Uploading from Mac to macvlan Gitea | Works from host network; if blocked, hop through the bastion host (see deployment-specific details below) |
| Embedding token in URL | Use `tea` auth or `~/.netrc` — never in git remotes (see memory) |
| Over-600MB single asset | Gitea default limit; split or raise `MAX_UPLOAD_SIZE` in Gitea config |

## Related

- `tea` — full Gitea CLI reference (repos, issues, PRs, code browsing)
- `ii-agent-ops` — upstream build step that produces the artifacts (project-scoped)
- `macvlan-net` — macvlan networking topology (relevant if Gitea runs on a macvlan interface)

## Verification contract (REQUIRED for all releases)

Every `tea release create` MUST end with a `<claims>` block:

```
<claims>
- claim: "Release TAG exists on REPO"
  probe: "tea release view --repo studio/REPO --tag TAG"
  expect: "TAG"
- claim: "Asset FILE uploaded"
  probe: "tea release view --repo studio/REPO --tag TAG | grep -c 'FILE'"
  expect: "1"
</claims>
```

Verdicts via `fleet-eval.nu verify`. See `fleet-eval` skill.

---

## Deployment-specific details

This skill is generic Gitea release tooling. Everything above applies to any
Gitea instance. Concrete host coordinates live in the deployment's fleet-ops repo.

**For the rust-town fleet:**
- `fleet-ops/docs/gitea-ci.md` — Gitea instance URLs, runner host table, CI secret conventions, fork-repo locations
- Gitea host and bastion hop IPs are in that doc; do not embed them here
