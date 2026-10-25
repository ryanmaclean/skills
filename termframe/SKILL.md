---
name: termframe
description: >
  Capture and render terminal output as SVG screenshots using termframe for
  command output, remote sessions, and OpenCode reporting workflows.
---

# Termframe Skill

Capture and display terminal output as SVG screenshots in OpenCode. Uses [termframe](https://github.com/pamburus/termframe) - a non-interactive terminal emulator that renders ANSI output to SVG.

## Installation

### macOS
```bash
brew install termframe
```

### Linux (x86_64)
```bash
curl -sSfL -o /usr/local/bin/termframe https://github.com/pamburus/termframe/releases/latest/download/termframe-linux-x86_64-musl.tar.gz | sudo tar xz
```

### Linux (arm64)
```bash
curl -sSfL -o /usr/local/bin/termframe https://github.com/pamburus/termframe/releases/latest/download/termframe-linux-arm64-musl.tar.gz | sudo tar xz
```

### Via Cargo
```bash
cargo install --locked --git https://github.com/pamburus/termframe.git
```

## Usage

### Capture Command Output

```bash
# Basic capture
termframe -o output.svg -- echo "Hello"

# Auto-size based on output
termframe --width auto --height auto -o output.svg -- ls -la

# Specify dimensions
termframe --width 120 --height 40 -o output.svg -- htop
```

### Capture from Stdin
```bash
echo "Hello, World" | termframe -o hello.svg
```

### Use with Fleet Nodes

Capture remote terminal state via SSH:
```bash
ssh ubrpi502 'htop' | termframe -o htop.svg
```

### Capture Zellij Session State
```bash
# Attach to session and capture
ssh ubrpi502 'zellij capture-pane 2>/dev/null' | termframe -o session.svg
```

### Custom Themes

Create themes in `~/.config/termframe/themes/`:
```toml
# ~/.config/termframe/themes/my-theme.toml
[colors]
background = "#1e1e2e"
foreground = "#cdd6f4"
```

Use custom theme:
```bash
termframe --theme my-theme -o output.svg -- command
```

## OpenCode Integration

Use termframe to capture agent work and display in conversations:

1. **Capture test results**:
   ```bash
   cd ~/rust-town/openclaw && cargo test 2>&1 | termframe -o test-results.svg
   ```

2. **Capture build output**:
   ```bash
   cd ~/rust-town/openclaw && cargo build --release 2>&1 | termframe -o build.svg
   ```

3. **Capture IRC output**:
   ```bash
   ssh ubrpi502 'irssi' 2>&1 | termframe -o irc.svg
   ```

4. **Fleet-wide status capture**:
   ```bash
   for host in ubrpi401 ubrpi502 ubrpi501; do
     ssh $host 'uptime && df -h' | termframe -o "status-$host.svg"
   done
   ```

## Key Files

- Config: `~/.config/termframe/config.toml`
- Themes: `~/.config/termframe/themes/`
- Window styles: `~/.config/termframe/window-styles/`
- Binary: `/usr/local/bin/termframe`

## Examples

### Capture Fleet Health Dashboard
```bash
# Create a comprehensive fleet status capture
cat > /tmp/fleet-status.sh << 'EOF'
#!/bin/bash
echo "=== Fleet Status Report ==="
echo "Time: $(date)"
echo ""
for host in ubrpi401 ubrpi502 ubrpi501 ubrpi408; do
  echo "--- $host ---"
  ssh -o ConnectTimeout=5 $host 'echo "Uptime: $(uptime)" && echo "Memory: $(free -h | grep Mem)"' 2>/dev/null || echo "Unreachable"
  echo ""
done
EOF
chmod +x /tmp/fleet-status.sh
/tmp/fleet-status.sh | termframe -o fleet-status.svg
```

### Capture Zellij Session Pane
```bash
# Capture specific zellij pane output
ssh ubrpi502 'zellij command pause "scroll-focus"' 2>/dev/null
ssh ubrpi502 'zellij dump-buffer > /tmp/session.log' 2>/dev/null || \
  ssh ubrpi502 'cat /tmp/session.log' | termframe -o zellij-pane.svg
```

## Configuration

Default config location: `~/.config/termframe/config.toml`

```toml
[terminal]
width = 120
height = 40

[theme]
name = "Catppuccin Mocha"

[window]
style = "rounded"
