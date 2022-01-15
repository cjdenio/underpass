# Underpass

Self-hosted [ngrok](https://ngrok.com) alternative.

## Installation (CLI)

```bash
brew install cjdenio/tap/underpass
```

Or, download a release from the [releases page](https://github.com/cjdenio/underpass/releases/latest).

## Usage with hosted server

There's a hosted instance running at https://underpass.clb.li.

```bash
underpass -p <port> -s <optional subdomain>
```

## Self-hosting

(more docs coming soon, possibly)

```bash
go run ./cmd/server --host <host here>
```

```bash
underpass -p <port> --host <host here> -s <optional subdomain>
```

## Known caveats

- No WebSocket support
