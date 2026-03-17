# AGENTS.md

## Scope
- This repo has two public entrypoints: `cmd/router-center/main.go` starts the router center plus HTTP monitor; `cmd/router-client/main.go` starts a router client and then blocks in `AwaitSystemClose()`.
- Prefer editing source under `cmd/`, `internal/`, `VirtualRouterClient/`, and `VirtualRouterServer/`. Treat `release/` as generated output from `一键打包全平台.ps1`.

## Architecture map
- The top-level `VirtualRouterServer/` and `VirtualRouterClient/` packages are thin public wrappers over `internal/VirtualRouterServer` and `internal/VirtualRouterClient`; change internals first unless the public API must move too.
- The router center is a TCP switchboard in `internal/VirtualRouterServer/server.go`: clients send framed `core.RouteMessage` packets, heartbeats register/update sessions, and RPC/data messages are forwarded by `ToRouteId`.
- Sessions live in `internal/VirtualRouterServer/session_manager.go`; inactive nodes time out after 30s and removal broadcasts `RouteMessageTypeRemoveRouteNode` to remaining clients.
- The client keeps a singleton route table in `internal/VirtualRouterClient/route_table.go`; route changes invalidate cached direct RPC clients.
- Wire format lives in `internal/core/framing.go` and `internal/core/route_message.go`: every TCP message is `4-byte big-endian length + payload`, and `DecodeRouteMessagePayload` intentionally tolerates an extra embedded length prefix for compatibility.

## RPC model
- RPC stubs must be registered before client start: `internal/VirtualRouterClient/client.go` calls `rpc.ServerStubManagerInstance().EnsureInitialized()` inside `Start()`, so missing registration causes a panic.
- Prefer `VirtualRouterClient.RegisterRpcFunc(...)` over manual stub registration when adding callable methods; it auto-generates `RpcStubMetadata` used by the monitor UI (`internal/rpc/orm.go`).
- `rpcMode` in `neko233-router-client.json` controls transport: `relay` routes calls through the center (`internal/rpc/relay_client.go` / `relay_handlers.go`), while `direct` also starts a local TCP stub server on `LocalRpcPort` (`internal/rpc/stub_server.go`).
- Heartbeats include `RpcServerInfo{Host, Port, Stubs}` (`internal/VirtualRouterClient/client.go`), so changing stub metadata affects routing, debug UI, and direct-call discovery together.

## HTTP monitor and auth
- The monitor server in `internal/VirtualRouterServer/http_server.go` serves JSON APIs and embedded static assets from `internal/VirtualRouterServer/static/` via `//go:embed` in `static_assets.go`.
- `/` redirects to `/login.html` unless a valid JWT is present; auth accepts either `Authorization: Bearer ...` or the `virtual-router-admin-token` cookie.
- Admin password changes are persisted back to `neko233-router-server.json` by `handleUpdateAdminPassword`, so tests use a temp working directory when exercising settings writes.
- `InstallProcessLogCapture(800)` in `cmd/router-center/main.go` enables in-memory log capture plus rotating files under `logs/`; the dashboard reads from that buffer.

## Developer workflows
- Run the main test suite with `go test ./...` or `./test-all.ps1`. Coverage is concentrated in `internal/core`, `internal/rpc`, and `internal/VirtualRouterServer`.
- Build the server binary locally with `go build ./cmd/router-center`; build the client binary with `go build ./cmd/router-client`.
- Release automation is PowerShell-first: `release.ps1` requires a clean git tree, runs `go test ./...`, bumps `version.txt`, creates a git tag, and pushes `main` + tags.
- Cross-platform artifacts are produced by `一键打包全平台.ps1`; it emits only the server binary/config into `release/<os>-<arch>/` and copies `devops_for_virtual_router_server.sh` for Unix targets.
- SSH deployment is `go run _ssh/deploy.go`, configured by `_ssh/config_for_ssh_deploy.yaml` plus optional `_ssh/config_for_ssh_deploy.local.yaml` overrides.

## Repo-specific conventions
- Config readers auto-create missing JSON files and then return an error telling the operator to fill them in (`internal/config/config.go`); do not “fix” this into silent defaults.
- The project uses `slog` with Chinese operator-facing messages; preserve that tone when touching logs and HTTP responses.
- Many tests exercise internals directly (for example `internal/VirtualRouterServer/forward_distributed_test.go` uses `net.Pipe()` to verify framed forwarding). Follow that style for protocol-level changes.
- When adding monitor functionality, update both the Go handlers in `http_server.go` and the embedded frontend in `internal/VirtualRouterServer/static/app.js`.

