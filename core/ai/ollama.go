package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/16chusi/duolasdk/core"
)

// Message represents a message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ModelDetails represents the details of a model.
type ModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// --- API Request/Response Structures ---

// GenerateRequest for POST /api/generate
type GenerateRequest struct {
	Model     string `json:"model"`
	KeepAlive *int   `json:"keep_alive,omitempty"` // Use pointer for omitempty on zero value
	Prompt    string `json:"prompt,omitempty"`     // For actual generation, not just unload
}

// GenerateResponse for POST /api/generate
type GenerateResponse struct {
	Model      string `json:"model"`
	CreatedAt  string `json:"created_at"`
	Response   string `json:"response"`
	Done       bool   `json:"done"`
	DoneReason string `json:"done_reason"`
}

// ChatRequest for POST /api/chat
type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"` // Now uses ai.Message
	KeepAlive *int      `json:"keep_alive,omitempty"`
	Stream    bool      `json:"stream,omitempty"` // Add stream option for chat
}

// ChatResponse for POST /api/chat (non-streaming)
type ChatResponse struct {
	Model      string  `json:"model"`
	CreatedAt  string  `json:"created_at"`
	Message    Message `json:"message"` // Now uses ai.Message
	DoneReason string  `json:"done_reason"`
	Done       bool    `json:"done"`
}

// CreateRequest for POST /api/create
type CreateRequest struct {
	Model string            `json:"model"`
	Files map[string]string `json:"files"` // Map of filename to SHA256 digest
}

// CreateResponse for streaming POST /api/create
type CreateResponse struct {
	Status string `json:"status"`
}

// Model for GET /api/tags
type Model struct {
	Name       string       `json:"name"`
	ModifiedAt string       `json:"modified_at"`
	Size       int64        `json:"size"`
	Digest     string       `json:"digest"`
	Details    ModelDetails `json:"details"`
}

// ListModelsResponse for GET /api/tags
type ListModelsResponse struct {
	Models []Model `json:"models"`
}

// ShowRequest for POST /api/show
type ShowRequest struct {
	Model string `json:"model"`
}

// ShowResponse for POST /api/show
type ShowResponse struct {
	Modelfile  string                 `json:"modelfile"`
	Parameters string                 `json:"parameters"`
	Template   string                 `json:"template"`
	Details    ModelDetails           `json:"details"`
	ModelInfo  map[string]interface{} `json:"model_info"` // Can be complex, use map[string]interface{}
}

// CopyRequest for POST /api/copy
type CopyRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// DeleteRequest for DELETE /api/delete
type DeleteRequest struct {
	Model string `json:"model"`
}

// PullRequest for POST /api/pull
type PullRequest struct {
	Model string `json:"model"`
}

// EmbedRequest for POST /api/embed (multiple inputs)
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbedResponse for POST /api/embed
type EmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
}

// RunningModel for POST /api/ps
type RunningModel struct {
	Name      string       `json:"name"`
	Model     string       `json:"model"`
	Size      int64        `json:"size"`
	Digest    string       `json:"digest"`
	Details   ModelDetails `json:"details"`
	ExpiresAt string       `json:"expires_at"`
	SizeVRAM  int64        `json:"size_vram"`
}

// ListRunningModelsResponse for POST /api/ps
type ListRunningModelsResponse struct {
	Models []RunningModel `json:"models"`
}

// SingleEmbedRequest for POST /api/embeddings (single input)
type SingleEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// SingleEmbedResponse for POST /api/embeddings
type SingleEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// OllamaClient encapsulates the Ollama API interactions.
type OllamaClient struct {
	httpClient *core.HttpCli
	log        *core.AppLog
}

// NewOllamaClient creates a new OllamaClient instance with a configurable base URL.
func NewOllamaClient(log *core.AppLog, baseURL string) *OllamaClient {
	cli := core.NewHttp(log)
	cli.Create(&core.Config{
		BaseURL: baseURL,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})
	log.Debug("NewOllamaClient: HttpCli baseURL after creation", "baseURL", baseURL) // Added debug log
	return &OllamaClient{
		httpClient: cli,
		log:        log,
	}
}

// HttpClient returns the underlying core.HttpCli instance.
func (oc *OllamaClient) HttpClient() *core.HttpCli {
	return oc.httpClient
}

// OllamaProvider implements core.AIProvider for Ollama.
type OllamaProvider struct {
	log          *core.AppLog
	ollamaClient *OllamaClient
}

