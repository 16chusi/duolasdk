package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/fzxs8/duolasdk/core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// --- Data Structures ---

// LogEntry represents a single log message sent to the frontend.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// ChatCompletionRequest mirrors the structure of an OpenAI chat completion request.
type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

// ChatMessage mirrors the structure of a message in an OpenAI request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletion mirrors the structure of a non-streaming OpenAI chat completion response.
type ChatCompletion struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice mirrors the structure of a choice in an OpenAI response.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage mirrors the structure of the usage field in an OpenAI response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk mirrors the structure of a streaming OpenAI chat completion chunk.
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

// ChunkChoice mirrors the structure of a choice in a streaming chunk.
type ChunkChoice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta mirrors the structure of the delta field in a streaming choice.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ErrorResponse mirrors the structure of an OpenAI error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail provides details about the error.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// --- Adapter Implementation ---

// OpenAIAdapter holds the dependencies for the adapter service.
type OpenAIAdapter struct {
	log          *core.AppLog
	ollamaClient *OllamaClient
	runtimeCtx   context.Context
}

// NewOpenAIAdapter creates a new adapter instance.
func NewOpenAIAdapter(log *core.AppLog, ollamaClient *OllamaClient, ctx context.Context) *OpenAIAdapter {
	return &OpenAIAdapter{
		log:          log,
		ollamaClient: ollamaClient,
		runtimeCtx:   ctx,
	}
}

// emitLog sends a log message to the frontend via Wails events.
func (a *OpenAIAdapter) emitLog(level string, message string) {
	if a.runtimeCtx == nil {
		return
	}
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}
	runtime.EventsEmit(a.runtimeCtx, "openai-adapter-log", entry)
}

// ServeHTTP is the main entry point for handling requests to /v1/chat/completions.
func (a *OpenAIAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.emitLog("INFO", fmt.Sprintf("Received request: %s %s", r.Method, r.URL.Path))

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight OPTIONS request
	if r.Method == http.MethodOptions {
		a.emitLog("DEBUG", "Handling CORS preflight request")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		errMsg := fmt.Sprintf("Method %s not allowed, please use POST", r.Method)
		a.emitLog("ERROR", errMsg)
		writeErrorResponse(w, http.StatusMethodNotAllowed, ErrorDetail{Message: errMsg, Type: "invalid_request_error"})
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errMsg := fmt.Sprintf("Failed to decode request body: %v", err)
		a.emitLog("ERROR", errMsg)
		writeErrorResponse(w, http.StatusBadRequest, ErrorDetail{Message: "Invalid request body", Type: "invalid_request_error"})
		return
	}

	if req.Stream {
		a.handleStream(w, r, &req)
	} else {
		a.handleNonStream(w, r, &req)
	}
}

// handleNonStream handles non-streaming chat completion requests.
func (a *OpenAIAdapter) handleNonStream(w http.ResponseWriter, r *http.Request, req *ChatCompletionRequest) {
	a.emitLog("INFO", fmt.Sprintf("Handling non-streaming request for model: %s", req.Model))

	ollamaMessages := make([]Message, len(req.Messages))
	for i, msg := range req.Messages {
		ollamaMessages[i] = Message{Role: msg.Role, Content: msg.Content}
	}

	ollamaReq := ChatRequest{Model: req.Model, Messages: ollamaMessages, Stream: false}

	a.emitLog("DEBUG", "Forwarding request to Ollama...")
	ollamaResp, _, err := a.ollamaClient.Chat(ollamaReq, false)
	if err != nil {
		errMsg := fmt.Sprintf("Ollama chat request failed: %v", err)
		a.emitLog("ERROR", errMsg)
		writeErrorResponse(w, http.StatusInternalServerError, ErrorDetail{Message: err.Error(), Type: "ollama_error"})
		return
	}
	a.emitLog("DEBUG", "Received response from Ollama.")

	openaiResp := ChatCompletion{ID: generateID(), Object: "chat.completion", Created: time.Now().Unix(), Model: ollamaResp.Model, Choices: []Choice{{Index: 0, Message: ChatMessage{Role: ollamaResp.Message.Role, Content: ollamaResp.Message.Content}, FinishReason: "stop"}}, Usage: Usage{PromptTokens: 0, CompletionTokens: 0, TotalTokens: 0}}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(openaiResp); err != nil {
		a.emitLog("ERROR", fmt.Sprintf("Failed to encode non-streaming response: %v", err))
	}
	a.emitLog("INFO", "Non-streaming request completed successfully.")
}

