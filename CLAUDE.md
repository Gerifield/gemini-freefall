# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -v ./...       # build all packages
go build -v ./cmd/server # build the server binary
go test -v ./...        # run all tests
```

## Architecture

**gemini-freefall** is an HTTP proxy for Google Gemini API calls that implements ordered failover across multiple API keys and models.

### Request flow

1. Client sends a request with `x-goog-api-key: <path-name>` (e.g. `path1`)
2. `Logic.handler()` looks up that path name in the config to get an ordered list of `backend.model` targets
3. The body is buffered once so it can be replayed across retries
4. Each `backend.model` is tried in order — first HTTP 200 wins; on failure the next is attempted
5. If all targets fail, the proxy returns HTTP 403

### Package layout

```
cmd/server/main.go          — entry point: loads config.yaml, wires Logic, starts server
internal/proxy/config.go    — Config/Backend structs, LoadConfig(), validation
internal/proxy/proxy.go     — Logic handler, request forwarding, ListenAndServe()
```

### Configuration

Config is loaded from `config.yaml` in the working directory (see `config.yaml.example`). No environment variables are used.

```yaml
backend:
  - name: backend1
    key: "<gemini-api-key>"
    models: [gemini-2.5-pro, gemini-2.5-flash]

config:
  port: 8080
  proxy:
    path1:
      - backend1.gemini-2.5-pro    # tried first
      - backend1.gemini-2.5-flash  # fallback
```

### Key implementation notes

- `slog` is used for all structured logging
- Sentinel errors (`ErrInvalidPort`, `ErrNoBackends`, …) are defined in `config.go`
- Tests use `testify` and run in parallel; they load `config.yaml.example` as fixture
- Known gaps (TODOs in `proxy.go`): streaming endpoint (`generateContentStream`) not yet supported; HTTP client lacks a Dialer timeout
