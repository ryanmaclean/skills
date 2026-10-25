---
name: qemu-fleet
description: Manage QEMU virtual machines on the fleet (Apple Silicon + Pi nodes). Use when creating, booting, snapshotting, or debugging QEMU VMs on fleet hosts.
version: 1.0.0
triggers:
  - qemu
  - vm
  - virtual machine
  - netbsd vm
  - ubuntu vm
  - hvf
---

# QEMU Fleet VM Management

Patterns for running QEMU VMs on fleet nodes, learned from hard-won debugging sessions.

**RAM policy:** check your deployment's `build-vm.md` (or equivalent doc)
for concurrent VM limits and spin-up/shutdown lifecycle rules before starting
a new VM. See `## Deployment-specific details` at the bottom of this file.

## Starting a VM on Apple Silicon (HVF)

Use `screen` for persistence and a Unix socket for serial console.

```bash
screen -dmS nbuild qemu-system-aarch64 \
  -M virt -accel hvf -cpu host \
  -m 8G -smp 4 \
  -drive file=/path/to/arm64.img,format=raw,if=virtio \
  -bios /opt/homebrew/share/qemu/edk2-aarch64-code.fd \
  -nic user,hostfwd=tcp::2222-:22 \
  -serial unix:/tmp/nb-serial.sock,server,nowait \
  -nographic
```

### Key Flags

| Flag | Purpose |
|------|---------|
| `-accel hvf` | Apple Hypervisor Framework -- 10x faster than TCG software emulation |
| `-cpu host` | Pass through host CPU features (requires HVF) |
| `-serial unix:...,server,nowait` | Serial console via Unix socket. **Critical:** `nowait` prevents boot from blocking on console attachment |
| `-nic user,hostfwd=tcp::2222-:22` | QEMU user-mode NAT with SSH port forwarding |
| `-nographic` | No GUI window, output to serial only |
| `-bios .../edk2-aarch64-code.fd` | UEFI firmware for aarch64 VMs (Homebrew path on macOS) |

### What NOT to Do

- **NEVER use `-serial stdio`** when running in screen/background -- it blocks boot waiting for terminal attachment and causes SSH hangs.
- **NEVER use named pipes (FIFOs) for console** -- they cause EOF on disconnect, killing the VM's serial output permanently.
- **NEVER use `-serial mon:stdio`** in detached mode -- same blocking problem as stdio.

## Serial Console Access

```bash
# Connect to running VM's serial console
nc -U /tmp/nb-serial.sock

# Alternative with socat (more reliable for interactive use)
socat UNIX-CONNECT:/tmp/nb-serial.sock STDIO

# Disconnect: Ctrl-C (nc) or Ctrl-\ (socat) -- VM keeps running
```

## SSH Setup Inside the VM

### CRITICAL: Remove XMSS Keys First

XMSS host key generation takes **hours** on arm64 and blocks sshd startup entirely. Always remove them before starting sshd.

```bash
# Remove XMSS keys (take hours to generate on arm64)
rm -f /etc/ssh/ssh_host_xmss_key*

# Generate only fast key types
ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ""
ssh-keygen -t rsa -f /etc/ssh/ssh_host_rsa_key -N ""
```

### Minimal sshd_config

```bash
cat > /etc/ssh/sshd_config << 'SSHEOF'
PermitRootLogin yes
PubkeyAuthentication yes
PasswordAuthentication no
UseDNS no
HostKey /etc/ssh/ssh_host_ed25519_key
HostKey /etc/ssh/ssh_host_rsa_key
SSHEOF

# Validate before restarting
sshd -t

# Restart sshd (NetBSD)
/etc/rc.d/sshd restart

# Restart sshd (Ubuntu/Debian)
systemctl restart sshd
```

### Connect from Host

```bash
ssh -p 2222 root@localhost
# Or with strict host key checking disabled for ephemeral VMs:
ssh -p 2222 -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@localhost
```

## Screen Session Management

```bash
screen -ls              # List all sessions
screen -r nbuild        # Reattach to VM session
screen -dmS nbuild ...  # Start detached session
screen -S nbuild -X quit  # Kill session (and VM)
```

VMs survive SSH disconnects when running inside screen. The VM process is owned by the screen session, not your shell.

## Port Allocation

Assign each VM host a contiguous port range starting at 2222, one port per
VM. Document allocations in your deployment's `build-vm.md` (or equivalent).
Increment by 1 per VM; never reuse a port on the same host while a VM is
running.

Example convention:
```
host-A: 2222 (VM-1), 2223 (VM-2, reserved), ...
host-B: 2223 (VM-1), 2224 (VM-2), ...
```

