# CI/CD Architecture

Visual overview of the GitHub Actions CI/CD pipeline for Datadog CLI.

## Pipeline Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         GitHub Events (Triggers)                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
                    ▼               ▼               ▼
              ┌─────────┐     ┌─────────┐    ┌──────────┐
              │  Push   │     │   PR    │    │   Tag    │
              │  Event  │     │  Event  │    │  Event   │
              └─────────┘     └─────────┘    └──────────┘
                    │               │               │
                    └───────┬───────┴───────┬───────┘
                            │               │
        ┌───────────────────┼───────────────┼────────────────────┐
        │                   │               │                    │
        ▼                   ▼               ▼                    ▼
┌──────────────┐    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│      CI      │    │    Build     │  │   Coverage   │  │   Docker     │
│   Workflow   │    │   Workflow   │  │   Workflow   │  │   Workflow   │
└──────────────┘    └──────────────┘  └──────────────┘  └──────────────┘
        │                   │               │                    │
        │                   │               │                    │
        ▼                   ▼               ▼                    ▼
┌──────────────┐    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Validate   │    │   Release    │  │              │  │              │
│   Workflow   │    │   Workflow   │  │  (on tag)    │  │  (on tag)    │
└──────────────┘    └──────────────┘  └──────────────┘  └──────────────┘
```

## CI Workflow (ci.yml)

```
┌────────────────────────────────────────────────────────────┐
│                    CI Workflow Trigger                      │
│              Push to main/develop or PR                     │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│    Test    │     │    Lint    │     │   Build    │
│   Matrix   │     │    Job     │     │    Job     │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  9 Jobs    │     │ golangci-  │     │  go build  │
│  3 OS x    │     │    lint    │     │  + verify  │
│  3 Go ver  │     │            │     │            │
└────────────┘     └────────────┘     └────────────┘
        │
        ▼
┌────────────┐
│  Security  │
│  Scanning  │
│  (Gosec)   │
└────────────┘
```

## Build Workflow (build.yml)

```
┌────────────────────────────────────────────────────────────┐
│                 Build Workflow Trigger                      │
│           Push, PR, or Manual Dispatch                      │
└────────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │        Build Matrix (6 Jobs)          │
        │                                       │
        │  ┌────────────┬────────────┐         │
        │  │   linux    │   linux    │         │
        │  │   amd64    │   arm64    │         │
        │  ├────────────┼────────────┤         │
        │  │  darwin    │  darwin    │         │
        │  │   amd64    │   arm64    │         │
        │  ├────────────┼────────────┤         │
        │  │  windows   │  windows   │         │
        │  │   amd64    │   arm64    │         │
        │  └────────────┴────────────┘         │
        └──────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │   For Each Platform:                  │
        │   1. Extract version info             │
        │   2. Build with ldflags               │
        │   3. Create archive                   │
        │   4. Generate checksum                │
        │   5. Upload artifact                  │
        └──────────────────────────────────────┘
```

## Release Workflow (release.yml)

```
┌────────────────────────────────────────────────────────────┐
│                Release Workflow Trigger                     │
│                 Version Tag (v*.*.*)                        │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┴──────────────────┐
        │                                     │
        ▼                                     ▼
┌────────────────┐                   ┌────────────────┐
│  Build Job     │                   │  Release Job   │
│  (6 platforms) │                   │  (depends on   │
│                │                   │   build)       │
└────────────────┘                   └────────────────┘
        │                                     │
        ▼                                     ▼
┌────────────────┐                   ┌────────────────┐
│  Upload        │                   │  Download      │
│  Artifacts     │──────────────────▶│  Artifacts     │
└────────────────┘                   └────────────────┘
                                              │
                                              ▼
                                     ┌────────────────┐
                                     │  Generate      │
                                     │  Changelog     │
                                     └────────────────┘
                                              │
                                              ▼
                                     ┌────────────────┐
                                     │  Create        │
                                     │  Release Notes │
                                     └────────────────┘
                                              │
                                              ▼
                                     ┌────────────────┐
                                     │  Create GitHub │
                                     │  Release       │
                                     └────────────────┘
```

## Coverage Workflow (coverage.yml)

```
┌────────────────────────────────────────────────────────────┐
│               Coverage Workflow Trigger                     │
│              Push to main/develop or PR                     │
└────────────────────────────────────────────────────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │  Run Tests    │
                   │  with Coverage│
                   └───────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │  Calculate    │
                   │  Percentage   │
                   └───────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Check     │     │  Upload to │     │  Generate  │
│  Threshold │     │  Codecov   │     │  HTML      │
│  (70%)     │     │            │     │  Report    │
└────────────┘     └────────────┘     └────────────┘
        │                                     │
        │                                     ▼
        │                              ┌────────────┐
        │                              │  Upload    │
        │                              │  Artifact  │
        │                              └────────────┘
        │
        ▼
┌────────────┐
│  Comment   │
│  on PR     │
│  (if PR)   │
└────────────┘
```

## Docker Workflow (docker.yml)

```
┌────────────────────────────────────────────────────────────┐
│                Docker Workflow Trigger                      │
│          Push, PR, Tag, or Manual Dispatch                  │
└────────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │  Setup                                │
        │  - QEMU (multi-arch)                  │
        │  - Docker Buildx                      │
        │  - Login to ghcr.io                   │
        └──────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │  Extract Metadata                     │
        │  - Generate tags                      │
        │  - Generate labels                    │
        └──────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │  Build Multi-arch Image               │
        │  - linux/amd64                        │
        │  - linux/arm64                        │
        │  - Push to registry                   │
        └──────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │  Security Scan                        │
        │  - Run Trivy                          │
        │  - Upload SARIF                       │
        └──────────────────────────────────────┘
