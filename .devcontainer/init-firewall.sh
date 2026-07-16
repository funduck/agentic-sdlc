#!/usr/bin/env bash
# Default-deny egress firewall for the devcontainer.
# Runs at container start (iptables rules do not persist across restarts).
set -euo pipefail
IFS=$'\n\t'

echo "[init-firewall] flushing existing rules"
iptables -F
iptables -X
ipset destroy allowed-domains 2>/dev/null || true

echo "[init-firewall] blocking all IPv6 (only IPv4 is allowlisted below)"
if command -v ip6tables > /dev/null 2>&1; then
    ip6tables -F
    ip6tables -P INPUT DROP
    ip6tables -P FORWARD DROP
    ip6tables -P OUTPUT DROP
fi

echo "[init-firewall] setting default-deny policy"
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT DROP

# Always allow loopback and established/related traffic.
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow DNS resolution.
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT
iptables -A INPUT -p udp --sport 53 -j ACCEPT
iptables -A INPUT -p tcp --sport 53 -j ACCEPT

ipset create allowed-domains hash:ip

ALLOWED_DOMAINS=(
    api.anthropic.com
    statsig.anthropic.com
    github.com
    api.github.com
    codeload.github.com
    objects.githubusercontent.com
    registry.npmjs.org
)

echo "[init-firewall] resolving allowlisted domains"
for domain in "${ALLOWED_DOMAINS[@]}"; do
    ips=$(getent ahostsv4 "$domain" | awk '{print $1}' | sort -u || true)
    if [ -z "$ips" ]; then
        echo "[init-firewall] WARNING: could not resolve $domain, skipping" >&2
        continue
    fi
    while read -r ip; do
        [ -n "$ip" ] && ipset add allowed-domains "$ip" 2>/dev/null || true
    done <<< "$ips"
done

# Allow HTTPS (and HTTP for redirects/package installs) only to allowlisted IPs.
iptables -A OUTPUT -p tcp --dport 443 -m set --match-set allowed-domains dst -j ACCEPT
iptables -A OUTPUT -p tcp --dport 80 -m set --match-set allowed-domains dst -j ACCEPT

echo "[init-firewall] self-test: verifying firewall behaves as expected"

fail=0

# Note: no -f here — any HTTP response (even 404) proves the TCP/TLS connection
# got through the firewall, which is all this self-test cares about.
if curl --max-time 5 -sS -o /dev/null https://api.anthropic.com 2>&1; then
    echo "[init-firewall] OK: allowlisted host api.anthropic.com is reachable"
else
    echo "[init-firewall] FAIL: allowlisted host api.anthropic.com is NOT reachable" >&2
    fail=1
fi

if curl --max-time 5 -sS -o /dev/null https://example.com 2>&1; then
    echo "[init-firewall] FAIL: non-allowlisted host example.com IS reachable (firewall not blocking)" >&2
    fail=1
else
    echo "[init-firewall] OK: non-allowlisted host example.com is blocked"
fi

if [ "$fail" -ne 0 ]; then
    echo "[init-firewall] firewall self-test FAILED, refusing to continue" >&2
    exit 1
fi

echo "[init-firewall] firewall is active and verified"
