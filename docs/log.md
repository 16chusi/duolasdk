# 日志模块 (log.go) API 文档

## 1. 核心思想

`log.go` 提供了一个简单而强大的日志记录能力。它基于 `github.com/labstack/gommon/log` 库进行封装，提供了可配置的日志实例 `AppLog`，能够满足开发和生产环境中的不同日志需求。

## 2. 核心组件 `AppLog`

`AppLog` 是对 `gommon/log.Logger` 的封装，增加了配置能力。

### 2.1. 创建日志实例

通过 `NewLogger(o *LoggerOption)` 函数创建一个新的 `AppLog` 实例。

### 2.2. 配置 (`LoggerOption`)

创建实例时，需要传入 `LoggerOption` 结构体进行配置：

-   `Type`: 日志输出类型。
    -   `"console"` (默认): 输出到标准控制台。
    -   `"file"`: 输出到文件。
-   `FileName`: 日志文件的完整路径。仅在 `Type` 为 `"file"` 时有效。
-   `Level`: 日志级别，不区分大小写。支持以下级别：
    -   `"debug"`
    -   `"info"`
    -   `"warn"`
    -   `"error"`
    -   `"off"` (关闭所有日志)
-   `Prefix`: 每条日志记录的前缀，用于区分不同的模块或实例。

### 2.3. 使用方法

`AppLog` 实例拥有与标准日志库类似的方法：

-   `Debug(message ...interface{})`
-   `Info(message ...interface{})`
-   `Warn(message ...interface{})`
-   `Error(message ...interface{})`
-   `Fatal(message ...interface{})` (会导致程序退出)
-   `Debugf(format string, args ...interface{})` (格式化输出)
-   ...以及 `Infof`, `Warnf`, `Errorf`, `Fatalf`

## 3. Wails 日志适配器

为了将 SDK 的日志与 Wails 框架的日志系统集成，模块提供了一个适配器。

-   `NewWailsLog(appLog *AppLog) *WailsLog`

该函数接收一个 `AppLog` 实例，返回一个 `*WailsLog` 对象。这个对象实现了 Wails 所需的日志接口 (`Print`, `Trace`, `Debug`, `Info`, `Warning`, `Error`, `Fatal`)，可以方便地在 Wails 项目的 `wails.json` 中配置为应用的日志记录器。

## 4. 使用示例

```go
package main

import "github.com/16chusi/duolasdk/core"

func main() {
	// --- 示例 1: 输出到控制台 ---
	consoleLogger := core.NewLogger(&core.LoggerOption{
		Type:   "console",
		Level:  "debug", // 显示 debug 及以上级别
		Prefix: "ConsoleApp",
	})

	consoleLogger.Debug("这是一条调试信息")
	consoleLogger.Info("服务已启动")
	consoleLogger.Warnf("配置项 '%s' 即将被弃用", "old_config")

	// --- 示例 2: 输出到文件 ---
	fileLogger := core.NewLogger(&core.LoggerOption{
		Type:     "file",
		FileName: "./logs/app.log",
		Level:    "info", // 只记录 info 及以上级别
		Prefix:   "FileApp",
	})

	fileLogger.Info("这是一条会写入文件的信息")
	fileLogger.Debug("这条调试信息不会写入文件，因为日志级别不够")
	fileLogger.Error("发生了一个错误：无法连接到数据库")

	// --- 示例 3: 创建带不同前缀的子 Logger ---
	// fileLogger 实例保持不变
	dbLogger := fileLogger.WithPrefix("FileApp-Database")
	dbLogger.Info("数据库模块初始化...")
}
```