// handleStream handles streaming chat completion requests.
func (a *OpenAIAdapter) handleStream(w http.ResponseWriter, r *http.Request, req *ChatCompletionRequest) {
	a.emitLog("INFO", fmt.Sprintf("Handling streaming request for model: %s", req.Model))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		errMsg := "Streaming unsupported by the server"
		a.emitLog("ERROR", errMsg)
		writeErrorResponse(w, http.StatusInternalServerError, ErrorDetail{Message: errMsg, Type: "server_error"})
		return
	}

	ollamaMessages := make([]Message, len(req.Messages))
	for i, msg := range req.Messages {
		ollamaMessages[i] = Message{Role: msg.Role, Content: msg.Content}
	}

	ollamaReq := ChatRequest{Model: req.Model, Messages: ollamaMessages, Stream: true}

	a.emitLog("DEBUG", "Forwarding streaming request to Ollama...")
	_, httpResp, err := a.ollamaClient.Chat(ollamaReq, true)
	if err != nil {
		errMsg := fmt.Sprintf("Ollama streaming chat request failed: %v", err)
		a.emitLog("ERROR", errMsg)
		return
	}
	defer httpResp.Body.Close()
	a.emitLog("DEBUG", "Received streaming response from Ollama.")

	reader := bufio.NewReader(httpResp.Body)
	id := generateID()
	created := time.Now().Unix()
	isFirstChunk := true

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				a.emitLog("ERROR", fmt.Sprintf("Error reading from Ollama stream: %v", err))
			}
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var ollamaChunk ChatResponse
		if err := json.Unmarshal([]byte(line), &ollamaChunk); err != nil {
			a.emitLog("WARN", fmt.Sprintf("Failed to unmarshal stream chunk, skipping: %s", line))
			continue
		}

		openaiChunk := ChatCompletionChunk{ID: id, Object: "chat.completion.chunk", Created: created, Model: req.Model, Choices: []ChunkChoice{{Index: 0, Delta: Delta{}}}}

		if isFirstChunk {
			openaiChunk.Choices[0].Delta.Role = "assistant"
			isFirstChunk = false
		}

		if ollamaChunk.Message.Content != "" {
			openaiChunk.Choices[0].Delta.Content = ollamaChunk.Message.Content
		}

		if ollamaChunk.Done {
			openaiChunk.Choices[0].FinishReason = "stop"
		}

		chunkBytes, err := json.Marshal(openaiChunk)
		if err != nil {
			a.emitLog("ERROR", fmt.Sprintf("Failed to marshal OpenAI chunk: %v", err))
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", chunkBytes)
		flusher.Flush()

		if ollamaChunk.Done {
			break
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
	a.emitLog("INFO", "Streaming request completed successfully.")
}

// --- Helper Functions ---

// writeErrorResponse sends a standardized OpenAI-compatible error response.
func writeErrorResponse(w http.ResponseWriter, statusCode int, errDetail ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errResp := ErrorResponse{Error: errDetail}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		http.Error(w, `{"error": {"message": "Failed to serialize error response"}}`, http.StatusInternalServerError)
	}
}

// generateID creates a unique ID with the "chatcmpl-" prefix.
func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 24)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return "chatcmpl-" + string(b)
}
