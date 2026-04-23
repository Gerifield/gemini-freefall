# GEMINI.md

This file provides guidance to Gemini Code Assist (and compatible AI coding tools) when working with code in this repository.

## Commands

```bash
go build -v ./...        # build all packages
go build -v ./cmd/server # build the server binary
go test -v ./...         # run all tests
```

## Architecture

**gemini-freefall** is an HTTP proxy for Google Gemini API calls that implements ordered failover across multiple API keys and models.

### Request flow

1. Client sends a request with `x-goog-api-key: <path-name>` (e.g. `path1`)
2. `Logic.handler()` resolves that path name to an ordered list of `backend.model` targets from config
3. The request body is read once into a buffer so it can be replayed across retries
4. Each `backend.model` is tried in order — first HTTP 200 wins; non-200 responses cause the next target to be tried
5. If all targets fail, the proxy returns HTTP 403

### Package layout

```
cmd/server/main.go          — entry point: loads config.yaml, wires Logic, starts server
internal/proxy/config.go    — Config/Backend structs, LoadConfig(), validation, sentinel errors
internal/proxy/proxy.go     — Logic handler, request forwarding to Gemini API, ListenAndServe()
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

Proxy targets use the format `<backend-name>.<model-name>`. Config validation enforces that all referenced backends and models exist.

### Key implementation notes

- All logging uses `log/slog` (structured, key-value pairs)
- Sentinel errors (`ErrInvalidPort`, `ErrNoBackends`, `ErrInvalidBackend`, `ErrInvalidProxy`, `ErrNoModels`) are defined in `config.go`
- Tests use `testify` with `t.Parallel()` and load `config.yaml.example` as a fixture
- Known gaps (TODOs in `proxy.go`): streaming endpoint (`generateContentStream`) not yet supported; HTTP client lacks a Dialer timeout
- The upstream target URL is hardcoded to `generativelanguage.googleapis.com/v1beta/models/{model}:generateContent`
