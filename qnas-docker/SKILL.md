---
name: qnas-docker
description: Docker operations on qnas (QNAP NAS at 10.0.2.35). Docker is not in PATH — use the full container-station binary path for ps, run, exec, logs, pull, and rm over SSH.
---

# qnas-docker

Use for all Docker operations on qnas (QNAP NAS at 10.0.2.35).

## Key Commands

Docker on qnas is NOT in PATH. Use full path:

```
/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker <command>
```

Common examples:

```bash
# List containers
ssh studio@10.0.2.35 '/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker ps -a'

# Run a container
ssh studio@10.0.2.35 '/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker run --rm -v /path:/project -w /project rustlang/rust:nightly <command>'

# Remove container
ssh studio@10.0.2.35 '/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker rm -f <name>'

# Inspect logs
ssh studio@10.0.2.35 '/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker logs <name>'

# Exec into running container
ssh studio@10.0.2.35 '/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker exec -it <name> bash'

# Pull image
ssh studio@10.0.2.35 '/share/ZFS530_DATA/.qpkg/container-station/usr/bin/.libs/docker pull <image>'
```

## Host

- IP: 10.0.2.35
- User: studio
- SSH key: ~/.ssh/id_ed25519 (should work without password)