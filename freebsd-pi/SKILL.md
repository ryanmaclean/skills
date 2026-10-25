---
name: freebsd-pi
description: FreeBSD 15 Raspberry Pi first-boot bootstrap — SSH key push, sshd config, hostname, pkg, static IP, fleet onboarding, SD card flashing. Use when bootstrapping or re-imaging a FreeBSD Pi node.
triggers: freebsd pi, freebsd raspberry, bsd pi, freebsd bootstrap, bsd bootstrap
---

# FreeBSD Pi Bootstrap Skill

## Default Credentials (FreeBSD 15 Pi Images)

| Account  | SSH            | Console      |
|----------|----------------|--------------|
| freebsd  | freebsd/freebsd | freebsd/freebsd |
| root     | disabled (PermitRootLogin no) | root/root    |

**CHANGE BOTH PASSWORDS IMMEDIATELY.** These are public defaults baked into the image.

The `freebsd` user is in the `wheel` group — use `su root` to escalate (password: `root`).

## First-Boot Bootstrap (One-Shot)

Use `expect` from the management host (macOS or Linux). `sshpass` handles the
freebsd user login; `expect` handles the interactive `su` password prompt.

```expect
#!/usr/bin/expect -f
set timeout 60
set pub_key "ssh-ed25519 AAAA... user@host"   ;# paste real key here
set target_ip "192.0.2.10"                    ;# substitute DHCP address
set hostname  "your-hostname"                  ;# substitute your hostname

spawn sshpass -p freebsd ssh -o StrictHostKeyChecking=no freebsd@$target_ip
expect "% "
send "su root\r"
expect "Password:"
send "root\r"
expect "# "

# Push SSH key for root
send "mkdir -p /root/.ssh && chmod 700 /root/.ssh\r"
expect "# "
send "echo '$pub_key' > /root/.ssh/authorized_keys\r"
expect "# "
send "chmod 600 /root/.ssh/authorized_keys\r"
expect "# "

# Enable root SSH login
send "grep -q 'PermitRootLogin yes' /etc/ssh/sshd_config || echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config\r"
expect "# "
send "service sshd restart\r"
expect "# "

# Set hostname
send "hostname $hostname\r"
expect "# "
send "sysrc hostname=\"$hostname\"\r"
expect "# "

# Change passwords from defaults
send "echo 'NEWPASS' | pw usermod root -h 0\r"
expect "# "
send "echo 'NEWPASS' | pw usermod freebsd -h 0\r"
expect "# "

send "uname -a\r"
expect "# "
send "echo BOOTSTRAP_COMPLETE\r"
expect "BOOTSTRAP_COMPLETE"
expect "# "
send "exit\r"
expect "% "
send "exit\r"
expect eof
```

**Verify root key login worked:**
```sh
ssh -o StrictHostKeyChecking=no -i ~/.ssh/id_ed25519 root@<ip> "uname -a"
```

## Hostname

```sh
# Set immediately (runtime)
hostname <your-hostname>

# Persist across reboots
sysrc hostname="<your-hostname>"
```

Default hostname on fresh images is `generic`.

## sshd Configuration

Key lines for `/etc/ssh/sshd_config`:

```
PermitRootLogin yes          # needed for fleet key-based root access
PasswordAuthentication no    # disable after key is in place
PubkeyAuthentication yes
```

After editing: `service sshd restart`

## pkg Bootstrap

Fresh images may not have pkg initialized:

```sh
# Auto-bootstrap pkg (fetches bootstrap binary from FreeBSD CDN)
ASSUME_ALWAYS_YES=yes pkg bootstrap

# Upgrade all ports
pkg upgrade -y

# Essentials for fleet node
pkg install -y python3 doas curl wget git
```

## Timezone

```sh
# Set timezone (interactive)
tzsetup

# Or non-interactive
ln -sf /usr/share/zoneinfo/America/Vancouver /etc/localtime
sysrc ntpd_enable=YES
service ntpd start
```

## Static IP via /etc/rc.conf

DHCP (default):
```sh
sysrc ifconfig_ue0="DHCP"       # USB ethernet (Pi 4/5 or adapter)
sysrc ifconfig_genet0="DHCP"    # Pi 4 built-in gigabit
sysrc ifconfig_smsc0="DHCP"     # Pi 3B USB ethernet
```

Static IP (substitute your network values):
```sh
sysrc ifconfig_genet0="inet <YOUR_IP> netmask <NETMASK>"
sysrc defaultrouter="<GATEWAY>"
echo "nameserver <DNS>" > /etc/resolv.conf
service netif restart
service routing restart
```

Find your NIC: `ifconfig -l` or `dmesg | grep -E 'ue|genet|smsc'`

| Pi Model | NIC driver |
|----------|-----------|
| Pi 3B    | smsc0      |
| Pi 4B    | genet0     |
| Pi 5     | genet0     |
| USB eth  | ue0        |

## Fleet Onboarding (ii-agent)

After root SSH key is in place:

1. Add to Ansible inventory (`ansible/inventory`) — substitute your hostname
   and IP (see `## Deployment-specific details` for the concrete group and
   `ansible_python_interpreter` path used in this fleet):