For the concrete port assignments in this fleet, see
`## Deployment-specific details` below.

## DNS Workaround (QEMU User-Mode NAT)

QEMU user-mode NAT creates an isolated network. Your fleet's DNS resolver is **not reachable** from inside the VM (the SLIRP stack does not route to your LAN).

```bash
# Inside the VM -- use public DNS
echo "nameserver 8.8.8.8" > /etc/resolv.conf

# Or for more resilience:
cat > /etc/resolv.conf << 'DNSEOF'
nameserver 8.8.8.8
nameserver 1.1.1.1
DNSEOF
```

The host is reachable from inside the VM at `10.0.2.2` (QEMU's default gateway).

## Troubleshooting

### SSH hangs on connect

**Cause:** Serial console is using stdio or a FIFO, blocking the boot process or causing I/O stalls.

**Fix:** Switch to Unix socket serial: `-serial unix:/tmp/nb-serial.sock,server,nowait`

### sshd won't start

**Cause 1:** XMSS key generation running in background, holding up sshd.
```bash
rm -f /etc/ssh/ssh_host_xmss_key*
```

**Cause 2:** Corrupt sshd_config (often from disk-full event truncating the file).
```bash
sshd -t          # Check config syntax
df -h             # Check disk space
```

**Cause 3:** Only `HostKey` lines pointing to keys that don't exist.
```bash
ls -la /etc/ssh/ssh_host_*_key  # Verify keys exist
ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ""  # Regenerate
```

### Disk full

```bash
df -h
# Common space hogs:
rm -rf /tmp/*
rm -rf /root/.cargo/registry/cache/*    # Cargo download cache
rm -rf /root/target/                     # Cargo build artifacts
```

**Prevention:** Use virtio disk images sized appropriately (20G+ for build VMs). Monitor with `df -h` before large builds.

### IPv6 connection hangs

Some operations (pkg_add, curl) try IPv6 first and hang when it's not routed.

```bash
# NetBSD: disable IPv6
sysctl -w net.inet6.ip6.accept_rtadv=0

# Linux: disable IPv6
sysctl -w net.ipv6.conf.all.disable_ipv6=1

# Or force IPv4-only in DNS resolution
echo "precedence ::ffff:0:0/96  100" >> /etc/gai.conf  # Linux
```

### HVF not available

```bash
# Check HVF support
qemu-system-aarch64 -accel help
# Should list: hvf

# If missing, QEMU may not be built with HVF support
# Reinstall via Homebrew:
brew reinstall qemu
```

HVF requires macOS and Apple Silicon (or Intel with Hypervisor.framework). Falls back to TCG (software emulation) which is ~10x slower.

### VM won't boot (EFI)

Verify the BIOS/firmware path exists:
```bash
ls /opt/homebrew/share/qemu/edk2-aarch64-code.fd
# Intel Mac path: /usr/local/share/qemu/edk2-aarch64-code.fd
```

## Quick Reference: Full VM Lifecycle

```bash
# 1. Start VM in screen
screen -dmS myvm qemu-system-aarch64 \
  -M virt -accel hvf -cpu host -m 4G -smp 2 \
  -drive file=disk.img,format=raw,if=virtio \
  -bios /opt/homebrew/share/qemu/edk2-aarch64-code.fd \
  -nic user,hostfwd=tcp::2222-:22 \
  -serial unix:/tmp/myvm-serial.sock,server,nowait \
  -nographic

# 2. Access serial console (for initial setup)
nc -U /tmp/myvm-serial.sock

# 3. Inside VM: fix SSH
rm -f /etc/ssh/ssh_host_xmss_key*
ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ""
sshd -t && /etc/rc.d/sshd restart

# 4. SSH from host
ssh -p 2222 root@localhost

# 5. Shutdown VM gracefully (from inside)
shutdown -p now    # NetBSD
shutdown -h now    # Linux

# 6. Or kill from host
screen -S myvm -X quit
```

## Deployment-specific details

This skill is generic QEMU/HVF knowledge. Everything above applies to any
Apple Silicon host running QEMU VMs. Concrete host coordinates, port
assignments, RAM policy, and virtiofs share paths for a specific deployment
live in that deployment's fleet-ops docs.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/docs/build-vm.md` — VM inventory,
  SSH access patterns, port table, Ansible groups, RAM policy, and QEMU
  launch wrapper details
- `/Users/studio/rust-town/fleet-ops/ansible/inventory` — source of truth
  for `[freebsd_vms]` / `[netbsd_vms]` / `[freebsd_builders]` host groups

**For a different deployment:** treat the port table and SSH examples above
as templates. Substitute your own host names/IPs and document allocations in
your own `build-vm.md` equivalent.
