# Gemini Freefall

**Gemini Freefall** is a lightweight, simple-to-use proxy for the Google Gemini API. It enables automatic fallback mechanisms between different models and API keys, ensuring that your application remains highly available even if an API key hits rate limits or a specific model encounters an error.

Sometimes it doesn't matter much *which* model is used, as long as you get an answer. Gemini Freefall lets you prioritize your preferred models (like Pro) and transparently fail over to faster or smaller models (like Flash or Flash-Lite) or even secondary API keys (like switching from a free-tier key to a paid-tier key).

## Features

- **Automatic Failover:** Seamlessly fall back to secondary models or API keys when an error occurs.
- **Multiple API Key Support:** Configure as many API keys (backends) as you need.
- **Flexible Routing:** Create multiple "proxy paths" (used as virtual API keys) to define different fallback sequences.
- **Cost Optimization:** Prioritize free-tier API keys, falling back to paid-tier keys only when necessary.

## Prerequisites

- Go 1.24 or later

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/gemini-freefall.git
   cd gemini-freefall
   ```

2. Copy the example configuration to create your own:
   ```bash
   cp config.yaml.example config.yaml
   ```

## Configuration

Edit `config.yaml` to define your backends and proxy routing.

- **`backend`**: Defines the physical API keys and the models each key has access to.
- **`proxy`**: Defines the virtual API keys that your clients will use. Each virtual key (`path1`, `path2`, etc.) maps to a priority-ordered list of `backend.model` combinations.

```yaml
backend:
  - name: "backend1"
    key: "<YOUR API KEY HERE1>"
    models:
      - "gemini-3.1-pro"
      - "gemini-3.1-flash"
      - "gemini-3.1-flash-lite"
  - name: "backend2"
    key: "<YOUR API KEY HERE2>"
    models:
      - "gemini-3.1-flash"
      - "gemini-3.1-flash-lite"

config:
  port: 8080
  proxy:
    path1:
      - "backend1.gemini-3.1-pro"
      - "backend1.gemini-3.1-flash"
      - "backend1.gemini-3.1-flash-lite"
    path2:
      - "backend1.gemini-3.1-flash"
      - "backend2.gemini-3.1-flash"
```

In this example:
- Clients sending the `x-goog-api-key: path1` header will first attempt to use `backend1`'s `gemini-3.1-pro`. If that fails, the proxy will automatically try `backend1`'s `gemini-3.1-flash`, and finally `backend1`'s `gemini-3.1-flash-lite`.
- Clients sending the `x-goog-api-key: path2` header will first attempt `backend1`'s `gemini-3.1-flash`, and if that fails, fail over to `backend2`'s `gemini-3.1-flash`.

## Usage

Start the proxy server:

```bash
go run cmd/server/main.go
```

The server will start on the port specified in your `config.yaml` (default is `8080`).

You can now point your desired service, client, or curl requests to the proxy. Instead of using your real Google API key, use the proxy path name (e.g., `path1`) in the `x-goog-api-key` header.

### Example Request

```bash
curl -H "x-goog-api-key: path1" \
     -H "Content-type: application/json" \
     -X POST http://127.0.0.1:8080/ \
     -d '{
       "contents": [
         {
           "role": "user",
           "parts": [
             {
               "text": "Explain how AI works in a few words."
             }
           ]
         }
       ]
     }'
```

### How it Works
1. The proxy receives the incoming request and reads the virtual API key (`path1`) from the `x-goog-api-key` header.
2. It looks up the fallback sequence defined in `config.yaml` for `path1`.
3. It rewrites the request to target the actual Google Gemini API (`https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent`).
4. It swaps the virtual key for the real API key configured for the current backend.
5. If the Gemini API returns a successful response (HTTP 200), the proxy forwards it back to the client.
6. If the Gemini API returns a non-200 status (e.g., rate limit exceeded, server error), the proxy intercepts the failure and retries the request using the next `backend.model` in the list.
7. If all configured fallbacks fail, an error is returned to the client.