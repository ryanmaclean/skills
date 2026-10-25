#!/bin/bash
# SFP/GBIC module diagnostics for UDM Pro
echo "=== SFP Module Diagnostics ==="
for port in eth8 eth9 eth10; do
    echo ""
    echo "--- $port ---"
    result=$(ssh unifi "ethtool -m $port 2>/dev/null" 2>/dev/null)
    if echo "$result" | grep -q "No module"; then
        echo "No module detected"
    elif [ -n "$result" ]; then
        echo "$result" | grep -E "^	(Vendor name|Module temperature|Laser output power|Receiver signal)" | sed 's/^[[:space:]]*//'
        alarms=$(echo "$result" | grep ": On$" | grep -v "threshold" | sed 's/^[[:space:]]*//')
        if [ -n "$alarms" ]; then
            echo ""
            echo "⚠️  ALARMS:"
            echo "$alarms"
        fi
    else
        echo "No module detected"
    fi
done