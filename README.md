# gemini-freefall

> Never hit a rate limit again. A zero-dependency failover proxy for the Gemini API.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/github/license/Gerifield/gemini-freefall)

**gemini-freefall** sits between your app and the Gemini API. It accepts standard Gemini requests and transparently retries them across a prioritized list of API keys and models — so a rate limit or quota exhaustion on one key never reaches your application.

---

## How it works

```
Your app  ──►  gemini-freefall  ──►  gemini-2.5-pro   (try 1 — rate limited)
                                ──►  gemini-2.5-flash  (try 2 — success ✓)
```

1. Your client sends a normal Gemini API request, using a **proxy path name** as the `x-goog-api-key` (e.g. `free-tier`).
2. The proxy looks up the ordered fallback chain for that path.
3. Each `backend.model` target is tried in order — the first HTTP 200 is returned immediately.
4. On failure, the next target is tried with the buffered request body replayed.
5. If all targets fail, the proxy returns `403`.

Your app never changes — just swap the endpoint and API key.

---

## Quick start

```bash
# 1. Clone and build
git clone https://github.com/gergelyradics/gemini-freefall
cd gemini-freefall
go build -o freefall ./cmd/server

# 2. Configure
cp config.yaml.example config.yaml
$EDITOR config.yaml

# 3. Run
./freefall

# 4. Run in OpenAI-compatible mode
./freefall -openai
```

---

## Configuration

```yaml
# config.yaml
backend:
  - name: free-key
    type: gemini            # (Default) Native Gemini API
    key: "AIza..."          # free-tier Gemini API key
    models:
      - gemini-2.5-pro
      - gemini-2.5-flash
      - gemini-2.5-flash-lite

  - name: paid-key
    type: gemini
    key: "AIza..."          # paid Gemini API key
    models:
      - gemini-2.5-flash

  - name: openai-backend
    type: openai            # Uses default OpenAI endpoint (requires -openai flag)
    key: "sk-proj..."
    models:
      - gpt-4o

  - name: anthropic-backend
    type: anthropic         # Uses Anthropic's OpenAI-compatible endpoint (requires -openai flag)
    key: "sk-ant..."
    models:
      - claude-3-5-sonnet-20241022

  - name: local-qwen
    type: custom_openai
    base_url: "http://192.168.1.50:8000/v1"
    key: "sk-dummy"
    models:
      - qwen-14b

config:
  port: 8080
  proxy:
    # "free-first" path: try free tier top models, fall back to paid
    free-first:
      - free-key.gemini-2.5-pro
      - free-key.gemini-2.5-flash
      - paid-key.gemini-2.5-flash

    # "fast" path: skip straight to flash
    fast:
      - free-key.gemini-2.5-flash
      - free-key.gemini-2.5-flash-lite
```

Each entry under `proxy` is a named **path** — use the path name as your `x-goog-api-key`. Targets are `<backend-name>.<model-name>` and are tried left-to-right.

### The `-openai` Flag

Because `gemini-freefall` operates as a zero-memory envelope router, it **does not** parse or translate JSON payloads.
If a client sends an OpenAI-formatted payload, it will fail if routed to a native Gemini backend, and vice versa.

To prevent misconfigurations:
- Running without flags allows **only** `type: gemini` backends. Non-Gemini targets are dropped from routing at startup.
- Running with `-openai` allows **only** OpenAI-compatible backends (`openai`, `anthropic`, `custom_openai`, `gemini_openai`). Native Gemini targets are dropped.

---

## Usage

Point any Gemini-compatible client at `http://localhost:8080` and use a proxy path name as the API key:

```bash
curl http://localhost:8080/ \
  -H "x-goog-api-key: free-first" \
  -H "Content-Type: application/json" \
  -X POST \
  -d '{
    "contents": [{
      "parts": [{"text": "Explain how AI works in a few words"}]
    }]
  }'
```

The proxy is a drop-in for `https://generativelanguage.googleapis.com/v1beta/models/<model>:generateContent` — no SDK changes needed, just change the base URL and API key.

---

## Common patterns

**Free-tier maximizer** — exhaust free quota before touching paid keys:
```yaml
proxy:
  default:
    - free1.gemini-2.5-pro
    - free2.gemini-2.5-pro
    - paid.gemini-2.5-pro
```

**Model degradation** — prefer quality, tolerate speed:
```yaml
proxy:
  quality:
    - key1.gemini-2.5-pro
    - key1.gemini-2.5-flash
    - key1.gemini-2.5-flash-lite
```

**Multi-tenant** — different SLAs per consumer:
```yaml
proxy:
  premium: [paid.gemini-2.5-pro]
  standard: [free.gemini-2.5-flash, free.gemini-2.5-flash-lite]
```
