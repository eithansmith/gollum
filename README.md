# Gollum — Local LLM Chat (Go + WebSockets + Ollama)

A minimal, streaming chat UI backed by a Go HTTP/WebSocket server that proxies to a local [Ollama](https://ollama.ai) model. Type a prompt in the browser, receive token-streamed responses, and copy answers with a single click.

---

## Features

- **Streaming responses** from Ollama via server→client WebSocket chunks
- **Zero framework** Go server (stdlib HTTP + Gorilla WebSocket)
- **Pure HTML/CSS/JS** front end
- **Quality-of-life UX**: auto-resizing textarea, keyboard shortcuts, copy-to-clipboard, simple toast notifications
- **Single-binary dev loop**: `go run .` with a static `index.html`

---

## Architecture

```
Browser (index.html, WebSocket client)
   │
   ├─ ws://localhost:8080/ws
   ▼
Go Server (main.go)
   ├─ Serves index.html over HTTP
   ├─ Upgrades /ws to WebSocket
   └─ Proxies requests to Ollama with streaming
        │
        └─ POST http://localhost:11434/api/generate  (Stream: true)
```

### Message Flow

1. **User** types a prompt → client sends `{"type":"message","content":"..."}` over WebSocket.
2. **Server** responds with:
   - `{"type":"response_start"}`
   - `{"type":"response_chunk","content":"..."}` (0..n times)
   - `{"type":"response_end"}`
3. **Client** renders chunks, formats markdown-ish text, and enables the copy button once done.

---

## Prerequisites

- **Go** 1.21+
- **Ollama** installed and running locally (defaults to `http://localhost:11434`)
- An Ollama model pulled locally (e.g., `llama3.1`, `qwen2`, etc.)

```bash
# Install / start Ollama (see ollama.ai for platform-specific steps)
ollama serve &

# Pull a model (pick your favorite)
ollama pull llama3.1
```

> The server defaults to the model name `"llama2"` in `main.go`. Change it to the model you actually have (e.g., `"llama3.1"`).

---

## Getting Started

```bash
# 1) Clone your project (or place these two files in a folder):
#    - main.go
#    - index.html

# 2) Add Gorilla WebSocket
go get github.com/gorilla/websocket@latest

# 3) Run Go server
go run .

# 4) Open the app
open http://localhost:8080           # macOS
# or
xdg-open http://localhost:8080       # Linux
# or just paste in your browser on Windows
```

### Configuration

- **Server Port**: hard-coded to `:8080` in `main.go`
- **Ollama URL**: hard-coded to `http://localhost:11434/api/generate`
- **Model Name**: `Model` field in `queryOllamaStreaming` request (default: `llama2`)

You can promote these to env vars later (e.g., `OLLAMA_URL`, `OLLAMA_MODEL`, `PORT`).

---

## Keyboard Shortcuts

- **Enter**: send message
- **Shift+Enter**: newline
- **Ctrl/Cmd + K**: clear chat
- **Ctrl/Cmd + L**: focus input
- **↑ / ↓**: cycle through previous messages

---

## Project Structure

```
.
├── index.html      # Chat UI (HTML/CSS/JS)
└── main.go         # Go HTTP server + WebSocket bridge to Ollama
```

---

## Production Hardening (Suggestions)

- **CORS/Origin checks**: lock `websocket.Upgrader.CheckOrigin` to your trusted host.
- **TLS**: serve `https://` and `wss://` behind a reverse proxy (e.g., NGINX, Caddy).
- **Model/URL/port via env**: avoid hard-coded values.
- **Backpressure / rate limit**: bound message rates per connection/IP.
- **Logging**: structured logs; redact prompts if sensitive.
- **Error surfacing**: send a single, friendly error to client and close the ws.
- **Template safety**: parse & cache templates at startup; avoid parsing per request.
- **Static assets**: serve `index.html` and assets via `http.FileServer` or embed with `go:embed`.

---

## WebSocket Protocol (Server ↔ Client)

Server sends these message types:
- `response_start`
- `response_chunk` with `{ content: string }`
- `response_end`

Client sends:
- `message` with `{ content: string }`

---

## Development Tips

- If the page doesn’t connect: make sure **Ollama** is running and **Go server** is on port 8080.
- If you get **no output**: confirm the model name in `main.go` matches what you pulled in Ollama.
- If chunks arrive but render oddly, check `formatMessage()` in `index.html`.

---

## License

MIT (adjust as needed for your organization).
