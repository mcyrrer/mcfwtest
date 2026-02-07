# AGENT.md - Claude Agent Guide for FWProbe

This document provides instructions for AI agents on how to use the FWProbe tool and how to generate valid FWProbe configuration files from arbitrary Linux system configuration files (iptables, nftables, firewalld, ufw, etc.).

---

## Table of Contents

1. [Using the FWProbe Command](#using-the-fwprobe-command)
2. [Configuration File Format](#configuration-file-format)
3. [Creating Config Files from Linux Firewall Rules](#creating-config-files-from-linux-firewall-rules)
4. [Examples: Converting Real Configs](#examples-converting-real-configs)
5. [Validation and Troubleshooting](#validation-and-troubleshooting)

---

## Using the FWProbe Command

### Installation

```bash
# Build from source
go build -o fwprobe ./cmd/fwprobe

# Or install directly
go install github.com/mcyrrer/mcfwtest/cmd/fwprobe@latest
```

### Core Commands

#### Generate a Template Config

```bash
# Print template to stdout
fwprobe createconfig

# Write template to a file
fwprobe createconfig -o myconfig.yaml
```

#### Validate a Config File

```bash
fwprobe validate -c myconfig.yaml
```

This checks YAML syntax, field validity, and constraint rules. It reports endpoint count and estimated total tests.

#### Run Firewall Tests

```bash
# Run with default config (fwprobe.yaml)
fwprobe run

# Run with a specific config
fwprobe run -c myconfig.yaml

# Run with JSON output
fwprobe run -c myconfig.yaml -o json

# Run with JUnit XML output (for CI/CD)
fwprobe run -c myconfig.yaml -o junit

# Filter to specific endpoints by glob pattern
fwprobe run -f "web-*"

# Stop on first failure
fwprobe run --fail-fast

# Only show failures
fwprobe run --quiet

# Limit to IPv4 addresses only
fwprobe run --ipv4-only

# Set concurrency (default: 10)
fwprobe run -j 20
```

#### Check Version

```bash
fwprobe version
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All tests passed |
| 1 | One or more tests failed |
| 2 | Configuration error (invalid YAML, validation failure) |
| 3 | Runtime error (permissions, network error) |

---

## Configuration File Format

Every FWProbe config file must follow this schema:

```yaml
version: "1"                # Required. Must be "1".

defaults:                   # Optional. Global defaults.
  timeout: 2s               # Connection timeout. Range: 100ms - 30s. Default: 2s.
  protocol: tcp             # Default protocol. Values: tcp, udp. Default: tcp.

endpoints:                  # Required. List of test definitions.
  - name: <unique-name>     # Required. Unique string identifier.
    description: "..."      # Optional. Human-readable label.
    host: <ip-or-hostname>  # One host, IP, or CIDR range.
    # OR
    hosts:                  # Multiple hosts (mutually exclusive with host).
      - <ip-or-hostname>
      - <ip-or-hostname>
    port: <number>          # Single port, 1-65535.
    # OR
    ports:                  # Multiple ports (mutually exclusive with port).
      - <number>
      - <number>
    protocol: tcp           # tcp or udp. Overrides global default.
    timeout: 3s             # Per-endpoint timeout. Overrides global default.
    expect: open            # Expected state: "open" or "closed".
```

### Constraints

- `version` must be `"1"`
- `name` is required and must be unique across all endpoints
- Each endpoint must have exactly one of `host` or `hosts` (not both, not neither)
- Each endpoint must have exactly one of `port` or `ports` (not both, not neither)
- Ports must be in range 1-65535
- `protocol` must be `tcp` or `udp`
- `timeout` must be between 100ms and 30s
- `expect` must be `open` or `closed`
- CIDR ranges are supported up to /16 (65,534 hosts max)
- Unknown fields are rejected (strict YAML parsing)

---

## Creating Config Files from Linux Firewall Rules

The following sections describe how to read common Linux firewall configuration formats and convert them into valid FWProbe test configurations.

### General Strategy

1. **Identify the firewall system** in use (iptables, nftables, firewalld, ufw, or custom)
2. **Extract rules** that ACCEPT or DROP/REJECT traffic on specific ports
3. **Map each rule** to an FWProbe endpoint:
   - ACCEPT rules become `expect: open`
   - DROP/REJECT rules become `expect: closed`
4. **Determine hosts**: Use the destination IP, network range, or the server's own IP
5. **Determine ports and protocols**: Extract from the rule's `--dport` / port specification
6. **Assign unique names**: Derive from the service name, port, or rule comment

### From iptables Rules

**Reading iptables rules:**

```bash
# Dump current rules
sudo iptables -L -n -v
sudo iptables-save > /tmp/iptables-rules.txt

# For IPv6
sudo ip6tables -L -n -v
```

**iptables rule format:**

```
-A INPUT -p tcp --dport 22 -j ACCEPT
-A INPUT -p tcp --dport 3306 -j DROP
-A INPUT -s 10.0.0.0/24 -p tcp --dport 443 -j ACCEPT
-A INPUT -p udp --dport 53 -j ACCEPT
```

**Conversion rules:**

| iptables field | FWProbe field |
|----------------|---------------|
| `-p tcp` or `-p udp` | `protocol: tcp` or `protocol: udp` |
| `--dport 22` | `port: 22` |
| `--dport 80:443` (range) | `ports: [80, 81, ..., 443]` or test boundary ports |
| `-s 10.0.0.0/24` | source context (use the server IP as `host`) |
| `-d 192.168.1.0/24` | `host: 192.168.1.0/24` |
| `-j ACCEPT` | `expect: open` |
| `-j DROP` or `-j REJECT` | `expect: closed` |
| `-m comment --comment "SSH"` | `name:` and `description:` |

**Example conversion:**

iptables input:
```
-A INPUT -p tcp --dport 22 -m comment --comment "SSH" -j ACCEPT
-A INPUT -p tcp --dport 80 -j ACCEPT
-A INPUT -p tcp --dport 443 -j ACCEPT
-A INPUT -p tcp --dport 3306 -j DROP
-A INPUT -p udp --dport 53 -j ACCEPT
```

FWProbe output:
```yaml
version: "1"

defaults:
  timeout: 2s
  protocol: tcp

endpoints:
  - name: ssh-access
    description: "SSH should be open"
    host: <server-ip>
    port: 22
    expect: open

  - name: http-https
    description: "Web ports should be open"
    host: <server-ip>
    ports:
      - 80
      - 443
    expect: open

  - name: mysql-blocked
    description: "MySQL should be blocked"
    host: <server-ip>
    port: 3306
    expect: closed

  - name: dns-udp
    description: "DNS over UDP should be open"
    host: <server-ip>
    port: 53
    protocol: udp
    expect: open
```

### From nftables Rules

**Reading nftables rules:**

```bash
sudo nft list ruleset
sudo nft list ruleset > /tmp/nftables-rules.txt
```

**nftables rule format:**

```
table inet filter {
    chain input {
        type filter hook input priority 0; policy drop;
        tcp dport 22 accept
        tcp dport { 80, 443 } accept
        udp dport 53 accept
        tcp dport 3306 drop
    }
}
```

**Conversion rules:**

| nftables field | FWProbe field |
|----------------|---------------|
| `tcp dport 22` | `protocol: tcp`, `port: 22` |
| `tcp dport { 80, 443 }` | `protocol: tcp`, `ports: [80, 443]` |
| `udp dport 53` | `protocol: udp`, `port: 53` |
| `accept` | `expect: open` |
| `drop` / `reject` | `expect: closed` |
| `ip saddr 10.0.0.0/24` | source context (use server IP as host) |
| `ip daddr 192.168.1.0/24` | `host: 192.168.1.0/24` |
| `policy drop` | default policy means unlisted ports should be `expect: closed` |

### From firewalld (zones/services)

**Reading firewalld config:**

```bash
# List active zones
sudo firewall-cmd --get-active-zones

# List allowed services in a zone
sudo firewall-cmd --zone=public --list-all

# List specific ports
sudo firewall-cmd --zone=public --list-ports
sudo firewall-cmd --zone=public --list-services
```

**Example firewalld output:**

```
public (active)
  target: default
  interfaces: eth0
  services: ssh http https dns
  ports: 8080/tcp 9090/tcp
  rich rules:
    rule family="ipv4" source address="10.0.0.0/24" port port="3306" protocol="tcp" accept
```

**Conversion:** Map each listed service to its well-known port, and each explicit port entry directly:

```yaml
version: "1"

defaults:
  timeout: 2s
  protocol: tcp

endpoints:
  - name: ssh
    description: "SSH service allowed in public zone"
    host: <server-ip>
    port: 22
    expect: open

  - name: web-services
    description: "HTTP and HTTPS in public zone"
    host: <server-ip>
    ports:
      - 80
      - 443
    expect: open

  - name: dns
    description: "DNS service in public zone"
    host: <server-ip>
    port: 53
    protocol: udp
    expect: open

  - name: custom-ports
    description: "Custom TCP ports allowed"
    host: <server-ip>
    ports:
      - 8080
      - 9090
    expect: open

  - name: mysql-from-internal
    description: "MySQL allowed from 10.0.0.0/24"
    host: <server-ip>
    port: 3306
    expect: open
```

### From UFW (Uncomplicated Firewall)

**Reading UFW rules:**

```bash
sudo ufw status verbose
sudo ufw status numbered
```

**Example UFW output:**

```
Status: active

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW       Anywhere
80/tcp                     ALLOW       Anywhere
443/tcp                    ALLOW       Anywhere
3306/tcp                   DENY        Anywhere
8080/tcp                   ALLOW       10.0.0.0/24
```

**Conversion:**

| UFW field | FWProbe field |
|-----------|---------------|
| `22/tcp` | `port: 22`, `protocol: tcp` |
| `ALLOW` | `expect: open` |
| `DENY` | `expect: closed` |
| `Anywhere` | test from any source against the server IP |
| `10.0.0.0/24` | source restriction (note: test from within that network) |

### From Custom Application Configs

Many services define their listen ports in configuration files. These can also be converted to FWProbe tests.

**Common config file locations:**

| Service | Config Path | Port Field |
|---------|-------------|------------|
| SSH | `/etc/ssh/sshd_config` | `Port 22` |
| Nginx | `/etc/nginx/nginx.conf` | `listen 80;` / `listen 443 ssl;` |
| Apache | `/etc/httpd/conf/httpd.conf` | `Listen 80` |
| MySQL | `/etc/mysql/my.cnf` | `port = 3306` |
| PostgreSQL | `/etc/postgresql/*/main/postgresql.conf` | `port = 5432` |
| Redis | `/etc/redis/redis.conf` | `port 6379` |
| Docker | `/etc/docker/daemon.json` | `"hosts": ["tcp://0.0.0.0:2376"]` |
| Kubernetes API | `/etc/kubernetes/manifests/kube-apiserver.yaml` | `--secure-port=6443` |

**Strategy:**

1. Read the service config to find the listen port and bind address
2. Determine if the service should be externally accessible or internal-only
3. Create endpoints with `expect: open` for services that should be reachable and `expect: closed` for services that should be firewalled off from external access

**Example: Nginx config to FWProbe:**

Nginx config (`/etc/nginx/nginx.conf`):
```
server {
    listen 80;
    listen 443 ssl;
    server_name example.com;
}

server {
    listen 8080;
    server_name internal.example.com;
}
```

FWProbe config:
```yaml
version: "1"

defaults:
  timeout: 3s
  protocol: tcp

endpoints:
  - name: nginx-http
    description: "Public HTTP port"
    host: example.com
    port: 80
    expect: open

  - name: nginx-https
    description: "Public HTTPS port"
    host: example.com
    port: 443
    expect: open

  - name: nginx-internal
    description: "Internal port should not be publicly accessible"
    host: example.com
    port: 8080
    expect: closed
```

### From /etc/services and ss/netstat

**Reading active listeners:**

```bash
# Show listening ports
sudo ss -tlnp    # TCP listeners
sudo ss -ulnp    # UDP listeners
sudo netstat -tlnp
```

This provides a snapshot of what is actually running and listening, which can be cross-referenced with firewall rules to generate comprehensive tests.

---

## Examples: Converting Real Configs

### Complete Workflow Example

Given a server at `192.168.1.100` with these rules:

```bash
# iptables-save output
*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [0:0]
-A INPUT -i lo -j ACCEPT
-A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
-A INPUT -p tcp --dport 22 -j ACCEPT
-A INPUT -p tcp --dport 80 -j ACCEPT
-A INPUT -p tcp --dport 443 -j ACCEPT
-A INPUT -p tcp --dport 3306 -s 10.0.0.0/24 -j ACCEPT
-A INPUT -p udp --dport 123 -j ACCEPT
COMMIT
```

Step-by-step conversion:

1. Default INPUT policy is DROP, so unlisted ports are closed
2. Port 22 (SSH) is open to all
3. Ports 80, 443 (HTTP/S) are open to all
4. Port 3306 (MySQL) is open only from 10.0.0.0/24, closed from elsewhere
5. Port 123 (NTP, UDP) is open to all

Result:

```yaml
version: "1"

defaults:
  timeout: 2s
  protocol: tcp

endpoints:
  - name: ssh
    description: "SSH open to all"
    host: 192.168.1.100
    port: 22
    expect: open

  - name: web-ports
    description: "HTTP and HTTPS open to all"
    host: 192.168.1.100
    ports:
      - 80
      - 443
    expect: open

  - name: mysql-external
    description: "MySQL should be blocked from external"
    host: 192.168.1.100
    port: 3306
    expect: closed

  - name: ntp-udp
    description: "NTP over UDP open"
    host: 192.168.1.100
    port: 123
    protocol: udp
    expect: open

  - name: telnet-blocked
    description: "Telnet should be blocked (default DROP policy)"
    host: 192.168.1.100
    port: 23
    expect: closed

  - name: redis-blocked
    description: "Redis should be blocked (default DROP policy)"
    host: 192.168.1.100
    port: 6379
    expect: closed
```

---

## Validation and Troubleshooting

### Always Validate Before Running

```bash
fwprobe validate -c myconfig.yaml
```

### Common Validation Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `unsupported config version` | Missing or wrong version field | Set `version: "1"` |
| `name is required` | Endpoint missing name | Add unique `name` field |
| `cannot specify both 'host' and 'hosts'` | Both singular and plural used | Use only one |
| `must specify either 'host' or 'hosts'` | Neither provided | Add `host` or `hosts` |
| `cannot specify both 'port' and 'ports'` | Both singular and plural used | Use only one |
| `port X is out of range` | Port < 1 or > 65535 | Fix port number |
| `timeout must be at least 100ms` | Timeout too low | Use >= 100ms |
| `timeout must not exceed 30s` | Timeout too high | Use <= 30s |
| `duplicate endpoint name` | Two endpoints share a name | Make names unique |
| `failed to parse YAML` with unknown field | Typo in field name | Check spelling against schema |

### Tips for Config Generation

- Use descriptive `name` values derived from the service or rule purpose
- Group related ports using `ports` arrays (e.g., HTTP + HTTPS together)
- Test both what should be open AND what should be closed
- Include a few "negative tests" for ports that should be blocked by default policy
- For CIDR ranges, be cautious with large ranges (/16 max, produces 65,534 test targets per port)
- Set appropriate timeouts: use shorter timeouts (1s) for LAN, longer (5s) for WAN
- Use `protocol: udp` explicitly when testing UDP services (DNS, NTP, SNMP, etc.)
- Remember that UDP probing requires root/admin privileges for ICMP detection
