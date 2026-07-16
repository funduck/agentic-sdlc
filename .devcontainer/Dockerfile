FROM mcr.microsoft.com/devcontainers/base:ubuntu

RUN apt-get update && apt-get install -y --no-install-recommends \
        iptables \
        ipset \
        iproute2 \
        dnsutils \
        curl \
    && rm -rf /var/lib/apt/lists/*

# Let the non-root "vscode" user (provisioned by the base image) run only the
# firewall init script as root, and nothing else.
COPY init-firewall.sh /usr/local/bin/init-firewall.sh
RUN chmod +x /usr/local/bin/init-firewall.sh \
    && echo "vscode ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/init-firewall \
    && chmod 0440 /etc/sudoers.d/init-firewall

USER vscode
