# Skill: Linux System Administration

## Overview

Linux is the dominant operating system for servers and containers. This skill covers essential command-line operations, file system management, process control, networking, and user management.

## Key Concepts

- **Shell**: The command-line interpreter (e.g., Bash, Zsh).
- **File system hierarchy**: Standard directory layout under `/` (root).
- **Permissions**: Read/write/execute bits for owner, group, and others.
- **Process**: A running instance of a program, identified by a PID.
- **Package manager**: Tool to install, update, and remove software (e.g., `apt`, `dnf`, `yum`).
- **Systemd**: The init system and service manager used by most modern Linux distributions.

## Common Tasks

### Navigate the file system
```bash
pwd                  # print working directory
ls -lah              # list files with details
cd /path/to/dir      # change directory
```

### View and edit files
```bash
cat file.txt         # print file contents
less file.txt        # scroll through a file
nano file.txt        # simple editor
vim file.txt         # advanced editor
```

### Search for content
```bash
grep -r "pattern" /path/to/dir
find /var/log -name "*.log" -mtime -1
```

### Manage file permissions
```bash
chmod 644 file.txt           # rw-r--r--
chmod +x script.sh           # add execute bit
chown user:group file.txt    # change owner
```

### Install packages (Debian/Ubuntu)
```bash
sudo apt update && sudo apt install -y <package>
```

### Manage services with systemd
```bash
sudo systemctl start   <service>
sudo systemctl stop    <service>
sudo systemctl enable  <service>   # start on boot
sudo systemctl status  <service>
journalctl -u <service> -f         # follow logs
```

### Monitor system resources
```bash
top                  # interactive process viewer
htop                 # improved process viewer
df -h                # disk space usage
free -h              # memory usage
vmstat 1             # system performance stats
```

### Manage users and groups
```bash
sudo useradd -m -s /bin/bash username
sudo passwd username
sudo usermod -aG sudo username
sudo deluser username
```

### Network diagnostics
```bash
ip addr show                    # show IP addresses
ss -tlnp                        # listening TCP ports
curl -I https://example.com     # HTTP headers
ping -c 4 8.8.8.8               # ICMP reachability
traceroute 8.8.8.8              # route tracing
```

## Best Practices

- Use `sudo` instead of running as root for interactive sessions.
- Prefer `ssh` key-based authentication; disable password login.
- Schedule routine tasks with `cron` or `systemd` timers.
- Rotate and centralise logs (e.g., with `logrotate` and a SIEM).
- Keep packages up to date and apply security patches promptly.
- Use `ufw` or `firewalld` to restrict inbound traffic.

## References

- [Linux Command Reference](https://man7.org/linux/man-pages/)
- [The Linux Documentation Project](https://tldp.org/)
- [Systemd Documentation](https://systemd.io/)
