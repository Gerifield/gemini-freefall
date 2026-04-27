# gemini-freefall

> Never hit a rate limit again. A zero-dependency failover proxy for Gemini — and any OpenAI-compatible LLM.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/github/license/Gerifield/gemini-freefall)

**gemini-freefall** sits between your app and any LLM API. It accepts standard Gemini or OpenAI-compatible requests and transparently retries them across a prioritized list of API keys and models — so a rate limit or quota exhaustion on one key never reaches your application.

---

## How it works

```
Your app  ──►  gemini-freefall  ──►  backend1 / gemini-2.5-pro    (try 1 — rate limited)
                                ──►  backend1 / gemini-2.5-flash   (try 2 — rate limited)
                                ──►  backend2 / gemini-2.5-flash   (try 3 — success ✓)
```

1. Your client sends a normal API request, using a **proxy path name** as the `x-goog-api-key` header (e.g. `free-first`).
2. The proxy looks up the ordered fallback chain for that path in `config.yaml`.
3. Each `backend.model` target is tried in order — the first HTTP 200 is returned to the caller immediately.
4. On any non-200 response the request body is replayed against the next target.
5. If all targets fail, the proxy returns `403`.

Your app never changes — just swap the base URL and API key.

---

## Quick start

```bash
# 1. Clone and build
git clone https://github.com/Gerifield/gemini-freefall
cd gemini-freefall
go build -o freefall ./cmd/server

# 2. Configure
cp config.yaml.example config.yaml
$EDITOR config.yaml

# 3. Run in native Gemini mode
./freefall

# 4. Run in OpenAI-compatible mode
./freefall -openai
```

---

## Configuration

Config is loaded from `config.yaml` in the working directory at startup. Below is a full annotated reference.

```yaml
# Optional: override built-in base URLs (useful for LiteLLM, corporate proxies, etc.)
base_urls:
  openai:        "https://api.openai.com/v1"                           # default
  anthropic:     "https://api.anthropic.com/v1"                        # default
  gemini_openai: "https://generativelanguage.googleapis.com/v1beta/openai"  # default

backend:
  # ── Native Gemini API ──────────────────────────────────────────────
  # Used without the -openai flag. Type defaults to "gemini" if omitted.
  - name: free-key
    type: gemini
    key: "AIza..."
    models:
      - gemini-2.5-pro
      - gemini-2.5-flash
      - gemini-2.5-flash-lite

  - name: paid-key
    type: gemini
    key: "AIza..."
    models:
      - gemini-2.5-pro
      - gemini-2.5-flash

  # ── OpenAI-compatible backends ─────────────────────────────────────
  # All of the following require the -openai flag.

  - name: openai-backend
    type: openai                # routes to api.openai.com/v1 by default
    key: "sk-proj-..."
    models:
      - gpt-4o
      - gpt-4o-mini

  - name: anthropic-backend
    type: anthropic             # routes to api.anthropic.com/v1 by default
    key: "sk-ant-..."
    models:
      - claude-opus-4-5
      - claude-sonnet-4-5

  - name: gemini-compat
    type: gemini_openai         # Gemini via its OpenAI-compatible endpoint
    key: "AIza..."
    models:
      - gemini-2.5-pro
      - gemini-2.5-flash

  - name: local-ollama
    type: custom_openai         # any OpenAI-compatible endpoint
    base_url: "http://localhost:11434/v1"
    key: "ollama"               # Ollama accepts any non-empty value
    models:
      - qwen2.5:14b
      - llama3.2

config:
  port: 8080
  proxy:
    # Each key is a path name — use it as the x-goog-api-key in your client.
    # Targets are tried left-to-right; first 200 wins.

    free-first:
      - free-key.gemini-2.5-pro
      - free-key.gemini-2.5-flash
      - paid-key.gemini-2.5-pro     # final paid fallback

    fast:
      - free-key.gemini-2.5-flash
      - free-key.gemini-2.5-flash-lite
```

### Backend types

| Type | Upstream | Auth header sent | Flag required |
|---|---|---|---|
| `gemini` | `generativelanguage.googleapis.com/v1beta` | `x-goog-api-key` | *(none)* |
| `openai` | `api.openai.com/v1` | `Authorization: Bearer` | `-openai` |
| `anthropic` | `api.anthropic.com/v1` | `x-api-key` | `-openai` |
| `gemini_openai` | `generativelanguage.googleapis.com/v1beta/openai` | `Authorization: Bearer` | `-openai` |
| `custom_openai` | value of `base_url` field | `Authorization: Bearer` | `-openai` |

