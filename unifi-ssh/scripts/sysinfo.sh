#!/bin/bash
# System info summary for UDM Pro
echo "=== UDM Pro System Info ==="
ssh unifi "info" 2>/dev/null
echo ""
echo "=== CPU Temp ==="
ssh unifi "ubnt-systool cputemp" 2>/dev/null
echo ""
echo "=== CPU Load ==="
ssh unifi "ubnt-systool cpuload" 2>/dev/null
echo ""
echo "=== Sensors ==="
ssh unifi "sensors" 2>/dev/null
echo ""
echo "=== Disk Usage ==="
ssh unifi "du -hs /data" 2>/dev/null
echo ""
echo "=== Uptime ==="
ssh unifi "uptime" 2>/dev/null
echo ""
echo "=== SFP Modules ==="
~/.agents/skills/unifi-ssh/scripts/sfp-temps.sh 2>/dev/null
