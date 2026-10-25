#!/bin/bash
# Discover Unifi devices on the network
echo "=== Unifi Device Discovery ==="
ssh unifi "ubnt-tools ubnt-discover" 2>/dev/null
echo ""
echo "=== ARP Table ==="
ssh unifi "arp -a" 2>/dev/null
echo ""
echo "=== Routing Table ==="
ssh unifi "netstat -rt -n" 2>/dev/null
