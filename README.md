# Gemini freefall

## Description
This package is a very-very simple proxy which allows you to add fallback models to Gemini API calls.

You can setup one or more API keys (free or paid) and specify a callback path in case of an error/rate limit etc.

Sometimes it doesn't matter much which model is used, you only need some answer.

You can setup multiple fallback paths and the chosen one will be based on the API key you send to this service.

Config example:
```yaml
backend:
  - name: "backend1"
    key: "<YOUR API KEY HERE1>"
    models:
      - "gemini-2.5-pro"
      - "gemini-2.5-flash"
      - "gemini-2.5-flash-lite"
  - name: "backend2"
    key: "<YOUR API KEY HERE2>"
    models:
      - "gemini-2.5-flash"
      - "gemini-2.5-flash-lite"

config:
    port: 8080
    proxy:
      path1:
      - "backend1.gemini-2.5-pro"
      - "backend1.gemini-2.5-flash"
      - "backend1.gemini-2.5-flash-lite"
      path2:
      - "backend1.gemini-2.5-flash"
      - "backend2.gemini-2.5-flash"
```


This setup will use 2 API keys on the backend and name these configs backend1 and backend2.
Additionally you can specify a list of models for each backend.

The proxy config then will setup the fallback list, the name there `path1` and `path2` are the "api keys" for the proxy service.

Example call:
```bash
curl -H "x-goog-api-key: path1" http://127.0.0.1:8080/ -H "Content-type: application/json" -X POST \
  -d '{
    "contents": [
      {
        "parts": [
          {
            "text": "Explain how AI works in a few words"
          }
        ]
      }
    ]
  }'
```

This will automatically try the `gemini-2.5-pro` from the `backend1`, if it fails it will try the `gemini-2.5-flash` and then `gemini-2.5-flash-lite` from the same backend.

If all these fails, it will return an error.

If you setup `backend1` with a free tier key and `backend2` with a paid key, you can create a setup which will try to use the free tier first and then fallback to the paid key if the free tier fails.

## Usage

First you need to create the config file, for this you can copy the `config.yaml.example` to config.yaml and edit it.

Then you can run the service with:
```bash
go run cmd/server/main.go
```

Then you can point your desired service/client/etc. to this service and use the API key you specified in the config.
