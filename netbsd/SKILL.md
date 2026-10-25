---
name: netbsd
description: NetBSD system administration — pkg_add/pkgin, rc.d, kernel, QEMU VM on Apple Silicon. Use when managing any NetBSD host.
---

# NetBSD Administration

## Not Linux, Not FreeBSD

| Linux | FreeBSD | NetBSD |
|-------|---------|--------|
| `apt install` | `pkg install` | `pkg_add` or `pkgin install` |
| `systemctl` | `service X start` | `/etc/rc.d/X start` |
| `/usr/bin/` | `/usr/local/bin/` | `/usr/pkg/bin/` |
| `sudo` | `doas` | `su root` (default) |
| bash | csh | sh |

## Package Management

**pkg_add** (base system, always available):
```sh
export PKG_PATH=https://cdn.netbsd.org/pub/pkgsrc/packages/NetBSD/aarch64/10.1/All
export PATH=/usr/sbin:/usr/bin:/bin:/usr/pkg/bin:/usr/pkg/sbin:$PATH
pkg_add rust git curl
pkg_info              # list installed
pkg_delete rust       # remove
```

**pkgin** (if installed — like apt for NetBSD):
```sh
pkgin install rust git
pkgin update
pkgin upgrade
pkgin search rust
```

IMPORTANT: `pkg_add` is at `/usr/sbin/pkg_add` — not in default PATH for non-root shells.

## PATH

Default PATH is minimal. Always set:
```sh
export PATH=/usr/sbin:/usr/bin:/bin:/usr/pkg/bin:/usr/pkg/sbin:$PATH
```

Packages install to `/usr/pkg/` (not `/usr/local/` like FreeBSD).

## Services (rc.d)

```sh
# Enable at boot
echo "sshd=YES" >> /etc/rc.conf

# Start/stop
/etc/rc.d/sshd start
/etc/rc.d/sshd stop
/etc/rc.d/sshd restart

# List services
ls /etc/rc.d/
```

## SSH

Default NetBSD images: root with no password, SSH enabled but empty password rejected.

Fix:
```sh
passwd root                    # set password
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config
/etc/rc.d/sshd restart
```

Or add key directly:
```sh
mkdir -p /root/.ssh
echo "ssh-ed25519 AAAA..." > /root/.ssh/authorized_keys
chmod 700 /root/.ssh && chmod 600 /root/.ssh/authorized_keys
```

## DNS

Must configure manually:
```sh
echo "nameserver 8.8.8.8" > /etc/resolv.conf
```

## QEMU VM on Apple Silicon

### Automated standup (preferred)

```sh
nu ~/.claude/skills/netbsd/standup-vm.nu
# Options: --ram 8G --disk 20G --ssh-port 2222 --name nb --dir /tmp/netbsd-vm
# Handles: image download, QEMU in screen, SSH key injection, DNS, PATH, pkgin
```

### Manual standup

```sh
# Download official image
curl -LO https://cdn.netbsd.org/pub/NetBSD/NetBSD-10.1/evbarm-aarch64/binary/gzimg/arm64.img.gz
gunzip arm64.img.gz
qemu-img resize arm64.img 20G

# Boot with hvf acceleration + entropy
qemu-system-aarch64 -M virt -accel hvf -cpu host -smp 4 -m 8G \
  -bios /opt/homebrew/share/qemu/edk2-aarch64-code.fd \
  -drive if=virtio,file=arm64.img \
  -device virtio-rng-pci \
  -device virtio-net-pci,netdev=net0 \
  -netdev user,id=net0,hostfwd=tcp::2222-:22 \
  -nographic
```

### Screen management (CRITICAL — VM dies without this)

```sh
# ALWAYS launch in screen — bare QEMU dies on SSH disconnect
screen -dmS nb qemu-system-aarch64 ...

# Send commands to VM console
screen -S nb -p 0 -X stuff $'command\r'

# Capture screen output to file
screen -S nb -p 0 -X hardcopy /tmp/out.txt
cat /tmp/out.txt

# Attach to console
screen -r nb
```

### Known failure patterns

| Symptom | Cause | Fix |
|---------|-------|-----|
| VM dies on SSH disconnect | QEMU not in screen/tmux | Always `screen -dmS nb` |
| pkg_add hangs | DNS not configured | `echo nameserver 8.8.8.8 > /etc/resolv.conf` FIRST |
| `command not found` for pkg_add | PATH missing /usr/sbin | Export full PATH (see PATH section) |
| SSH rejected | Empty password disallowed | Inject key via screen console, not SSH |
| pkg_add rust stalls | Large download on slow VM | Use prebuilt Rust binary or cross-compile instead |
| Boot takes 2+ min | Normal for QEMU aarch64 | Poll SSH, don't assume failure early |

### Prebuilt Rust (faster than pkg_add)

If `pkg_add rust` stalls (common on aarch64 VMs), cross-compile from macOS instead:
```sh
# On macOS (Apple Silicon) — install NetBSD cross target if available
rustup target list | grep netbsd
# If no NetBSD target: build static musl binary on a Linux host and scp over
# Or: pkg_add rust-bin (prebuilt binary package, smaller download)
```

## Ansible

```ini
<host> ansible_host=<IP> ansible_user=root ansible_become=yes ansible_become_method=su
```

Only `raw` module works (no Python on NetBSD by default). Install Python first: `pkg_add python311`

## Deployment-specific details

This skill is generic NetBSD. Everything above applies to any NetBSD host.
The concrete hosts, IPs, and rollout state for a specific deployment live
in that deployment's fleet-ops repo.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/docs/netbsd.md` — host table (e.g. `nbrpi301` at `10.0.3.130`, NetBSD 10.1 aarch64, Pi 3B+, raw-module-only Ansible) and rollout state
- `/Users/studio/rust-town/fleet-ops/ansible/inventory/` — source of truth for hosts/groups

**For a different deployment:** substitute your own hostnames/IPs and
point this section at your own fleet docs (e.g. `<your-fleet>/docs/netbsd.md`).
