package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fzxs8/duolasdk/core/ai"
)

// AIProvider AI提供者接口
type AIProvider interface {
	// Chat 发送聊天消息
	Chat(model string, messages []ai.ChatMessage) (string, error)
	// ChatStream 发送聊天消息并流式返回结果
	ChatStream(model string, messages []ai.ChatMessage, callback func(string)) error
	// ListModels 获取模型列表
	ListModels() ([]string, error)
	// Validate 验证连接和配置
	Validate() error
}

// OllamaProvider Ollama提供者实现
type OllamaProvider struct {
	log          *AppLog
	ollamaClient *ai.OllamaClient
}

// NewOllamaProvider 创建Ollama提供者实例
func NewOllamaProvider(log *AppLog, baseURL string) *OllamaProvider {
	return &OllamaProvider{
		log:          log,
		ollamaClient: ai.NewOllamaClient(log, baseURL),
	}
}

// Chat 发送聊天消息
func (o *OllamaProvider) Chat(model string, messages []ai.ChatMessage) (string, error) {
	chatReq := ai.ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	resp, _, err := o.ollamaClient.Chat(chatReq, false)
	if err != nil {
		return "", fmt.Errorf("ollama chat failed: %w", err)
	}

	return resp.Message.Content, nil
}

// ChatStream 发送聊天消息并流式返回结果
func (o *OllamaProvider) ChatStream(model string, messages []ai.ChatMessage, callback func(string)) error {
	chatReq := ai.ChatRequest{
		Model:    model,
		Messages: messages,
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

		var chatResponse ai.ChatResponse // Use ai.ChatResponse for parsing stream chunks
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

// ListModels 获取模型列表
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

// Validate 验证连接和配置
func (o *OllamaProvider) Validate() error {
	_, err := o.ollamaClient.ListModels()
	if err != nil {
		return fmt.Errorf("ollama validation failed: %w", err)
	}
	return nil
}
