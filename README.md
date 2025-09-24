# Duola SDK

Duola SDK 是一个功能丰富的桌面应用程序开发工具包，为 Duola 桌面应用程序提供核心功能支持。

## 功能特性

### 🗃️ 多模式数据存储
- **持久化存储**: 基于 SQLite 的本地存储，数据持久化保存
- **内存存储**: 高性能内存存储，适用于临时数据
- **Redis 风格 API**: 支持 Key-Value、Hash、List、Set 等多种数据结构
- **过期机制**: 支持键值过期自动清理
- **SQL 接口**: 提供原生 SQL 执行接口，满足复杂查询需求

### 🌐 网络通信
- **HTTP 客户端**: 功能完整的 HTTP 客户端实现
- **Cookie 管理**: 自动管理会话状态
- **拦截器支持**: 支持请求和响应拦截器
- **多种数据格式**: 支持 JSON、表单等数据发送
- **验证码支持**: 内置验证码图片获取功能

### 📝 统一日志
- **多级别日志**: 支持 DEBUG、INFO、WARN、ERROR 等日志级别
- **多输出方式**: 支持文件和控制台输出
- **自定义格式**: 支持自定义日志格式
- **线程安全**: 线程安全的日志记录实现

### 🔧 其他能力
- **AI 能力**: 集成人工智能相关功能接口
- **支付支持**: 提供支付功能接口
- **系统接口**: 提供操作系统级功能访问
- **发布工具**: 提供应用打包发布功能

## 安装

```bash
go get github.com/16chusi/duolasdk
```

## 快速开始

### 初始化存储

```go
import (
    "github.com/16chusi/duolasdk"
    "github.com/16chusi/duolasdk/core"
)

// 创建日志实例
logs := core.NewLogger(&core.LoggerOption{
    Level: "debug",
})

// 创建存储实例
store := duolasdk.NewStore(core.StoreOption{
    Logger: logs,
    FilePath: "/path/to/store",  // 存储文件路径
    FileName: "mydata.db",       // 存储文件名
})
```

### 使用 KV 存储

```go
// 设置键值
err := store.Set("username", "Alice", true)  // true 表示持久化存储
if err != nil {
    logs.Error("设置键值失败:", err)
}

// 获取键值
value, err := store.Get("username", true)
if err != nil {
    logs.Error("获取键值失败:", err)
} else {
    logs.Info("用户名:", value)
}

// 删除键值
err = store.Delete("username", true)
if err != nil {
    logs.Error("删除键值失败:", err)
}
```

### 使用 Redis 风格数据结构

```go
// Hash 操作
err = store.HSet("user:1001", "name", "Alice", true)
err = store.HSet("user:1001", "email", "alice@example.com", true)

name, _ := store.HGet("user:1001", "name", true)
userMap, _ := store.HGetAll("user:1001", true)

// List 操作
err = store.LPush("tasks", "task1", "task2", true)
err = store.RPush("tasks", "task3", true)

length, _ := store.LLen("tasks", true)
tasks, _ := store.LRange("tasks", 0, -1, true)

// Set 操作
err = store.SAdd("tags", "go", "sdk", "duola", true)
tags, _ := store.SMembers("tags", true)
```

### 网络请求

```go
// 创建 HTTP 客户端
httpCli := core.NewHttp(logs)

// GET 请求
response, err := httpCli.Get("https://api.example.com/users", core.Options{
    Query: map[string]string{
        "page": "1",
        "size": "10",
    },
})

// POST 请求 (JSON)
response, err := httpCli.Post("https://api.example.com/users", core.Options{
    Headers: map[string]string{
        "Content-Type": "application/json",
    },
    Body: map[string]interface{}{
        "name": "Alice",
        "age":  30,
    },
})
```

## 文档

详细文档请查看 [开发说明文档](./docs/docs.md)

## 许可证

MIT License

## 联系方式

如有问题，请提交 Issue 或联系开发团队。