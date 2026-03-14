# yqctl

CLI for the [YoorQuezt MEV](https://github.com/yoorquezt-labs) gateway. Submit bundles, protect transactions, query auctions, and stream real-time events over WebSocket JSON-RPC 2.0.

## Install

### Homebrew

```bash
brew install yoorquezt-labs/tap/yqctl
```

### Shell script

```bash
curl -fsSL https://raw.githubusercontent.com/yoorquezt-labs/yqctl/master/scripts/install.sh | sh
```

### Go install

```bash
go install github.com/yoorquezt-labs/yqctl@latest
```

### From source

```bash
git clone https://github.com/yoorquezt-labs/yqctl.git
cd yqctl
make install
```

## Usage

```bash
yqctl [flags] <command> [args...]
```

### Flags

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `-gw` | `YQMEV_GATEWAY` | `ws://localhost:9099/ws` | Gateway WebSocket URL |
| `-key` | `YQMEV_API_KEY` | | API key (bearer token) |

### Commands

| Command | Description |
|---------|-------------|
| `health` | Gateway health check |
| `auction` | Show current auction pool |
| `bundle submit` | Submit a bundle (`--bid <wei> --tx '<json>'`) |
| `bundle get <id>` | Get bundle by ID |
| `protect submit` | Submit protected tx (`--from --to --payload`) |
| `protect status <id>` | Get protection status by tx ID |
| `intent submit` | Submit an intent (`--type --chain`) |
| `intent get <id>` | Get intent by ID |
| `relay register` | Register a relay (`-url <endpoint>`) |
| `relay list` | List registered relays |
| `relay stats` | Relay marketplace statistics |
| `blocks` | List recent blocks |
| `bundles` | List stored bundles |
| `orderflow summary` | Orderflow statistics |
| `watch <topic>` | Stream events (auction, blocks, mempool, protect, intents) |

### Examples

```bash
# Health check
yqctl health

# View auction pool
yqctl auction

# Submit a bundle
yqctl bundle submit --bid 1000000 --tx '{"tx_id":"t1","chain":"ethereum","payload":"0xdead"}'

# Get bundle status
yqctl bundle get ABC123

# Protect a transaction
yqctl protect submit --from 0xabc --to 0xdef --payload 0x1234

# Stream auction events
yqctl watch auction

# With custom gateway
yqctl -gw ws://gateway.example.com:9099/ws -key mytoken health
```

## SDK

The `pkg/client` package provides a Go SDK for programmatic access:

```go
import "github.com/yoorquezt-labs/yqctl/pkg/client"

c, err := client.Dial(client.Config{
    GatewayURL: "ws://localhost:9099/ws",
    APIKey:     "my-api-key",
})
defer c.Close()

// Submit a bundle
res, err := c.SendBundle(ctx, types.BundleMessage{...})

// Subscribe to events
subID, ch, err := c.Subscribe(ctx, "auction")
for event := range ch {
    fmt.Println(string(event))
}
```

## Development

```bash
make build      # Build binary to bin/yqctl
make test       # Run tests
make lint       # Run linter
make snapshot   # GoReleaser snapshot build
```

### Release

Tag a version and push — GitHub Actions runs GoReleaser automatically:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## License

MIT
