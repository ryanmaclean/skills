---
name: esp32-tang-bridge
description: ESP32-C6 RISC-V UART-to-IRC bridge for Tang FPGA console — ESP-IDF v5.3 build, USB flash from studio, WiFi to fleet LAN, plaintext IRC to Ergo, WireGuard support. Use when working on the Tang FPGA serial bridge, ESP32-C6 firmware, or UART-to-IRC plumbing.
trigger: esp32, tang bridge, tang console, esp-idf, risc-v bridge, uart irc, tang-bridge
---

# ESP32-C6 Tang Bridge — UART-to-IRC Fleet Agent

RISC-V microcontroller (Seeed XIAO ESP32-C6) bridging Tang FPGA console UART to IRC.

## Hardware

```
Tang Console UART TX/RX ──── ESP32-C6 XIAO ──── WiFi ──── Ergo IRC
     (3.3V, no level shifter needed)        (AERO SSID)   (10.0.3.203:6667)
```

- **MCU:** ESP32-C6, single-core RISC-V 160MHz, 512KB SRAM, 4MB flash
- **WiFi:** 802.11b/g/n 2.4GHz (CYW43439 equivalent)
- **UART:** GPIO 20 (RX from Tang), GPIO 21 (TX to Tang), 115200 baud
- **USB:** /dev/cu.usbmodem214401 on studio (MAC 58:8c:81:5d:22:70)
- **IRC nick:** `tang-bridge` in `#agents`

## Architecture

NOT ii-agent. Custom C firmware using ESP-IDF. The ESP32-C6 is too constrained
for Zig/ii-agent. It runs a minimal IRC client that:

1. Connects to WiFi (AERO SSID)
2. Opens TCP to Ergo IRC at 10.0.3.203:6667 (plaintext — no TLS, not enough RAM)
3. Registers as `tang-bridge`, joins `#agents`
4. Forwards UART RX from Tang Console to IRC as `[tang] <text>`
5. Forwards IRC messages to UART TX for Tang Console display
6. WireGuard tunnel support (trombik/esp_wireguard, BSD-3-Clause)

## Project Location

`/Users/studio/rust-town/esp32-tang-bridge/`

## Build

**Requires:** ESP-IDF v5.3+ (RISC-V toolchain for esp32c6)

```bash
# ESP-IDF setup (one-time)
cd ~/esp/esp-idf && source export.sh

# Build
cd ~/rust-town/esp32-tang-bridge
idf.py set-target esp32c6
idf.py build
```

**Build boxes:**
- studio (macOS): ESP-IDF v5.3.0 installed at `~/esp/esp-idf` — works now
- pop4090 (Linux): ESP-IDF installing in background — use for future builds
- i9-zfs-pop, i7-zfs-pop: no ESP-IDF

**Common build issues:**
- Stale CMake cache from old path: `idf.py fullclean` then rebuild
- Component namespace: use `trombik/esp_wireguard` (slash), not `trombik__esp_wireguard` (double underscore)

## Flash

ESP32-C6 is physically connected to studio via USB.

```bash
cd ~/rust-town/esp32-tang-bridge
source ~/esp/esp-idf/export.sh
idf.py -p /dev/cu.usbmodem214401 flash

# Monitor serial output
idf.py -p /dev/cu.usbmodem214401 monitor
```

Flash partitions:
- 0x0: bootloader
- 0x8000: partition table
- 0x10000: app (`tang-bridge.bin`, ~877KB, 16% free on partition)

## Configuration

WiFi and IRC settings in `sdkconfig` (menuconfig) or `main/main.c`:

```
CONFIG_IDF_TARGET="esp32c6"
CONFIG_IDF_TARGET_ARCH="riscv"
```

- WiFi SSID: `AERO` (credentials in sdkconfig)
- IRC server: `10.0.3.203:6667` (plaintext)
- IRC nick: `tang-bridge`
- IRC channel: `#agents`
- UART baud: 115200

## Wiring

```
Tang Console          ESP32-C6 XIAO
-----------           -------------
UART TX  ──────────>  GPIO 20 (RX)
UART RX  <──────────  GPIO 21 (TX)
GND      ──────────>  GND
```

Both 3.3V logic — no level shifter needed.

## Ansible

The ESP32-C6 is NOT SSH-managed. It's in the inventory for documentation only:

```ini
[esp32]
tang-bridge ansible_host=TBD ansible_python_interpreter=/usr/bin/false # ESP32-C6 RISC-V, UART-IRC bridge, not SSH-managed
```

Build and flash operations happen on studio or build boxes, not via Ansible to the device.

## License

- ESP-IDF: Apache-2.0
- trombik/esp_wireguard: BSD-3-Clause
- Project code: MIT

No GPL dependencies.

## Related Skills

- `atem-mini` — HDMI switching for the Tang Console video
- `fleet-irc` — IRC protocol, Ergo server
- `zig-cc` — cross-compile for other targets (NOT used for ESP32-C6)
