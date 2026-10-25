---
name: qnap-snmp-triage
description: End-to-end QNAP SNMP monitoring triage across Datadog NDM, pup metrics, SSH reachability, UDP/161, and optional snmpwalk checks.
---

# QNAP SNMP Triage

Use this skill when QNAP/NAS SNMP metrics, traps, or Datadog Network Device Monitoring status are missing, stale, noisy, or inconsistent.

## Scope

This skill verifies the full path:

1. Local name resolution and SSH reachability.
2. SNMP UDP/161 reachability from this workstation when possible.
3. Optional direct `snmpwalk` proof, if credentials are provided in the environment.
4. Datadog-side metric liveness with `pup`.
5. Tag identity consistency for both QNAP storage devices.

## Canonical Devices

Use deterministic addresses for automation.

| Device | Preferred target | Aliases seen | Notes |
|---|---:|---|---|
| qnas | `10.0.2.35` | `qnas.local`, `10.0.3.22`, `10.0.12.2` | Prefer `10.0.2.35` for 10GbE path. Local SSH and UDP/161 have been observed working, but Datadog NDM may not be onboarded. |
| qnas2 | `10.0.3.123` | `qnas2`, `qnas2.local` | Datadog tags have shown `device_hostname:qnas2`, `snmp_device:qnas2`, and `device_ip:10.0.3.123`. |

Do not put SNMP community strings, v3 passwords, OAuth URLs, or Datadog tokens in this skill.

## Quick Start

Run the read-only probe:

```bash
nu ~/.codex/skills/qnap-snmp-triage/scripts/qnap_snmp_check.nu
```

Run a single target:

```bash
nu ~/.codex/skills/qnap-snmp-triage/scripts/qnap_snmp_check.nu --target qnas2
```

Run with optional SNMP proof:

```bash
export QNAP_SNMP_VERSION=2c
export QNAP_SNMP_COMMUNITY='...'
nu ~/.codex/skills/qnap-snmp-triage/scripts/qnap_snmp_check.nu --snmpwalk
```

For SNMPv3:

```bash
export QNAP_SNMP_VERSION=3
export QNAP_SNMP_USER='...'
export QNAP_SNMP_AUTH_PROTO=SHA
export QNAP_SNMP_AUTH_PASS='...'
export QNAP_SNMP_PRIV_PROTO=AES
export QNAP_SNMP_PRIV_PASS='...'
nu ~/.codex/skills/qnap-snmp-triage/scripts/qnap_snmp_check.nu --snmpwalk
```

## Datadog Checks

Use `pup` for Datadog-side proof.

```bash
pup auth status
pup metrics list --filter='qnap' --output=json
pup metrics tags list snmp.qnap.disk.temperature --from='2h' --output=json
pup metrics query --query='avg:snmp.device.reachable{snmp_device:qnas2} by {device_ip,device_hostname,host}' --from='2h' --to='now' --output=json
```

Useful QNAP metrics:

- `snmp.device.reachable`
- `snmp.device.unreachable`
- `snmp.qnap.disk.temperature`
- `snmp.qnap.disk.capacity_bytes`
- `snmp.qnap.disk.count`
- `snmp.qnap.raid.count`
- `snmp.qnap.storagepool.status`
- `snmp.qnap.storagepool.free_bytes`
- `snmp.qnap.cache.available_percent`
- `snmp.qnap.cache.read_hit_rate`
- `snmp.qnap.cache.write_hit_rate`

## Device-Side Checks

SSH should work before deeper checks:

```bash
ssh studio@10.0.2.35 true
ssh studio@qnas2.local true
```

UDP checks are less definitive than `snmpwalk`, but failures are useful:

```bash
nc -vzu -w 3 10.0.2.35 161
nc -vzu -w 3 10.0.3.123 161
```

Direct SNMPv2c proof, if credentials are allowed:

```bash
snmpwalk -v2c -c "$QNAP_SNMP_COMMUNITY" 10.0.3.123 1.3.6.1.2.1.1
```

Direct SNMPv3 proof:

```bash
snmpwalk -v3 -l authPriv -u "$QNAP_SNMP_USER" \
  -a "$QNAP_SNMP_AUTH_PROTO" -A "$QNAP_SNMP_AUTH_PASS" \
  -x "$QNAP_SNMP_PRIV_PROTO" -X "$QNAP_SNMP_PRIV_PASS" \
  10.0.3.123 1.3.6.1.2.1.1
```

## Diagnosis Rules

- SSH fails and SNMP fails: likely host/network/path issue.
- SSH works and SNMP direct fails: likely QNAP SNMP service, ACL, version, or credential issue.
- SNMP direct works and Datadog is stale: likely Datadog Agent SNMP config, profile, namespace, or tag problem.
- SNMP direct works and Datadog has no reachable series: the device is probably not onboarded in NDM, or it is monitored under a different namespace/tag.
- `snmp.device.reachable` is `1` but QNAP-specific metrics are stale: likely profile/OID mapping issue.
- Metrics exist under the wrong hostname/IP: normalize tags and prefer deterministic target IPs.
- Polling misses short events: add SNMP traps and syslog, not faster polling by default.

## Security Rules

- Prefer SNMPv3 authPriv when the device and collector support it.
- If SNMPv2c is used, restrict source IPs to the Datadog Agent/collector.
- Never use default `public` or `private` community strings.
- Treat v1/v2c community strings as sensitive even though they are weak.
- Keep SNMP read-only unless a specific management workflow requires write access.

## TRIZ Design Note

Contradiction: the workflow needs shared checks across both NAS devices, but each device has different addresses, tags, and network paths.

Resolution: segment target identity from check logic. The script keeps a small device table and runs the same proof sequence against each target.

## Resources

- Helper: [scripts/qnap_snmp_check.nu](scripts/qnap_snmp_check.nu)
- Datadog SNMP metrics: https://docs.datadoghq.com/network_monitoring/devices/snmp_metrics/
- Datadog SNMP traps: https://docs.datadoghq.com/network_monitoring/devices/snmp_traps/
- Prometheus SNMP exporter: https://github.com/prometheus/snmp_exporter
