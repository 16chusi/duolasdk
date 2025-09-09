package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Message 聊天消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Model   string  `json:"model"`
	Message Message `json:"message"`
}

// AIProvider AI提供者接口
type AIProvider interface {
	// Chat 发送聊天消息
	Chat(model string, messages []Message) (string, error)
	// ChatStream 发送聊天消息并流式返回结果
	ChatStream(model string, messages []Message, callback func(string)) error
	// ListModels 获取模型列表
	ListModels() ([]string, error)
	// Validate 验证连接和配置
	Validate() error
}

// OllamaProvider Ollama提供者实现
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// NewOllamaProvider 创建Ollama提供者实例
func NewOllamaProvider(baseURL string) *OllamaProvider {
	return &OllamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

// Chat 发送聊天消息
func (o *OllamaProvider) Chat(model string, messages []Message) (string, error) {
	requestBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求体失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", o.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	var chatResponse ChatResponse
	err = json.Unmarshal(body, &chatResponse)
	if err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	return chatResponse.Message.Content, nil
}

// ChatStream 发送聊天消息并流式返回结果
func (o *OllamaProvider) ChatStream(model string, messages []Message, callback func(string)) error {
	requestBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", o.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("读取流式响应失败: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var chatResponse ChatResponse
		err = json.Unmarshal([]byte(line), &chatResponse)
		if err != nil {
			// 忽略解析错误，继续处理下一行
			continue
		}

		if chatResponse.Message.Content != "" {
			callback(chatResponse.Message.Content)
		}
	}

	return nil
}

// ListModels 获取模型列表
func (o *OllamaProvider) ListModels() ([]string, error) {
	url := fmt.Sprintf("%s/api/tags", o.baseURL)
	resp, err := o.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	// 简化的模型列表解析
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	models := make([]string, 0)
	if modelList, ok := result["models"].([]interface{}); ok {
		for _, item := range modelList {
			if modelInfo, ok := item.(map[string]interface{}); ok {
				if name, ok := modelInfo["name"].(string); ok {
					models = append(models, name)
				}
			}
		}
	}

	return models, nil
}

// Validate 验证连接和配置
func (o *OllamaProvider) Validate() error {
	url := fmt.Sprintf("%s/api/tags", o.baseURL)
	resp, err := o.client.Get(url)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("验证失败，状态码: %d", resp.StatusCode)
	}

	return nil
}
