---
name: atem-mini
description: Manage webcam streaming from an ATEM Mini connected to Raspberry Pis — USB-C webcam capture (macOS avfoundation, Linux v4l2), HDMI input switching via the ATEM network protocol (UDP 9910), raw-socket Python control, Ansible playbooks, and IRC bot dispatch. Use when switching ATEM inputs, streaming Pi webcams, or debugging HDMI/USB video paths.
trigger: atem, atem mini, blackmagic, hdmi switch, webcam switch, video switcher, fleet dashboard, pi console
---

# ATEM Mini — Webcam Streaming & Fleet Integration

Lab-bench setup: Raspberry Pis have HDMI out to ATEM Mini HDMI inputs. USB-C output acts as a
USB webcam. ATEM Ethernet port allows programmatic input switching.

## Hardware overview

```
[Pi 1] HDMI ---+
[Pi 2] HDMI ---+  ATEM Mini  --(USB-C)-->  Mac/Linux: /dev/video0 or AVFoundation
[Pi 3] HDMI ---+  (Ethernet: 192.168.x.x)
[Pi 4] HDMI ---+
```

- ATEM Mini: 4 HDMI inputs, source IDs 1-4
- USB-C output: UVC-class webcam, no driver needed on macOS 11+ or Linux 5.x+
- Ethernet: ATEM protocol over UDP port 9910

## 1. USB-C Webcam Capture

### macOS — AVFoundation (ffmpeg)

```bash
# List devices
ffmpeg -f avfoundation -list_devices true -i "" 2>&1 | grep -i atem

# Capture to file (replace "0" with device index)
ffmpeg -f avfoundation -framerate 30 -video_size 1920x1080 -i "0" -c:v libx264 -preset fast out.mp4

# Low-latency preview
ffplay -f avfoundation -framerate 30 -video_size 1920x1080 -i "0"

# Single-frame snapshot
ffmpeg -f avfoundation -framerate 30 -video_size 1920x1080 -i "0" -vframes 1 /tmp/atem-snap.jpg
```

### Linux — v4l2

```bash
# List devices
v4l2-ctl --list-devices

# Capture
ffmpeg -f v4l2 -framerate 30 -video_size 1920x1080 -i /dev/video0 -c:v libx264 -preset fast out.mp4

# Snapshot
ffmpeg -f v4l2 -framerate 30 -video_size 1920x1080 -i /dev/video0 -vframes 1 /tmp/atem-snap.jpg
```

udev rule for stable symlink:
```
# /etc/udev/rules.d/99-atem-mini.rules
SUBSYSTEM=="video4linux", ATTRS{idVendor}=="1edb", ATTRS{idProduct}=="bd6e", SYMLINK+="atem-mini"
```

USB IDs: Vendor `1edb` (Blackmagic Design), Product `bd6e` (ATEM Mini), `bd6f` (ATEM Mini Pro).

## 2. ATEM Network Protocol (UDP 9910)

### Source IDs

| Source ID | Meaning      |
|-----------|--------------|
| 1         | HDMI Input 1 |
| 2         | HDMI Input 2 |
| 3         | HDMI Input 3 |
| 4         | HDMI Input 4 |
| 0         | Black        |
| 1000      | Color Bars   |

### Minimal raw-socket switcher (Python, stdlib only)

License note: PyATEMMax=GPL-3.0, pyatem=LGPL-3.0 — both disallowed. This uses only stdlib.

```python
#!/usr/bin/env python3
"""atem_switch.py — Minimal ATEM Mini input switcher (stdlib only)
Usage: python3 atem_switch.py 192.168.1.240 2
"""
import socket, struct, sys

ATEM_PORT = 9910
INIT_PKT = b'\x10\x14\x53\xab\x00\x00\x00\x00\x00\x3a\x00\x00'

def atem_switch(ip: str, source_id: int, timeout: float = 3.0):
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    s.settimeout(timeout)
    addr = (ip, ATEM_PORT)

    # Initiate connection
    s.sendto(INIT_PKT, addr)
    data, _ = s.recvfrom(4096)
    session_id = struct.unpack_from('>H', data, 2)[0]
    atem_pkt_id = struct.unpack_from('>H', data, 10)[0]

    # ACK the init
    ack = struct.pack('>BBHHHH', 0x80, 0x0c, session_id, atem_pkt_id, 0, 0)
    s.sendto(ack, addr)

    # Drain state dump
    s.settimeout(0.2)
    try:
        while True:
            data, _ = s.recvfrom(65535)
            atem_pkt_id = struct.unpack_from('>H', data, 10)[0]
    except socket.timeout:
        pass
    s.settimeout(timeout)

    # CPgI command: switch program input
    cmd = struct.pack('>HH4sHH', 10, 0, b'CPgI', 0, source_id)
    header = struct.pack('>BBHHHH', 0x01, 12 + len(cmd), session_id, atem_pkt_id, 0, 1)
    s.sendto(header + cmd, addr)

    s.settimeout(1.0)
    try:
        s.recvfrom(4096)
    except socket.timeout:
        pass
    s.close()

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print(f'Usage: {sys.argv[0]} <atem-ip> <source 1-4>')
        sys.exit(1)
    atem_switch(sys.argv[1], int(sys.argv[2]))
    print(f'Switched program to input {sys.argv[2]}')
```

### Command tags

| Tag    | Action               |
|--------|----------------------|
| `CPgI` | Change Program Input |
| `CPvI` | Change Preview Input |
| `DAut` | Auto transition      |
| `DCut` | Cut transition       |

## 3. Fleet Integration — Ansible

Convention: Pi N = ATEM input N.

```bash
# Switch to Pi 3
ansible-playbook playbooks/atem_switch.yml -e "input=3"

# Cycle all inputs with snapshots
ansible-playbook playbooks/atem_cycle.yml
```

## 4. Pi HDMI Gotchas

Force HDMI output (Pis won't output without detected display):
```
# /boot/config.txt
hdmi_force_hotplug=1
hdmi_group=1
hdmi_mode=16
```

## 5. ATEM Network Setup

```bash
# Find ATEM on subnet
nmap -sU -p 9910 192.168.1.0/24 2>/dev/null | grep -B2 "open"
```

Set static IP via ATEM Software Control > Preferences > Network.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| ffmpeg stalls on avfoundation | Match -video_size/-framerate to ATEM output |
| Black/green frame | USB cable is power-only, use data cable |
| UDP 9910 no response | Wrong IP, run nmap probe |
| PyATEMMax or pyatem | GPL/LGPL — disallowed, use stdlib script |
| Pi shows "No Signal" | Add hdmi_force_hotplug=1 to /boot/config.txt |

## Related Skills

- `raspberry-pi` — Pi fleet management, boot config
- `ansible-fleet` — safety-first rollout, inventory
- `fleet-irc` — IRC dispatch, #agents channel
