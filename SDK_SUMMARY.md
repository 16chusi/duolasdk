# Duola SDK 项目概要

本文档为 `duolasdk` 的核心功能和架构概要，旨在为 AI 模型提供一个浓缩的、节省上下文的参考，以便快速理解该项目。

---

## 1. 项目概述

`duolasdk` 是一个用 Go 语言编写的、功能丰富的工具包（SDK），旨在为桌面应用程序（特别是基于 Wails 的应用）提供一系列通用的核心功能。它通过模块化的设计，封装了数据存储、网络通信、日志记录和 AI 能力等常用操作。

---

## 2. 项目核心目录结构

```
duolasdk/
├── SDK_SUMMARY.md      # (本文档)
├── app_store.go        # SDK主入口，封装了统一的存储接口
├── go.mod              # Go模块依赖
├── core/               # 核心功能实现目录
│   ├── store.go        # 核心存储逻辑 (LocalStore, MemStore, IStore接口)
│   ├── net.go          # 网络客户端(HttpCli)实现
│   ├── log.go          # 日志实现
│   ├── ai.go           # AI提供者(AIProvider)通用接口
│   └── ai/             # AI具体实现子目录
│       ├── ollama.go
│       └── openai_adapter.go
└── docs/               # 详细文档
```

---

## 3. 核心模块解析

### 3.1. 存储模块 (`app_store.go`, `core/store.go`)

这是 SDK 最核心的功能，提供了一个强大的、支持多种模式的数据存储解决方案。

*   **设计思想**: 
    *   **双模式存储**: 同时提供了基于 **SQLite 的持久化存储** (`LocalStore`) 和纯**内存存储** (`MemStore`)。
    *   **统一接口**: 两种存储模式都实现了统一的 `core.IStore` 接口，使得上层调用者可以用同样的方式操作，只需在初始化时决定数据是否持久化。
    *   **Redis 风格 API**: 接口设计模仿了 Redis，提供了 `KV`, `Hash`, `List`, `Set` 等多种数据结构的操作，非常易于使用。
    *   **原生 SQL 支持**: 除了封装好的方法，它也暴露了 `Exec`, `Query`, `QueryRow` 和事务 (`BeginTx`) 等接口，允许执行原生 SQL 以应对复杂查询。

*   **关键入口点**: 
    *   `duolasdk.NewStore()`: 创建一个 `AppStore` 实例，该实例内部同时管理 `LocalStore` 和 `MemStore`。
    *   `appStore.GetStore(persistent bool)`: 根据布尔值参数返回具体的存储实例（持久化或内存）。
    *   大部分方法（如 `Set`, `Get`）都接受一个可选的 `persistent` 布尔参数，用于动态选择本次操作的存储目标。

### 3.2. 网络模块 (`core/net.go`)

提供了一个功能完整的 HTTP 客户端。

*   **设计思想**:
    *   **封装 `http.Client`**: 核心为 `HttpCli` 结构体，封装了 Go 的标准 HTTP 客户端。
    *   **会话保持**: 内置 `cookiejar`，可以自动管理 Cookie，保持会话状态。
    *   **拦截器**: 支持请求（`RequestInterceptor`）和响应（`ResponseInterceptor`）拦截器，便于统一处理认证、日志、数据转换等逻辑。
    *   **流式响应**: 提供了 `PostStream` 方法，可以直接返回 `*http.Response`，用于处理流式数据。

*   **关键入口点**:
    *   `core.NewHttp()`: 创建一个新的 `HttpCli` 实例。
    *   `httpCli.Create()`: 用于配置客户端，如设置 `BaseURL` 和全局 `Headers`。
    *   `httpCli.Get()`, `httpCli.Post()` 等: 发起标准 HTTP 请求。

### 3.3. 日志模块 (`core/log.go`)

提供了一套统一的日志记录方案。

*   **设计思想**:
    *   **封装 `labstack/gommon/log`**: `AppLog` 结构体是对 `gommon/log` 的封装。
    *   **多目标输出**: 支持输出到控制台或文件。
    *   **可配置**: 支持配置日志级别、前缀和输出路径。

*   **关键入口点**:
    *   `core.NewLogger()`: 创建一个 `AppLog` 实例。
    *   `appLog.WithPrefix()`: 创建一个带有新前缀的子 Logger 实例，便于模块化日志记录。

### 3.4. AI 模块 (`core/ai/`)

提供了与大语言模型（LLM）交互的抽象和具体实现。

*   **设计思想**:
    *   **提供者模式**: 定义了 `core.AIProvider` 接口，包含 `Chat`, `ChatStream`, `ListModels` 等核心方法，使得可以接入不同的 AI 后端。
    *   **Ollama 实现**: `ai.OllamaProvider` 是 `AIProvider` 接口的具体实现，用于与 Ollama 服务进行交互。
    *   **OpenAI 适配器**: `ai.OpenAIAdapter` 是一个关键组件，它是一个 `http.Handler`，能将来自外部的、符合 OpenAI API 格式的请求（如 `/v1/chat/completions`），转换为对内部 Ollama 客户端的调用，从而让 `duolasdk` 能对外“伪装”成一个 OpenAI 服务。这对于兼容现有生态工具非常重要。

*   **关键入口点**:
    *   `ai.NewOllamaProvider()`: 创建一个与 Ollama 通信的 AI 提供者。
    *   `ai.NewOpenAIAdapter()`: 创建一个 OpenAI API 兼容层。

---

## 4. 如何使用 SDK (简例)

```go
import (
    "github.com/fzxs8/duolasdk"
    "github.com/fzxs8/duolasdk/core"
)

// 1. 初始化日志
logs := core.NewLogger(&core.LoggerOption{
    Level: "debug",
})

// 2. 初始化存储
// AppStore 同时管理持久化和内存存储
store := duolasdk.NewStore(core.StoreOption{
    Logger: logs,
    FileName: "my_app.db",
})

// 3. 使用存储 (默认持久化)
store.Set("username", "duola")
username, _ := store.Get("username")

// 4. 初始化网络客户端
httpCli := core.NewHttp(logs).Create(&core.Config{
    BaseURL: "https://api.example.com",
})

// 5. 发起网络请求
response, _ := httpCli.Get("/health", core.Options{})
```