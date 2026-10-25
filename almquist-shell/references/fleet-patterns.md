## 7. Fleet Script Patterns

### Portable shebang
```sh
#!/bin/sh
# Always use /bin/sh — never /bin/bash on BSD fleet nodes
```

### Standard error-safe header
```sh
#!/bin/sh
set -eu
# For FreeBSD sh targets only — add pipefail:
# set -euo pipefail
```

### Temp files
```sh
tmpfile=$(mktemp /tmp/myscript.XXXXXX)
trap 'rm -f "$tmpfile"' EXIT

# mktemp is available on FreeBSD, NetBSD, and dash systems
```

### Logging helper
```sh
log() { printf '[%s] %s\n' "$(date '+%Y-%m-%dT%H:%M:%S')" "$*" >&2; }
die() { log "ERROR: $*"; exit 1; }

log "Starting deployment"
[ -f /etc/rc.conf ] || die "/etc/rc.conf not found"
```

### Remote execution via Ansible raw module
```sh
# In playbook YAML:
- name: Run script on BSD host
  raw: /bin/sh /path/to/script.sh
  # Do NOT use shell: module with bash-specific syntax
```

### getopts option parsing
```sh
#!/bin/sh
usage() { printf 'Usage: %s [-v] [-o file] arg\n' "$0" >&2; exit 1; }

verbose=0
outfile=""
while getopts 'vo:h' opt; do
  case "$opt" in
    v) verbose=1 ;;
    o) outfile="$OPTARG" ;;
    h) usage ;;
    ?) usage ;;
  esac
done
shift $((OPTIND - 1))
```

### rc.d service script (FreeBSD)
```sh
#!/bin/sh

# PROVIDE: myservice
# REQUIRE: NETWORKING
# KEYWORD: shutdown

. /etc/rc.subr

name="myservice"
rcvar="myservice_enable"
command="/usr/local/sbin/myserviced"
command_args="--config /usr/local/etc/myservice.conf"
pidfile="/var/run/${name}.pid"

load_rc_config "$name"
run_rc_command "$1"
```

Enable/disable:
```sh
sysrc myservice_enable=YES
service myservice start
service myservice status
```

### rc.d service script (NetBSD)
```sh
#!/bin/sh

# PROVIDE: myservice
# REQUIRE: NETWORKING

$_rc_subr_loaded . /etc/rc.subr

name="myservice"
rcvar=$name
command="/usr/pkg/sbin/myserviced"

load_rc_config $name
run_rc_command "$1"
```

Enable in `/etc/rc.conf`:
```sh
myservice=YES
```

### Checking installed packages portably
```sh
# FreeBSD:
pkg info -e python3 && echo "installed" || echo "not installed"

# NetBSD (pkgsrc):
pkgin list | grep -q "^python3" && echo "installed" || echo "not installed"
```

## 8. Why NOT bash on Fleet Nodes

| Reason | Detail |
|--------|--------|
| **GPL-3.0 license** | bash is GPL-3.0 — copyleft, violates project license policy (MIT/BSD/Apache-2.0 only) |
| **Not in FreeBSD base** | bash requires `pkg install bash` (ports); not present on minimal/TinyBSD installs |
| **Not in NetBSD base** | bash requires `pkgin install bash`; not present on base installs |
| **TinyBSD contract** | TinyBSD v1 defaults: zero added packages, `/bin/sh` only |
| **Size and speed** | ash is ~92–200 KB; bash is several MB. On Raspberry Pi Zero/1, matters |
| **Complexity** | bash bashisms creep into scripts and break portability silently |
| **Ansible default** | Ansible `shell:` module defaults to `/bin/sh`; bash must be explicitly forced |

### ash is sufficient
Everything needed for fleet automation — service control, config templating,
health checks, deploy hooks — is expressible in POSIX sh. The constraints
are features: they enforce portable, auditable scripts.