```ini
[freebsd_pis]
<hostname> ansible_host=<IP> ansible_user=root ansible_python_interpreter=/usr/local/bin/python3.11
```

2. Bootstrap Python (needed for Ansible):
```sh
ssh root@<IP> "ASSUME_ALWAYS_YES=yes pkg bootstrap && pkg install -y python3"
```

3. Run fleet bootstrap playbook:
```sh
ansible-playbook -i ansible/inventory ansible/playbooks/bootstrap-freebsd.yml --limit <hostname>
```

4. Deploy ii-agent:
   - See `rigfleet-ii-agent-ops` skill for ii-agent deploy steps
   - FreeBSD service unit goes in `/usr/local/etc/rc.d/ii_agent`
   - Enable: `sysrc ii_agent_enable=YES && service ii_agent start`

## Pi-Specific: config.txt (Boot Partition)

FreeBSD mounts the FAT32 boot partition at `/boot/msdos/` (or check `df`).
Edit `/boot/msdos/config.txt` for hardware tweaks:

```ini
# Disable overscan (better for monitors)
disable_overscan=1

# Force HDMI output even with no display attached
hdmi_force_hotplug=1

# HDMI safe mode (lower res, useful for troubleshooting)
# hdmi_safe=1

# Set GPU memory split (MB) — keep low for headless
gpu_mem=16

# For Pi 4: force HDMI 1 (HDMI0 port)
# hdmi_group=1
# hdmi_mode=16   # 1080p 60Hz
```

DTB selection: FreeBSD auto-selects the correct `.dtb` from the boot partition.
Manual override in `config.txt`:
```ini
device_tree=bcm2710-rpi-3-b.dtb      # Pi 3B
device_tree=bcm2711-rpi-4-b.dtb      # Pi 4B
```

Boot partition layout:
```
/boot/msdos/
  config.txt         # Pi firmware config
  *.dtb              # Device tree blobs
  overlays/          # DTB overlays
  kernel8.img        # (or u-boot.bin for FreeBSD)
  EFI/               # UEFI boot files
```

## SD Card Flashing from macOS

```sh
# List disks to find SD card
diskutil list

# Unmount (but do NOT eject — keeps /dev/rdiskN available)
diskutil unmountDisk /dev/disk4

# Flash image (use rdisk for raw/fast I/O)
# FreeBSD 15 Pi image: FreeBSD-15.0-RELEASE-arm64-aarch64-RPI.img
xz -d FreeBSD-15.0-RELEASE-arm64-aarch64-RPI.img.xz   # decompress if .xz

sudo dd if=FreeBSD-15.0-RELEASE-arm64-aarch64-RPI.img \
        of=/dev/rdisk4 \
        bs=4m \
        status=progress

# Eject when done
diskutil eject /dev/disk4
```

**WARNING:** Double-check `diskN` — dd with wrong target destroys data.
Use `diskutil list` output to confirm the SD card size matches.

Download images: https://download.freebsd.org/releases/arm64/aarch64/ISO-IMAGES/

## Quick Bootstrap Checklist

- [ ] Flash SD card with FreeBSD Pi image
- [ ] Boot Pi, note DHCP address from router
- [ ] SSH in: `sshpass -p freebsd ssh freebsd@<ip>`
- [ ] `su root` (password: `root`)
- [ ] Push root SSH key to `/root/.ssh/authorized_keys`
- [ ] `echo 'PermitRootLogin yes' >> /etc/ssh/sshd_config`
- [ ] `service sshd restart`
- [ ] Set hostname: `hostname <hostname> && sysrc hostname="<hostname>"`
- [ ] Set static IP in `/etc/rc.conf` if needed
- [ ] `ASSUME_ALWAYS_YES=yes pkg bootstrap && pkg install -y python3`
- [ ] Change passwords: `echo 'newpass' | pw usermod root -h 0`
- [ ] Add to Ansible inventory
- [ ] Verify: `ssh -i ~/.ssh/id_ed25519 root@<ip> uname -a`

## Deployment-specific details

This skill is generic FreeBSD 15 Pi bootstrap knowledge. Everything above
applies to any FreeBSD aarch64 Pi image. Concrete hostnames, IPs, Ansible
inventory group, and fleet-wide rollout state live in the deployment's
fleet-ops docs.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/docs/freebsd.md` — host table, rollout
  state, per-node notes
- `/Users/studio/rust-town/fleet-ops/ansible/inventory` — `[freebsd_pis]`
  group; `ansible_python_interpreter=/usr/local/bin/python3.11` (not `/usr/bin/python3`)
- Static IP subnet: `10.0.2.0/23` (/23 covers both 10.0.2.x and 10.0.3.x);
  default gateway `10.0.2.1`; DNS `10.0.2.1` (UDM Pro)
- Bootstrap playbook: `fleet-ops/ansible/bootstrap-freebsd-pi.yml`

**For a different deployment:** treat the expect script and inventory example
as templates. Substitute your own hostname/IP/gateway/DNS values and point
this section at your own fleet docs.
