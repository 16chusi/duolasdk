# AI 模块 API 文档

## 1. 核心思想

AI 模块为应用提供了与大语言模型（LLM）交互的统一抽象能力。其核心是 `AIProvider` 接口，它定义了通用的聊天、模型管理等操作，使得上层业务可以方便地切换和使用不同的 AI 服务提供商。

该模块目前主要包含两部分：

1.  **Ollama 实现**: 内置了一个完整的 Ollama 客户端，可以直接与本地运行的 Ollama 服务进行交互。
2.  **OpenAI 适配器**: 提供了一个符合 OpenAI API 规范的 HTTP 服务端点，能将来自第三方工具的 OpenAI 格式请求，转换为对内部 Ollama 客户端的调用。

## 2. 核心接口 `AIProvider`

这是所有 AI 服务提供商都需要实现的接口，定义在 `core/ai.go` 中。

-   `Chat(model string, messages []Message) (string, error)`
    发送一次性的聊天请求，并等待完整的响应。`Message` 结构体包含 `Role` 和 `Content`。

-   `ChatStream(model string, messages []Message, callback func(string)) error`
    发送聊天请求并以流式方式接收响应。每当收到新的内容片段时，都会调用 `callback` 函数。

-   `ListModels() ([]string, error)`
    获取当前 AI 服务提供商可用的模型列表。

-   `Validate() error`
    验证与 AI 服务提供商的连接和配置是否正确。

## 3. Ollama 实现

SDK 内置了对 Ollama 的完整支持，位于 `core/ai/ollama.go`。

### `OllamaProvider`

`AIProvider` 接口的 Ollama 实现。通过 `NewOllamaProvider(log *core.AppLog, baseURL string)` 函数创建实例，其中 `baseURL` 是 Ollama 服务的地址（例如 `http://127.0.0.1:11434`）。

### `OllamaClient`

这是一个更底层的客户端，封装了对 Ollama REST API 的直接调用。它提供了比 `AIProvider` 更丰富的功能，包括：

-   模型管理：`ListModels`, `ShowModelDetails`, `CopyModel`, `DeleteModel`, `PullModel`
-   生成文本：`GenerateCompletion`, `Chat` (支持流式和非流式)
-   生成词向量：`GenerateEmbeddings`, `GenerateSingleEmbedding`
-   查看运行状态：`ListRunningModels`

## 4. OpenAI 兼容适配器

这是该模块的一个强大功能，位于 `core/ai/openai_adapter.go`。它启动一个 HTTP 服务器，监听 `/v1/chat/completions` 端点。

### 核心功能

-   **协议转换**: 接收完全符合 OpenAI `chat/completions` API 规范的 HTTP 请求（包括 JSON 结构、Authorization 头等）。
-   **无缝代理**: 将接收到的请求转换为对内部 `OllamaClient` 的调用，然后将 Ollama 的返回结果再包装成 OpenAI 的格式返回给调用方。
-   **支持流式与非流式**: 能正确处理 `stream: true` 和 `stream: false` 两种模式。

### 使用场景

当你的应用需要集成一个只支持 OpenAI API 的第三方客户端或工具时，这个适配器就非常有用了。你可以将这个第三方工具的 API Endpoint 指向 `duola-desktop` 启动的本地适配器地址，从而让它在不知情的情况下使用你本地的 Ollama 模型，极大地增强了应用的兼容性和扩展性。

通过 `NewOpenAIAdapter(...)` 创建实例，它本身实现了 `http.Handler` 接口，可以轻松集成到 Go 的 HTTP 服务器中。

## 5. 使用示例

### 5.1. 使用 `AIProvider` 进行聊天

这个例子展示了如何使用 `AIProvider` 接口进行一次性的和流式的聊天。

```go
package main

import (
	"fmt"
	"log"

	"github.com/fzxs8/duolasdk/core"
	"github.com/fzxs8/duolasdk/core/ai"
)

func main() {
	appLogger := core.NewLogger(&core.LoggerOption{Level: "info"})
	ollamaBaseURL := "http://127.0.0.1:11434" // 你的 Ollama 服务地址

	// 创建一个 Ollama 提供者
	provider := ai.NewOllamaProvider(appLogger, ollamaBaseURL)

	// 验证与 Ollama 的连接
	if err := provider.Validate(); err != nil {
		log.Fatalf("无法连接到 Ollama: %v", err)
	}
	fmt.Println("Ollama 连接成功！")

	// 列出可用模型
	models, _ := provider.ListModels()
	fmt.Printf("可用模型: %v
", models)

	messages := []core.Message{
		{Role: "user", Content: "你好，请用中文简单介绍一下自己。"},
	}

	// --- 1. 一次性聊天 ---
	fmt.Println("
--- 一次性聊天 --- ")
	response, err := provider.Chat("llama3", messages)
	if err != nil {
		log.Printf("聊天失败: %v", err)
	} else {
		fmt.Println("AI 响应:", response)
	}

	// --- 2. 流式聊天 ---
	fmt.Println("
--- 流式聊天 --- ")
	fmt.Print("AI 响应: ")
	err = provider.ChatStream("llama3", messages, func(chunk string) {
		// 每收到一个内容片段，就直接打印
		fmt.Print(chunk)
	})
	if err != nil {
		log.Printf("流式聊天失败: %v", err)
	}
	fmt.Println()
}
```

### 5.2. 启动 OpenAI 兼容适配器

这个例子展示了如何启动一个 HTTP 服务器来托管 OpenAI 适配器，使其能被其他工具调用。

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/fzxs8/duolasdk/core"
	"github.com/fzxs8/duolasdk/core/ai"
)

func main() {
	appLogger := core.NewLogger(&core.LoggerOption{Level: "info"})
	ollamaBaseURL := "http://127.0.0.1:11434"
	adapterListenAddr := ":8080" // 适配器监听的地址和端口

	// 1. 创建底层的 Ollama 客户端
	ollamaClient := ai.NewOllamaClient(appLogger, ollamaBaseURL)

	// 2. 创建 OpenAI 适配器实例
	// 注意：在 Wails 应用中，此 context 由 Wails 运行时提供
	ctx := context.Background()
	openaiAdapter := ai.NewOpenAIAdapter(appLogger, ollamaClient, ctx)

	// 3. 设置路由并启动 HTTP 服务器
	http.Handle("/v1/chat/completions", openaiAdapter)

	log.Printf("OpenAI 适配器正在监听 %s", adapterListenAddr)
	log.Printf("你可以使用以下 curl 命令进行测试:")
	log.Printf(`curl http://localhost:8080/v1/chat/completions -H "Content-Type: application/json" -d '{
	 "model": "llama3",
	 "messages": [{"role": "user", "content": "你好！"}],
	 "stream": false
	}' `)

	if err := http.ListenAndServe(adapterListenAddr, nil); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
```