```

## Workflow Dependencies

```
PR Merge Requirements:
┌─────────────────────────────────────────┐
│  ✓ CI Workflow (all jobs pass)          │
│  ✓ Validate Workflow (all jobs pass)    │
│  ✓ Coverage Workflow (>= 70%)           │
│  ✓ Code Review Approval                 │
└─────────────────────────────────────────┘

Release Flow:
┌─────────────────────────────────────────┐
│  1. Developer creates tag (v1.2.3)      │
│  2. Release workflow triggers           │
│  3. Build all 6 platforms               │
│  4. Create GitHub Release               │
│  5. Docker workflow triggers            │
│  6. Build and push images               │
└─────────────────────────────────────────┘
```

## Security Architecture

```
┌────────────────────────────────────────────────────────────┐
│                     Security Layers                         │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Input     │     │  Code      │     │  Image     │
│  Validation│     │  Scanning  │     │  Scanning  │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  No        │     │  Gosec     │     │  Trivy     │
│  Command   │     │  SARIF     │     │  SARIF     │
│  Injection │     │  Upload    │     │  Upload    │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        └──────────────────┴──────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────────┐
        │  GitHub Security Tab                  │
        │  - Code scanning alerts               │
        │  - Dependabot alerts                  │
        │  - Secret scanning                    │
        └──────────────────────────────────────┘
```

## Artifact Flow

```
┌────────────────────────────────────────────────────────────┐
│                      Artifact Types                         │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Build     │     │  Coverage  │     │  Docker    │
│  Artifacts │     │  Reports   │     │  Images    │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  7-day     │     │  30-day    │     │  Permanent │
│  retention │     │  retention │     │  (registry)│
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  6 platform│     │  HTML      │     │  Multi-    │
│  binaries  │     │  report    │     │  arch      │
│  + SHA256  │     │            │     │  images    │
└────────────┘     └────────────┘     └────────────┘
```

## Caching Strategy

```
┌────────────────────────────────────────────────────────────┐
│                      Cache Layers                           │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Go        │     │  Docker    │     │  Action    │
│  Modules   │     │  Layers    │     │  Cache     │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  setup-go  │     │  buildx    │     │  go.sum    │
│  cache:    │     │  cache-to: │     │  based     │
│  true      │     │  gha       │     │  key       │
└────────────┘     └────────────┘     └────────────┘
```

## Notification Flow

```
┌────────────────────────────────────────────────────────────┐
│                   Workflow Events                           │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Success   │     │  Failure   │     │  Comment   │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  GitHub    │     │  GitHub    │     │  PR        │
│  Check     │     │  Check     │     │  Comment   │
│  (green)   │     │  (red)     │     │  (coverage)│
└────────────┘     └────────────┘     └────────────┘
```

## Workflow Matrix

| Workflow | Platforms | Go Versions | Parallel Jobs | Avg Time |
|----------|-----------|-------------|---------------|----------|
| CI       | 3 OS      | 3 versions  | 9             | 10 min   |
| Build    | 6 combo   | 1 (latest)  | 6             | 4 min    |
| Coverage | 1 (ubuntu)| 1 (latest)  | 1             | 3 min    |
| Docker   | 2 arch    | 1 (latest)  | 1             | 6 min    |
| Release  | 6 combo   | 1 (latest)  | 6 + 1         | 5 min    |
| Validate | 5 checks  | 1 (latest)  | 5             | 2 min    |

## Environment Variables

```
Build-time Variables (via ldflags):
┌────────────────────────────────────┐
│  version   ← git tag or describe   │
│  commit    ← git commit SHA        │
│  buildDate ← ISO 8601 timestamp    │
└────────────────────────────────────┘

Runtime Variables:
┌────────────────────────────────────┐
│  GITHUB_TOKEN    (automatic)       │
│  CODECOV_TOKEN   (optional)        │
└────────────────────────────────────┘
```

## Integration Points

```
┌────────────────────────────────────────────────────────────┐
│                   External Services                         │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Codecov   │     │  GitHub    │     │  GitHub    │
│  (coverage)│     │  Packages  │     │  Security  │
│            │     │  (ghcr.io) │     │  (SARIF)   │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Badge     │     │  Docker    │     │  Alerts    │
│  Update    │     │  Images    │     │  Dashboard │
└────────────┘     └────────────┘     └────────────┘
```

## Quality Gates

```
┌────────────────────────────────────────────────────────────┐
│              Pull Request Quality Gates                     │
└────────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
┌────────────┐     ┌────────────┐     ┌────────────┐
│  Tests     │     │  Coverage  │     │  Linting   │
│  Pass      │     │  >= 70%    │     │  No Issues │
└────────────┘     └────────────┘     └────────────┘
        │                  │                  │
        └──────────────────┴──────────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │  Code Review  │
                   │  Approval     │
                   └───────────────┘
                           │
                           ▼
                   ┌───────────────┐
                   │  Merge to     │
                   │  Main         │
                   └───────────────┘
```

This architecture provides:
- High availability through parallel execution
- Fast feedback with strategic caching
- Security through multiple scanning layers
- Quality through comprehensive testing
- Reliability through validation and checks
