# portfwd

Cross-platform TCP/UDP port forwarder with saved configurations, traffic statistics, access control, rate limiting, and optional TLS termination.

Forward local ports to remote destinations without SSH -- as a single static binary.

## Install

### Binary download

Download the latest release for your platform from the [releases page](https://github.com/jmsperu/portfwd/releases).

### go install

```sh
go install github.com/jmsperu/portfwd@latest
```

### Build from source

```sh
git clone https://github.com/jmsperu/portfwd.git
cd portfwd
make build
```

## Quick start

```sh
# Forward TCP port 8080 to a remote host
portfwd 8080:remote.host:80

# Forward UDP port
portfwd -u 5060:sip.server:5060

# Save, start, and manage forwards
portfwd add web 8080:remote.host:80
portfwd up web
portfwd stats
```

## Commands

### Inline forward

```sh
portfwd <local_port:remote_host:remote_port>
```

Starts an ad-hoc forward immediately. Press Ctrl+C to stop.

| Flag | Description |
|------|-------------|
| `-u, --udp` | Use UDP instead of TCP |
| `--tls-cert` | TLS certificate file for TLS termination |
| `--tls-key` | TLS key file for TLS termination |
| `--allow` | Allowed source CIDRs (comma-separated) |
| `--deny` | Denied source CIDRs (comma-separated) |
| `--rate-limit` | Rate limit in bytes/sec per connection (0 = unlimited) |

### add

Save a named forward configuration for later use.

```sh
portfwd add web 8080:remote.host:80
portfwd add sip 5060:sip.server:5060 -u
portfwd add secure 443:backend:8443 --tls-cert cert.pem --tls-key key.pem
portfwd add restricted 8080:internal:80 --allow 10.0.0.0/8 --deny 192.168.1.0/24
```

### list (ls)

List all saved port forwards in a table.

```sh
portfwd list
portfwd ls
```

### up

Start one or more saved forwards.

```sh
portfwd up web           # start a single forward
portfwd up web sip       # start multiple forwards
portfwd up --all         # start all saved forwards
```

| Flag | Description |
|------|-------------|
| `-a, --all` | Start all saved forwards |

### remove (rm)

Remove a saved forward by name.

```sh
portfwd remove web
portfwd rm web
```

### stats

Show traffic statistics for recent forwarding sessions.

```sh
portfwd stats
```

Displays connections, bytes in/out, and last active time for each forward.

## Configuration

Saved forwards are stored in `~/.portfwd.yml`:

```yaml
forwards:
  web:
    protocol: tcp
    listen_port: 8080
    remote_host: remote.host
    remote_port: 80
  sip:
    protocol: udp
    listen_port: 5060
    remote_host: sip.server
    remote_port: 5060
    rate_limit: 1048576
  secure:
    protocol: tcp
    listen_port: 443
    remote_host: backend
    remote_port: 8443
    tls_cert: /path/to/cert.pem
    tls_key: /path/to/key.pem
    allow_cidrs:
      - 10.0.0.0/8
    deny_cidrs:
      - 192.168.1.0/24
```

## License

MIT