// NewOllamaProvider creates an OllamaProvider instance.
func NewOllamaProvider(log *core.AppLog, baseURL string) core.AIProvider {
	return &OllamaProvider{
		log:          log,
		ollamaClient: NewOllamaClient(log, baseURL),
	}
}

// Chat sends a chat message to Ollama.
func (o *OllamaProvider) Chat(model string, messages []core.Message) (string, error) {
	// Convert core.Message to ai.Message
	ollamaMessages := make([]Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	chatReq := ChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   false,
	}

	resp, _, err := o.ollamaClient.Chat(chatReq, false)
	if err != nil {
		return "", fmt.Errorf("ollama chat failed: %w", err)
	}

	return resp.Message.Content, nil
}

// ChatStream sends a chat message to Ollama and streams the response.
func (o *OllamaProvider) ChatStream(model string, messages []core.Message, callback func(string)) error {
	// Convert core.Message to ai.Message
	ollamaMessages := make([]Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	chatReq := ChatRequest{
		Model:    model,
		Messages: ollamaMessages,
		Stream:   true,
	}

	_, httpResp, err := o.ollamaClient.Chat(chatReq, true)
	if err != nil {
		return fmt.Errorf("ollama chat stream failed: %w", err)
	}
	defer httpResp.Body.Close()

	reader := bufio.NewReader(httpResp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("reading stream response failed: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var chatResponse ChatResponse // Use ChatResponse for parsing stream chunks
		err = json.Unmarshal([]byte(line), &chatResponse)
		if err != nil {
			// Ignore parsing errors, continue to next line. Ollama sometimes sends non-JSON keep-alive messages.
			continue
		}

		if chatResponse.Message.Content != "" {
			callback(chatResponse.Message.Content)
		}

		// If Ollama marks stream as done, exit proactively
		if chatResponse.Done {
			break
		}
	}

	return nil
}

// ListModels gets the list of models from Ollama.
func (o *OllamaProvider) ListModels() ([]string, error) {
	resp, err := o.ollamaClient.ListModels()
	if err != nil {
		return nil, fmt.Errorf("ollama list models failed: %w", err)
	}

	models := make([]string, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = m.Name
	}

	return models, nil
}

// Validate validates the connection and configuration with Ollama.
func (o *OllamaProvider) Validate() error {
	_, err := o.ollamaClient.ListModels()
	if err != nil {
		return fmt.Errorf("ollama validation failed: %w", err)
	}
	return nil
}

// GenerateCompletion sends a request to the /api/generate endpoint.
// It can be used for text generation or to unload a model.
func (oc *OllamaClient) GenerateCompletion(req GenerateRequest) (*GenerateResponse, error) {
	resp, err := oc.httpClient.Post("/generate", core.Options{Body: req})
	if err != nil {
		return nil, fmt.Errorf("failed to send generate request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generate request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var genResp GenerateResponse
	if err := json.Unmarshal([]byte(resp.Body), &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal generate response: %w", err)
	}
	oc.log.Debug("Generate completion response received", "model", genResp.Model, "done", genResp.Done)
	return &genResp, nil
}

// Chat sends a request to the /api/chat endpoint.
// If stream is true, it returns a raw http.Response for streaming.
// If stream is false, it returns a parsed ChatResponse.
func (oc *OllamaClient) Chat(req ChatRequest, stream bool) (*ChatResponse, *http.Response, error) {

	if stream {
		httpResp, err := oc.httpClient.PostStream("/chat", core.Options{Body: req})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to send streaming chat request: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			return nil, nil, fmt.Errorf("streaming chat request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
		}
		oc.log.Debug("Streaming chat response initiated")
		return nil, httpResp, nil
	}

	resp, err := oc.httpClient.Post("/chat", core.Options{Body: req})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send chat request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("chat request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal([]byte(resp.Body), &chatResp); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal chat response: %w", err)
	}
	oc.log.Debug("Chat response received", "model", chatResp.Model, "done", chatResp.Done)
	return &chatResp, nil, nil
}

// CreateModel sends a request to the /api/create endpoint.
// This is a streaming API, so it returns a raw http.Response.
func (oc *OllamaClient) CreateModel(req CreateRequest) (*http.Response, error) {
	httpResp, err := oc.httpClient.PostStream("/create", core.Options{Body: req})
	if err != nil {
		return nil, fmt.Errorf("failed to send create model request: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("create model request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}
	oc.log.Debug("Create model response initiated")
	return httpResp, nil
}

// ListModels sends a request to the /api/tags endpoint.
func (oc *OllamaClient) ListModels() (*ListModelsResponse, error) {
	resp, err := oc.httpClient.Get("/tags", core.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to send list models request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var listResp ListModelsResponse
	if err := json.Unmarshal([]byte(resp.Body), &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list models response: %w", err)
	}
	oc.log.Debug("List models response received", "count", len(listResp.Models))
	return &listResp, nil
}

// ShowModelDetails sends a request to the /api/show endpoint.
func (oc *OllamaClient) ShowModelDetails(req ShowRequest) (*ShowResponse, error) {
	resp, err := oc.httpClient.Post("/show", core.Options{Body: req})
	if err != nil {
		return nil, fmt.Errorf("failed to send show model details request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("show model details request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var showResp ShowResponse
	if err := json.Unmarshal([]byte(resp.Body), &showResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal show model details response: %w", err)
	}
	oc.log.Debug("Show model details response received", "model", req.Model)
	return &showResp, nil
}

// CopyModel sends a request to the /api/copy endpoint.
func (oc *OllamaClient) CopyModel(req CopyRequest) error {
	resp, err := oc.httpClient.Post("/copy", core.Options{Body: req})
	if err != nil {
		return fmt.Errorf("failed to send copy model request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("copy model request failed with status %d: %s", resp.StatusCode, resp.Body)
	}
	oc.log.Debug("Copy model request successful")
	return nil
}

// DeleteModel sends a request to the /api/delete endpoint.
func (oc *OllamaClient) DeleteModel(req DeleteRequest) error {
	resp, err := oc.httpClient.Delete("/delete", core.Options{Body: req})
	if err != nil {
		return fmt.Errorf("failed to send delete model request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete model request failed with status %d: %s", resp.StatusCode, resp.Body)
	}
	oc.log.Debug("Delete model request successful")
	return nil
}

// PullModel sends a request to the /api/pull endpoint.
// This is a streaming API, so it returns a raw http.Response.
func (oc *OllamaClient) PullModel(req PullRequest) (*http.Response, error) {
	httpResp, err := oc.httpClient.PostStream("/pull", core.Options{Body: req})
	if err != nil {
		return nil, fmt.Errorf("failed to send pull model request: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		return nil, fmt.Errorf("pull model request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}
	oc.log.Debug("Pull model response initiated")
	return httpResp, nil
}

// GenerateEmbeddings sends a request to the /api/embed endpoint for multiple inputs.
func (oc *OllamaClient) GenerateEmbeddings(req EmbedRequest) (*EmbedResponse, error) {
	resp, err := oc.httpClient.Post("/embed", core.Options{Body: req})
	if err != nil {
		return nil, fmt.Errorf("failed to send generate embeddings request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generate embeddings request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var embedResp EmbedResponse
	if err := json.Unmarshal([]byte(resp.Body), &embedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal generate embeddings response: %w", err)
	}
	oc.log.Debug("Generate embeddings response received", "model", embedResp.Model)
	return &embedResp, nil
}

// ListRunningModels sends a request to the /api/ps endpoint.
func (oc *OllamaClient) ListRunningModels() (*ListRunningModelsResponse, error) {
	resp, err := oc.httpClient.Post("/ps", core.Options{}) // /ps is POST, but no body needed
	if err != nil {
		return nil, fmt.Errorf("failed to send list running models request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list running models request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var listResp ListRunningModelsResponse
	if err := json.Unmarshal([]byte(resp.Body), &listResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list running models response: %w", err)
	}
	oc.log.Debug("List running models response received", "count", len(listResp.Models))
	return &listResp, nil
}

// GenerateSingleEmbedding sends a request to the /api/embeddings endpoint for a single input.
func (oc *OllamaClient) GenerateSingleEmbedding(req SingleEmbedRequest) (*SingleEmbedResponse, error) {
	resp, err := oc.httpClient.Post("/embeddings", core.Options{Body: req})
	if err != nil {
		return nil, fmt.Errorf("failed to send generate single embedding request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generate single embedding request failed with status %d: %s", resp.StatusCode, resp.Body)
	}

	var embedResp SingleEmbedResponse
	if err := json.Unmarshal([]byte(resp.Body), &embedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal generate single embedding response: %w", err)
	}
	oc.log.Debug("Generate single embedding response received")
	return &embedResp, nil
}
