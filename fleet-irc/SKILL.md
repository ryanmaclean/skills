---
name: fleet-irc
description: Use when communicating with fleet agents, checking agent status, or sending commands via IRC. The fleet runs TLS IRC on 10.0.3.203; agents listen in #agents and respond to DMs. Use for real-time agent coordination, status checks, and out-of-band command delivery when ansible/SSH is unavailable or slow.
---

# Fleet IRC — Agent Communication via irssi

## Access

IRC server: `10.0.3.203` (irclaw macvlan container on QNAS 10.0.2.35), TLS
Nick to use: `ryan`
Primary channel: `#agents`

```bash
# Connect via irssi on the IRC container host
ssh ubuntu@10.0.3.203 'irssi --connect=10.0.3.203 --nick=ryan --tls'

# Send a single message and exit (non-interactive)
ssh ubuntu@10.0.3.203 'irssi --connect=10.0.3.203 --nick=ryan --tls \
  --eval "/msg #agents hello" --eval "/quit"' 2>/dev/null

# Or use netcat directly for TLS (requires openssl s_client)
echo -e "NICK ryan\r\nUSER ryan 0 * :ryan\r\nJOIN #agents\r\nPRIVMSG #agents :message\r\nQUIT\r\n" \
  | openssl s_client -connect 10.0.3.203:6697 -quiet 2>/dev/null
```

## Non-Interactive Messaging (preferred for automation)

```bash
# Post to #agents channel
ssh ubuntu@10.0.3.203 "echo 'PRIVMSG #agents :your message here' | \
  openssl s_client -connect 127.0.0.1:6697 -quiet 2>/dev/null" &

# DM a specific agent (e.g. irclaw bot named "ztudio")
ssh ubuntu@10.0.3.203 "echo 'PRIVMSG ztudio :run status' | \
  openssl s_client -connect 127.0.0.1:6697 -quiet 2>/dev/null" &
```

## Ansible Alternative

The IRC host is reachable via ansible group or direct:
```yaml
# inventory entry
irc-host ansible_host=10.0.3.203 ansible_user=ubuntu
```

```bash
ansible -i inventory irc-host -m shell -a 'irssi --connect=...'
```

## Agent Names in #agents

Agents announce themselves on join. Common names:
- irclaw bots: usually match hostname (e.g. `ubrpi407-zclaw`)
- Coordinator agents: `i9-agent`, `i7-agent`

## Use Cases

| Situation | Use IRC when |
|-----------|-------------|
| Check agent alive | ping via DM faster than SSH |
| Broadcast to fleet | single `#agents` message vs ansible loop |
| SSH unreachable | IRC runs independently of gastown/ansible |
| Real-time coordination | watch #agents during deploys |
| Out-of-band commands | agent can exec commands and report back |

## Notes

- Server is on macvlan IP 10.0.3.203 — NOT on i9 (10.0.2.230)
- TLS port: 6697 (standard IRC TLS)
- Plain port: 6667 (if TLS fails, fallback)
- irssi is installed on the container host itself
