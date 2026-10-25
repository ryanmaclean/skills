---
name: unifi
description: Read-only forensics against the UDM Pro (10.0.3.5) — dnsmasq leases, DHCP client history, ARP table, firewall state. Use when you need to answer "is host X actually online", "what new devices joined the LAN", "what IP does MAC Y have", "why can't I reach host Z". Does NOT modify UDM Pro config — that is high-risk and requires explicit per-task user approval per CLAUDE.md.
trigger: unifi, udm pro, dnsmasq, dhcp lease, fleet census, who is on the lan, mac lookup, dhcp client, new device, arp, 10.0.3.5
---

# UniFi / UDM Pro — Read-Only Forensics

The UDM Pro at `10.0.3.5` is the fleet's router + DHCP server + DNS + firewall.
This skill is the canonical pattern for **read-only** queries: who has a lease,
who's ARP-active right now, which hostnames have landed since a given time.
**Write operations (config changes, restarts, firewall edits) are NOT covered
by this skill — those are high-risk and require explicit per-task user
approval per the project CLAUDE.md.**

Studio is on `10.0.2.0/23` and can reach `10.0.3.5` directly.

## Access pattern

SSH directly as root with the fleet's SSH key. If your user's standard key
works for other fleet hosts, it probably works here too. If auth fails, STOP
and ask — never guess.

```bash
ssh root@10.0.3.5 '<command>'
```

Prefer `ansible udmpro -m shell -a '<command>' -i openclaw/openclaw/ansible/inventory`
if the UDM Pro is in the inventory — it respects the no-password-guess policy.

## Where to find the dnsmasq leases

UDM Pro firmware scatters lease state across a few paths. Try in order:

1. **`/data/udapi-config/dnsmasq.lease`** — primary live file, dnsmasq format:
   `<expiry_epoch> <mac> <ip> <hostname_or_*> <client_id>`
2. **`/run/dnsmasq.dns.conf.d/hosts.d/leases`** — hosts-file mirror, handy for
   grep-by-hostname
3. **`/mnt/data/udapi-config/dnsmasq.lease`** — some firmwares mount here instead
4. **Fallback**: `find /run /data /mnt -name '*dnsmasq*lease*' -o -name 'dhcp*leases*' 2>/dev/null`

Don't trust any single path blindly — verify the file was recently touched
(`stat` → mtime < 5 min for an active DHCP server).

## Cross-check with live state

The lease file says "host X has a lease". It does NOT say "host X is answering
right now". Liveness requires **live ICMP or TCP probes from the gateway
itself**, not just the ARP cache.

**⚠️ Critical correction (added after 2026-04-11 incident):** `/proc/net/arp`
flag `0x2` means the kernel has a cached MAC↔IP mapping. It does **NOT** mean
"host is alive". The mapping can persist for a long time after a host goes
dark. An earlier agent misread `0x2` as "alive" and reported 5 dead Pis as
"online but sshd-regressed", which sent the incident response down the wrong
path. Don't repeat that mistake.

The correct liveness probe is ICMP from the UDM Pro:

```bash
# Actual L3 liveness — run ON THE UDM PRO, ping each suspected host
for ip in 10.0.3.70 10.0.3.71 10.0.3.72; do
  ping -c 1 -W 1 -q "$ip" >/dev/null 2>&1 \
    && echo "$ip alive" \
    || echo "$ip DARK"
done
```

If ICMP is blocked on the target hosts (some Linux distros firewall it by
default), fall back to a TCP probe on a known-open service port (`22`, `80`,
`443`, whatever is expected on that host):

```bash
for ip in 10.0.3.70 10.0.3.71; do
  (echo > /dev/tcp/$ip/22) 2>/dev/null \
    && echo "$ip tcp/22 open" \
    || echo "$ip tcp/22 closed-or-dark"
done
```

`/proc/net/arp` is still useful — but only as a **secondary** signal: a host
with an ARP entry AND failing ICMP is "was alive recently, now dark", which
is different from "never seen". Don't trust ARP alone to say "alive".

## Parse pattern (sort + bucket)

```bash
# leases: epoch mac ip hostname client_id
awk '{print $1, $2, $3, $4}' /data/udapi-config/dnsmasq.lease \
  | while read expiry mac ip hostname; do
      now=$(date +%s)
      remain=$((expiry - now))
      printf "%s\t%s\t%s\t%ds\n" "$ip" "$mac" "$hostname" "$remain"
    done | sort
```

Bucket hosts into:

- **ONLINE_EXPECTED** — hostname matches inventory AND ARP flag 0x2
- **OFFLINE_EXPECTED** — in inventory, no lease or expired lease, or ARP 0x0
- **NEW_UNEXPECTED** — not in inventory, matches Pi/fleet naming or Pi MAC prefix

## Raspberry Pi MAC prefixes

Useful when the DHCP hostname is empty (`*`):

- `b8:27:eb:` — Pi 3 / Zero W
- `dc:a6:32:` — Pi 4 (8GB+)
- `d8:3a:dd:` — Pi 5
- `e4:5f:01:` — Pi 5
- `28:cd:c1:` — Pi CM4

A lease with `hostname=*` from one of these MAC prefixes is a **misbehaving
Pi** — cloud-init / netplan probably didn't register the hostname. That's
a smoking gun for a netplan regression (see
`.planning/sshd-regression-ubrpi405-forensics-2026-04-11.md` for the 2026-04-11
investigation of exactly this pattern).

## Common investigations

### "Is host X actually online?"

```bash
# 1. Does it have a lease?
grep -i 'ubrpi405' /data/udapi-config/dnsmasq.lease
# 2. Is it ARP-reachable right now?
grep -i '10.0.3.' /proc/net/arp | awk '$3 == "0x2"' | grep <ip>
```

If lease yes + ARP yes, the Pi is on the LAN and the problem is service-layer
(sshd, firewall, cloud-init). If lease yes + ARP no, the Pi booted once but
is currently dark. If no lease, the Pi hasn't been on the network in at least
the lease window (default 12h or 24h).

### "What new devices joined the LAN recently?"

```bash
# Sort leases by expiry (newest first)
sort -nr /data/udapi-config/dnsmasq.lease | head -20
```

Recent leases at the top. Cross-check against inventory to find unknown
hostnames / MACs.

### "What's this MAC?"

```bash
grep -i '<mac>' /data/udapi-config/dnsmasq.lease /proc/net/arp
```

## What this skill does NOT do

- **No config writes.** No `unifi-os restart`, no `uci set`, no firewall rule
  edits, no DHCP reservation changes. Those require explicit per-task user
  approval per `openclaw/CLAUDE.md` §Remote Access Policy.
- **No port/service probes from the UDM Pro itself.** Run those from studio
  or a Pi — the UDM Pro is shared infrastructure.
- **No controller UI / API writes.** If you need to provision a new device
  or change an SSID, that's a user-driven GUI task.

## Related

- `ssh-fleet` — access config, auth policy, no-password-guess rule
- `macvlan-net` — why some 10.0.3.x hosts are containers, not physical
- `rigfleet-fleet-inventory` — source of truth for "which hostname is in our inventory"
- `.planning/udmpro-dnsmasq-census-2026-04-11.md` — worked example of a full
  fleet census using this skill's patterns
