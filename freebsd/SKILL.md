---
name: freebsd
description: FreeBSD system administration — pkg, doas, rc.conf, jails, pf, Raspberry Pi. Use when managing any FreeBSD host.
---

# FreeBSD Administration

Generic FreeBSD knowledge. For concrete host inventories and fleet-specific
rollout state, see the `## Deployment-specific details` section at the
bottom of this file.

## Not Linux — Key Differences

| Linux | FreeBSD |
|-------|---------|
| `sudo` | `doas` (or `su root`) |
| `apt install` / `dnf install` | `pkg install` |
| `systemctl enable/start` | `sysrc X_enable=YES` + `service X start` |
| `/usr/bin/python3` | `/usr/local/bin/python3` |
| `iptables` / `nftables` | `pf` (`/etc/pf.conf`) |
| Docker | Jails |
| `.bashrc` | `.cshrc` (default shell is csh) |
| `su -` | `su root` |

## Package Management (pkg)

```sh
pkg install python3 git vim    # install
pkg upgrade                     # upgrade all
pkg search curl                 # search
pkg audit                       # security audit
pkg delete nginx                # remove
pkg info                        # list installed
```

All packages install to `/usr/local/` (bin, etc, lib).

## Service Management (rc)

```sh
sysrc sshd_enable=YES           # enable at boot
sysrc nginx_enable=NO           # disable
service sshd start              # start now
service sshd restart            # restart
service sshd status             # check status
service -l                      # list all services
```

Config: `/etc/rc.conf` (main), `/etc/rc.conf.d/servicename` (per-service overrides).

## Privilege Escalation

**doas** (if installed):
```sh
pkg install doas
echo "permit nopass keepenv :wheel" > /usr/local/etc/doas.conf
chmod 600 /usr/local/etc/doas.conf
```

**su root** (always works, needs root password):
```sh
su root
# do admin stuff
exit
```

**No sudo by default.** Don't install it — doas is simpler and sufficient.

**Non-interactive password setting** (FreeBSD-specific):
```sh
echo 'NEW_PASSWORD' | pw usermod root -h 0   # pw reads password from fd 0 (stdin)
echo 'NEW_PASSWORD' | pw usermod freebsd -h 0
```

**wheel group:** Only grants ability to run `su(1)`. Does NOT grant filesystem access to /root/ or any other root-owned paths.

## Raspberry Pi Default Credentials

FreeBSD Pi images ship with public default credentials:
- **freebsd / freebsd** (regular user, in wheel group)
- **root / root** (root account)

**CHANGE ROOT PASSWORD IMMEDIATELY** after first access. Default is public knowledge.

PermitRootLogin defaults to `no` in FreeBSD sshd_config -- root SSH is disabled out of the box.

## Bootstrap: Fresh FreeBSD Pi (No doas/sudo/Python)

**Problem:** Fresh image has no pkg, no Python, no doas. `su root` requires interactive tty. Need to escalate non-interactively.

**Solution: Ansible `raw` module + `become_method=su` + `become_pass`**

The `raw` module does not need Python on the remote host. Ansible handles the su password prompt internally.

```yaml
---
- name: Bootstrap FreeBSD Pi (no Python required)
  hosts: freebsd_bootstrap
  gather_facts: false
  become: yes
  become_method: su
  vars:
    ansible_become_password: "root"   # default Pi image password

  tasks:
    - name: Bootstrap pkg
      ansible.builtin.raw: "ASSUME_ALWAYS_YES=yes pkg bootstrap"

    - name: Install Python 3 + doas
      ansible.builtin.raw: "pkg install -y python3 doas"

    - name: Configure doas for wheel group
      ansible.builtin.raw: |
        echo 'permit nopass keepenv :wheel' > /usr/local/etc/doas.conf
        chmod 600 /usr/local/etc/doas.conf

    - name: Change root password from default
      ansible.builtin.raw: "echo 'SECURE_PASSWORD_HERE' | pw usermod root -h 0"
```

**Known issue:** Ansible su + FreeBSD can hang (ansible/ansible#23699). If it hangs, use expect fallback:

```expect
#!/usr/bin/expect -f
set timeout 120
spawn ssh -o StrictHostKeyChecking=no freebsd@$HOST
expect "$ "
send "su root\r"
expect "Password:"
send "root\r"
expect "# "
send "ASSUME_ALWAYS_YES=yes pkg bootstrap && pkg install -y python3 doas\r"
expect "# "
send "echo 'permit nopass keepenv :wheel' > /usr/local/etc/doas.conf && chmod 600 /usr/local/etc/doas.conf\r"
expect "# "
send "exit\r"
expect "$ "
send "exit\r"
expect eof
```

**After bootstrap**, update inventory to use doas (example — substitute your own hostname/IP):
```ini
pi-example ansible_host=192.0.2.10 ansible_user=freebsd ansible_python_interpreter=/usr/local/bin/python3 ansible_become=yes ansible_become_method=doas
```

**SD card mount is NOT viable** — macOS cannot write FreeBSD UFS2. Linux can (`mount -t ufs -o ufstype=ufs2,rw`).

## Ansible

Inventory vars for FreeBSD hosts (pre-bootstrap, su mode):
```ini
pi-example ansible_host=192.0.2.10 ansible_user=freebsd ansible_become=yes ansible_become_method=su
```

Post-bootstrap (doas mode):
```ini
pi-example ansible_host=192.0.2.10 ansible_user=freebsd ansible_python_interpreter=/usr/local/bin/python3 ansible_become=yes ansible_become_method=doas
```

Python must be installed first: `pkg install python3`

## Shells

- Root: `/bin/csh` (keep it — safe if /usr/local unmounts)
- Users: `/bin/sh` (POSIX)
- Bash: `pkg install bash && chsh -s /usr/local/bin/bash username`

For Ghostty terminal:
```sh
# On FreeBSD host, add to ~/.cshrc or ~/.profile:
setenv TERM xterm-256color
# Or in SSH config on client side:
# Host my-freebsd-host
#     SetEnv TERM=xterm-256color
```

## Raspberry Pi Notes

- Image: `FreeBSD-14.3-RELEASE-arm64-aarch64-RPI.img.xz`
- Default hostname: `generic` — change with `sysrc hostname="<your-hostname>"` + `hostname <your-hostname>`
- Ethernet works out of box (smsc driver)
- WiFi: not supported natively, use USB dongle
- 1GB RAM on Pi 3B — keep services minimal

## Jails (Containers)

```sh
# Install jail management
pkg install bastille

# Create a thin jail (RFC 5737 TEST-NET address used as example)
bastille create web 14.3-RELEASE 192.0.2.100

# Start/stop
bastille start web
bastille stop web

# Shell into jail
bastille console web
```

## PF Firewall

Enable: `sysrc pf_enable=YES`

Basic `/etc/pf.conf`:
```
pass quick on lo0
block in all
pass out all keep state
pass in proto tcp to port { 22, 80, 443 }
```

```sh
pfctl -f /etc/pf.conf    # load rules
pfctl -sr                 # show rules
pfctl -ss                 # show state table
```

## Deployment-specific details

This skill is generic FreeBSD. Everything above applies to any FreeBSD
host. The concrete hosts, IPs, and rollout state for a specific deployment
live in that deployment's fleet-ops repo.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/docs/freebsd.md` — host table, rollout state
- `/Users/studio/rust-town/fleet-ops/ansible/inventory/` — source of truth for hosts

**For a different deployment:** treat the inventory examples above as
templates. Substitute your own hostnames/IPs, and point this section at
your own fleet docs (e.g. `<your-fleet>/docs/freebsd.md`).
