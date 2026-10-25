#!/bin/bash
# Get DHCP leases from UDM Pro with human-readable output
ssh unifi "cat /data/udapi-config/dnsmasq.lease" 2>/dev/null | awk '{print $3, $4}' | column -t