### The `-openai` flag

The proxy is a zero-memory envelope router — it never parses or translates JSON bodies. A Gemini-format payload will fail against an OpenAI endpoint and vice versa. The flag enforces the correct backend set at startup:

- **Without `-openai`** — only `gemini` backends are active; all other types are silently dropped at startup.
- **With `-openai`** — only OpenAI-compatible backends (`openai`, `anthropic`, `gemini_openai`, `custom_openai`) are active; `gemini` backends are dropped.

Proxy paths that end up with no valid targets after filtering are also removed.

### Overriding base URLs

Use `base_urls` to redirect a built-in provider type to a different host without changing each backend entry individually:

```yaml
# Route all "openai" and "anthropic" traffic through a local LiteLLM gateway
base_urls:
  openai:    "http://litellm.internal:4000/v1"
  anthropic: "http://litellm.internal:4000/v1"
```

---

## Usage

### Native Gemini mode

Use a proxy path name as `x-goog-api-key`. The body is a standard Gemini `generateContent` request:

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

Python (`requests`):

```python
import requests

response = requests.post(
    "http://localhost:8080/",
    headers={
        "x-goog-api-key": "free-first",
        "Content-Type": "application/json",
    },
    json={
        "contents": [{"parts": [{"text": "Explain how AI works in a few words"}]}]
    },
)
print(response.json()["candidates"][0]["content"]["parts"][0]["text"])
```

### OpenAI-compatible mode (`-openai`)

The proxy forwards the incoming URL path to the backend, stripping a leading `/v1` if present to avoid doubling the version prefix. Use a proxy path name as `x-goog-api-key`:

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "x-goog-api-key: openai-path" \
  -H "Content-Type: application/json" \
  -X POST \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Explain how AI works in a few words"}]
  }'
```

Python — OpenAI SDK:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="unused",                           # ignored; routing uses x-goog-api-key
    default_headers={"x-goog-api-key": "openai-path"},
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Explain how AI works in a few words"}],
)
print(response.choices[0].message.content)
```

TypeScript / Node.js — OpenAI SDK:

```typescript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:8080/v1",
  apiKey: "unused",
  defaultHeaders: { "x-goog-api-key": "openai-path" },
});

const response = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role: "user", content: "Explain how AI works in a few words" }],
});
console.log(response.choices[0].message.content);
```

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

**Model degradation** — prefer quality, accept lower latency on failure:
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
  premium:  [paid.gemini-2.5-pro]
  standard: [free.gemini-2.5-flash, free.gemini-2.5-flash-lite]
```

**Cross-provider fallback** (OpenAI mode) — fall back across providers:
```yaml
proxy:
  resilient:
    - openai-backend.gpt-4o
    - anthropic-backend.claude-sonnet-4-5
    - local-ollama.qwen2.5:14b
```

**LiteLLM gateway** — route all traffic through a self-hosted gateway with per-provider fallback:
```yaml
base_urls:
  openai:    "http://litellm.internal:4000/v1"
  anthropic: "http://litellm.internal:4000/v1"

backend:
  - name: gpt
    type: openai
    key: "sk-proj-..."
    models: [gpt-4o]
  - name: claude
    type: anthropic
    key: "sk-ant-..."
    models: [claude-sonnet-4-5]

config:
  port: 8080
  proxy:
    default:
      - gpt.gpt-4o
      - claude.claude-sonnet-4-5
```

---

## Limitations

- **No streaming**: only non-streaming responses are proxied. `generateContentStream` (Gemini) and SSE responses (OpenAI) are not yet supported.
- **No load balancing**: targets are tried strictly in the configured order; there is no round-robin or least-load selection.
- **No dial timeout**: the HTTP client does not set a Dialer timeout; a hung backend will block until the OS-level connection timeout fires.
- **Restart to reload**: config is read once at startup; changes require a process restart.

---

## Building

```bash
go build -v ./...                   # build all packages
go build -o freefall ./cmd/server   # build the server binary
go test -v ./...                    # run tests
```
