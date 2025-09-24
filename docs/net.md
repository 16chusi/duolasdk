# 网络模块 (net.go) API 文档

## 1. 核心思想

`net.go` 文件提供了一个功能强大且易于使用的 HTTP 客户端 `HttpCli`。它的设计受到了流行的 JavaScript 库 `axios` 的启发，旨在为 Go 应用程序提供一个统一、可配置的网络请求解决方案。

## 2. 核心组件 `HttpCli`

`HttpCli` 是网络模块的核心，所有的网络操作都通过它来完成。通过 `NewHttp(log *AppLog)` 来创建一个新的实例。

### 主要特性

-   **实例配置**: 可以通过 `Create(cfg *Config)` 方法配置实例，设置 `BaseURL`、全局 `Headers` 等。
-   **Cookie 管理**: 内置 `cookiejar`，可以自动管理和发送 Cookie，实现会话保持。
-   **拦截器**: 支持请求拦截器 (`UseRequestInterceptor`) 和响应拦截器 (`UseResponseInterceptor`)，方便对请求和响应进行统一的预处理或后处理（例如：添加认证头、记录日志、统一错误处理等）。
-   **简洁的请求方法**: 提供了 `Get`, `Post`, `Put`, `Delete` 等便捷方法，简化了 RESTful API 的调用。
-   **灵活的选项**: 所有请求方法都接受一个 `Options` 结构体，可以方便地设置单次请求的 `Headers`, `Query` 参数和 `Body`。
-   **智能的 Body 处理**: 能根据 `Content-Type` 头自动将 `Body` 序列化为 JSON 或 form-urlencoded 格式。

## 3. 使用方法

### 3.1. 创建和配置

```go
// 1. 创建一个日志实例
logs := core.NewLogger(&core.LoggerOption{ Level: "debug" })

// 2. 创建 HttpCli 实例
httpCli := core.NewHttp(logs)

// 3. (可选) 配置实例
httpCli.Create(&core.Config{
    BaseURL: "https://api.example.com/v1/",
    Headers: map[string]string{
        "Accept": "application/json",
    },
})
```

### 3.2. 发起请求

```go
// 发起 GET 请求
resp, err := httpCli.Get("/users", core.Options{
    Query: map[string]string{
        "page": "1",
    },
})

// 发起 POST 请求
newUser := map[string]interface{}{"name": "Alice"}
resp, err := httpCli.Post("/users", core.Options{
    Headers: map[string]string{
        "Content-Type": "application/json",
    },
    Body: newUser,
})
```

## 4. 特殊功能

### 4.1. 流式请求

-   `PostStream(path string, options Options) (*http.Response, error)`

    对于需要处理流式响应的场景（例如 AI 对话），可以使用此方法。它不会自动读取和关闭响应体，而是直接返回原生的 `*http.Response`，让调用者可以自行处理响应流。

### 4.2. 获取验证码

-   `GetCaptchaImage(path string) (string, error)`

    一个便捷的工具函数，用于请求图片资源（如验证码），并将其直接编码为 Base64 的 Data URL 格式（例如 `data:image/png;base64,iVBORw0KGgo...`），可以直接在前端 `<img>` 标签中使用。

### 4.3. 断点续传下载

-   `DownloadFileWithResume(url string, filepath string, progressCallback func(int64, int64)) error`

    一个强大的文件下载工具，支持断点续传。它会将文件先下载到 `.tmp` 文件中，下载完成后再重命名。如果中途程序中断，下次调用时会自动从上次中断的位置继续下载。

    -   `url`: 文件的下载地址。
    -   `filepath`: 文件保存的最终路径。
    -   `progressCallback`: 一个可选的回调函数，用于实时接收下载进度，参数为 `(当前已下载字节数, 文件总字节数)`。

## 5. 使用示例

### 5.1. 基本 GET/POST 请求

```go
package main

import (
	"fmt"
	"log"

	"github.com/16chusi/duolasdk/core"
)

func main() {
	appLogger := core.NewLogger(&core.LoggerOption{Level: "debug"})
	httpCli := core.NewHttp(appLogger)

	httpCli.Create(&core.Config{
		BaseURL: "https://httpbin.org/",
	})

	// --- GET 请求 ---
	fmt.Println("--- 发起 GET 请求 ---")
	getResp, err := httpCli.Get("get", core.Options{
		Query: map[string]string{"client": "duolasdk"},
	})
	if err != nil {
		log.Fatalf("GET 请求失败: %v", err)
	}
	fmt.Printf("GET 响应状态码: %d
", getResp.StatusCode)
	//fmt.Printf("GET 响应体: %s
", getResp.Body)

	// --- POST JSON 请求 ---
	fmt.Println("
--- 发起 POST JSON 请求 ---")
	postBody := map[string]interface{}{
		"name":    "duola",
		"version": "v1",
	}
	postResp, err := httpCli.Post("post", core.Options{
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    postBody,
	})
	if err != nil {
		log.Fatalf("POST 请求失败: %v", err)
	}
	fmt.Printf("POST 响应状态码: %d
", postResp.StatusCode)
	//fmt.Printf("POST 响应体: %s
", postResp.Body)
}
```

### 5.2. 使用拦截器添加认证头

```go
func setupAuthInterceptor() *core.HttpCli {
	appLogger := core.NewLogger(&core.LoggerOption{Level: "debug"})
	httpCli := core.NewHttp(appLogger)

	// 定义一个请求拦截器
	authInterceptor := func(req *http.Request) error {
		// 在每个请求发送前，添加 Bearer Token
		token := "your-secret-api-token"
		req.Header.Set("Authorization", "Bearer "+token)
		fmt.Println("请求拦截器：已添加 Authorization 头")
		return nil
	}

	// 注册拦截器
	httpCli.UseRequestInterceptor(authInterceptor)

	httpCli.Create(&core.Config{
		BaseURL: "https://httpbin.org/",
	})

	return httpCli
}

func main() {
	authCli := setupAuthInterceptor()
	fmt.Println("--- 使用带拦截器的客户端发起请求 ---")
	authCli.Get("headers", core.Options{})
}
```

### 5.3. 断点续传下载文件

```go
func downloadFileExample() {
	appLogger := core.NewLogger(&core.LoggerOption{Level: "info"})
	httpCli := core.NewHttp(appLogger)

	fileURL := "https://proof.ovh.net/files/100Mio.dat" // 一个用于测试的大文件
	savePath := "./100Mio.dat"

	fmt.Printf("开始下载文件: %s
", fileURL)

	// 定义进度回调函数
	progressCallback := func(current, total int64) {
		if total > 0 {
			progress := float64(current) * 100 / float64(total)
			fmt.Printf("下载进度: %.2f%% (%d/%d bytes)
", progress, current, total)
		}
	}

	err := httpCli.DownloadFileWithResume(fileURL, savePath, progressCallback)
	if err != nil {
		log.Fatalf("
下载失败: %v", err)
	}

	fmt.Println("
文件下载完成！")
}
```