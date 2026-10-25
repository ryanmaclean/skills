---
name: macvlan-net
description: Macvlan networking topology, limitations, and troubleshooting. Use when deploying containers on macvlan networks, diagnosing connectivity, or choosing where to run services that need internet access.
compatibility: opencode
metadata:
  tools_needed: "Bash, SSH, Read"
  triggers: "macvlan, network, container, connectivity, internet, packet loss, deploy container"
---

# Macvlan Networking Skill

Generic macvlan topology, limitations, and troubleshooting. Deployment
topology (which hosts run macvlan, which services need internet) lives in
the `## Deployment-specific details` footer.

## The One Rule

**Macvlan containers typically have NO outbound internet access.**
Services that need to reach external APIs, download models, or fetch remote
data MUST run on a real host with host or bridge networking.

## Why Macvlan Has No Internet

Docker macvlan gives each container its own MAC address and IP on the LAN.
The container can talk to anything on the local network, but:

1. The upstream gateway/router typically does not NAT traffic from the
   macvlan MAC addresses (no L3 NAT for new unknown MACs on many SOHO
   routers).
2. **The macvlan-host cannot communicate with its own macvlan children**
   — this is a Linux kernel macvlan limitation: the host and its macvlan
   children are isolated at L2.
3. DNS resolution for external domains fails (no upstream DNS forwarding).

Result: 100% packet loss to any IP outside the LAN.

## Before Deploying a New Container

Ask these questions:

1. **Does this service need outbound internet?**
   - Calls external APIs (OpenAI, Anthropic, cloud services) → YES
   - Downloads packages, models, updates at runtime → YES
   - Purely LAN service (database, cache, message broker) → NO

2. **If YES → deploy on a real host** with `--network host` (or bridge
   networking that has a NATted route out).

3. **If NO → macvlan is fine:**
   - Pick an unused IP in the macvlan subnet.
   - Use your container platform's macvlan driver.

## Diagnosing Connectivity Issues

### From a macvlan container

```bash
# LAN connectivity — should SUCCEED
docker exec <container> ping -c 1 <LAN_PEER_IP>

# Internet connectivity — will FAIL on macvlan
docker exec <container> ping -c 1 8.8.8.8

# DNS resolution — will FAIL on macvlan
docker exec <container> ping -c 1 google.com
```

### From a real host (host/bridge networking)

```bash
# Reaching macvlan containers — should SUCCEED from any other real host
ping -c 1 <macvlan_container_ip>

# Reaching other real hosts — should SUCCEED
ping -c 1 <lan_peer_ip>
```

### From the macvlan host itself

```bash
# Host CANNOT reach its own macvlan containers directly.
# This is kernel-level macvlan isolation — not a misconfiguration.
ping -c 1 <macvlan_container_ip>   # May FAIL
```

### Common symptoms and causes

| Symptom                                       | Likely Cause                            |
|-----------------------------------------------|-----------------------------------------|
| Container can't reach external API            | Macvlan has no internet — move to host  |
| Peer host can't reach LAN service on macvlan  | Check Docker is running on macvlan host |
| Container-to-container works, internet not    | Expected macvlan behavior               |
| Host can't reach its own macvlan containers   | Macvlan L2 isolation (kernel limit)     |
| Service was on macvlan, now unreachable       | Container may have restarted/stopped    |

## Workarounds for Internet-Dependent Services

If you must keep a service in a macvlan-constrained environment but it
needs internet:

1. **Move to a host with `--network host`** (preferred)
   ```bash
   docker run -d --network host --name my-service ...
   ```

2. **Reverse proxy on the physical host**
   - Run an nginx/caddy on the macvlan host (not in a container)
   - Proxy external requests through the host, which HAS internet
   - Fragile; adds complexity

3. **Pull-through cache**
   - Pre-download models, packages, data to a volume
   - Mount into the macvlan container
   - Only works for static assets, not live API calls

## Cross-Subnet Service Dependencies

When a service on a real host needs to reach a service on macvlan (or vice
versa):

- **Real host → macvlan:** Works.
- **Macvlan → real host on LAN:** Works.
- **Macvlan → internet:** FAILS. No NAT for macvlan MACs.
- **Macvlan host → its own macvlan containers:** May fail (L2 isolation).

Example connection string from a real host to a macvlan Postgres:
`postgresql://user:pass@<macvlan_ip>:5432/dbname`

## Related

- `ansible-fleet` — deploying across a fleet
- `ssh-fleet` — SSH patterns for reaching macvlan hosts

## Deployment-specific details

This skill is generic macvlan. Everything above applies to any Docker
macvlan deployment. The concrete topology (which hosts run macvlan, which
services need internet access and therefore must live on host networking,
macvlan subnet layout) lives in that deployment's fleet-ops repo.

**For the rust-town fleet:**
- `/Users/studio/rust-town/fleet-ops/docs/macvlan-net.md` — hosts, subnet, service placement

**For a different deployment:** substitute your own subnet layout and
point this section at your own fleet docs.
