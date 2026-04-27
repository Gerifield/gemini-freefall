# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -v ./...        # build all packages
go build -v ./cmd/server # build the server binary
go test -v ./...         # run all tests
```

## Architecture

**gemini-freefall** is an HTTP proxy that implements ordered failover across multiple API keys and models. It supports native Gemini backends and any OpenAI-compatible backend (OpenAI, Anthropic, Gemini-via-OpenAI, custom).

### Request flow

1. Client sends a request with `x-goog-api-key: <path-name>` (e.g. `free-first`)
2. `Logic.handler()` looks up that path name in the config to get an ordered list of `backend.model` targets
3. The body is buffered once so it can be replayed across retries
4. Each `backend.model` is tried in order — first HTTP 200 wins; on non-200 the next target is attempted
5. If all targets fail, the proxy returns HTTP 403

### Package layout

```
cmd/server/main.go          — entry point: parses -openai flag, loads config.yaml, wires Logic, starts server
internal/proxy/config.go    — Config/Backend structs, LoadConfig(), validation, sentinel errors
internal/proxy/proxy.go     — Logic handler, request forwarding, URL construction, ListenAndServe()
```

### Configuration

Config is loaded from `config.yaml` in the working directory (see `config.yaml.example`). No environment variables are used.

```yaml
# Optional: override base URLs for any built-in provider type
base_urls:
  openai:    "https://api.openai.com/v1"         # default
  anthropic: "https://api.anthropic.com/v1"      # default
  gemini_openai: "https://generativelanguage.googleapis.com/v1beta/openai"  # default

backend:
  - name: free-key
    type: gemini              # default type; omit for Gemini backends
    key: "<gemini-api-key>"
    models: [gemini-2.5-pro, gemini-2.5-flash]

  - name: openai-backend
    type: openai              # requires -openai flag
    key: "sk-proj-..."
    models: [gpt-4o]

  - name: custom-local
    type: custom_openai       # requires -openai flag; base_url required
    base_url: "http://localhost:11434/v1"
    key: "ollama"
    models: [qwen2.5:14b]

config:
  port: 8080
  proxy:
    free-first:
      - free-key.gemini-2.5-pro    # tried first
      - free-key.gemini-2.5-flash  # fallback
```

### Backend types

| Type | Upstream | Auth header | Flag |
|---|---|---|---|
| `gemini` | `generativelanguage.googleapis.com/v1beta` | `x-goog-api-key` | *(none)* |
| `openai` | `api.openai.com/v1` | `Authorization: Bearer` | `-openai` |
| `anthropic` | `api.anthropic.com/v1` | `x-api-key` | `-openai` |
| `gemini_openai` | `generativelanguage.googleapis.com/v1beta/openai` | `Authorization: Bearer` | `-openai` |
| `custom_openai` | value of `base_url` field | `Authorization: Bearer` | `-openai` |

### Key implementation notes

- `slog` is used for all structured logging
- Sentinel errors (`ErrInvalidPort`, `ErrNoBackends`, `ErrInvalidBackend`, `ErrInvalidProxy`, `ErrNoModels`) are defined in `config.go`
- `LoadConfig` filters backends by mode: `-openai` drops `gemini` backends; default mode drops everything else. Proxy paths that lose all valid targets are also removed.
- In OpenAI mode the incoming URL path is forwarded to the backend with a leading `/v1` stripped once (to avoid doubling the version prefix when the base URL already ends in `/v1`)
- Tests use `testify` and run in parallel; they load `config.yaml.example` and `config_mixed_test.yaml` as fixtures
- Known gaps (TODOs in `proxy.go`): streaming endpoint (`generateContentStream` / SSE) not yet supported; HTTP client lacks a Dialer timeout
