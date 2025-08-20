package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// OllamaRequest models the JSON body sent to Ollama's /api/generate endpoint.
// Stream=true instructs Ollama to return a sequence of JSON objects (one per chunk).
// See https://github.com/ollama/ollama for details.
type OllamaRequest struct {
	Model  string `json:"model"`  // Name of the local model to use (must be pulled in Ollama)
	Prompt string `json:"prompt"` // Full user prompt to generate on
	Stream bool   `json:"stream"` // Enable server-side streaming of tokens/chunks
}

// OllamaStreamResponse models *one* streamed JSON object emitted by Ollama.
// Ollama writes a series of objects where `response` carries the next text
// fragment and `done=true` marks the final object.
type OllamaStreamResponse struct {
	Response string `json:"response"` // Next chunk of generated text
	Done     bool   `json:"done"`     // True on the final message
}

// upgrader handles the HTTP→WebSocket upgrade for /ws.
// NOTE: CheckOrigin currently allows all origins which is fine for localhost dev
// but should be restricted in production.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: restrict origin in production
	},
}

func main() {
	// Serve the chat interface at GET / by parsing index.html as a Go template.
	// (Currently no dynamic data is injected; template parsing could be done once
	// at startup and cached for performance.)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("index.html"))
		_ = tmpl.Execute(w, nil)
	})

	// WebSocket endpoint used by the front-end for streaming responses.
	http.HandleFunc("/ws", handleWebSocket)

	log.Println("Starting Gollum server on :8080")
	log.Println("Open http://localhost:8080 in your browser")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleWebSocket upgrades the incoming HTTP connection and drives a simple
// request/stream-response protocol with the browser. The browser sends a JSON
// object like: { "type": "message", "content": "prompt..." }. We then call
// Ollama in streaming mode and forward chunks to the client as they arrive.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		switch msg["type"] {
		case "message":
			userMessage, _ := msg["content"].(string)

			// Announce the start of a new assistant response so the client can show
			// a typing indicator and allocate a placeholder element.
			_ = conn.WriteJSON(map[string]string{"type": "response_start"})

			// Stream from Ollama and forward each chunk to the client. Any error here
			// is logged and surfaced as a user-friendly message.
			if err := queryOllamaStreaming(userMessage, func(chunk string, done bool) {
				if !done {
					_ = conn.WriteJSON(map[string]string{
						"type":    "response_chunk",
						"content": chunk,
					})
					return
				}
				_ = conn.WriteJSON(map[string]string{"type": "response_end"})
			}); err != nil {
				log.Printf("Ollama query error: %v", err)
				_ = conn.WriteJSON(map[string]string{
					"type":    "response_chunk",
					"content": "Sorry, I encountered an error processing your request.",
				})
				_ = conn.WriteJSON(map[string]string{"type": "response_end"})
			}
		default:
			// Ignore unknown message types to keep the protocol forward-compatible.
		}
	}
}

// queryOllamaStreaming calls the local Ollama server with Stream=true and
// decodes the NDJSON-like stream. For each decoded object it invokes `callback`
// with the text chunk and whether the stream is done.
func queryOllamaStreaming(prompt string, callback func(string, bool)) error {
	reqBody := OllamaRequest{
		Model:  "llama2", // TODO: make this configurable; must match a pulled model
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		"http://localhost:11434/api/generate", // TODO: make URL configurable
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var streamResp OllamaStreamResponse
		if err := decoder.Decode(&streamResp); err != nil {
			return err
		}

		callback(streamResp.Response, streamResp.Done)

		if streamResp.Done {
			break
		}

		// Small delay to make streaming visually pleasant in the UI.
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
