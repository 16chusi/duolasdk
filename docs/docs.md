# Duola SDK 开发说明文档

## 目录
1. [概述](#概述)
2. [核心模块](#核心模块)
3. [存储模块](#存储模块)
4. [网络模块](#网络模块)
5. [日志模块](#日志模块)
6. [使用示例](#使用示例)

## 概述

Duola SDK 是一个多模块的工具包，为 Duola 桌面应用程序提供核心功能支持。该 SDK 包含了存储、网络、日志等多个模块，旨在为开发者提供统一、易用的接口来构建功能丰富的桌面应用程序。

## 核心模块

核心模块 (`core`) 是 Duola SDK 的基础，包含了各种能力接口和基础实现。

### 主要组件
- **存储能力** - 提供本地和内存数据存储功能
- **网络能力** - 提供 HTTP 客户端功能
- **日志能力** - 提供统一的日志记录功能
- **操作系统能力** - 提供系统级功能访问
- **AI 能力** - 提供人工智能相关功能支持
- **支付能力** - 提供支付功能接口
- **发布能力** - 提供应用打包发布功能

## 存储模块

存储模块是 SDK 中最重要的模块之一，提供了多种数据存储方式。

### 架构设计

存储模块采用接口化设计，支持两种存储方式：
1. **LocalStore** - 基于 SQLite 的本地持久化存储
2. **MemStore** - 基于内存的临时存储

两种存储方式都实现了统一的 [IStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/core/store.go#L39-L73) 接口，保证了接口一致性。

### 接口规范

[IStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/core/store.go#L39-L73) 接口包含以下功能：

#### 基础 KV 操作
- `Set(key, value string) error` - 设置键值对
- `Get(key string) (string, error)` - 获取键值
- `Delete(key string) error` - 删除键值对
- `List() (map[string]string, error)` - 列出所有键值对

#### 过期操作
- `Expire(key string, milliseconds int64) error` - 设置键过期时间
- `TTL(key string) (int64, error)` - 获取键剩余生存时间

#### Hash 操作 (类似 Redis)
- `HSet(key, field, value string) error` - 设置哈希字段
- `HGet(key, field string) (string, error)` - 获取哈希字段值
- `HGetAll(key string) (map[string]string, error)` - 获取哈希所有字段
- `HDel(key string, fields ...string) error` - 删除哈希字段
- 等等...

#### List 操作 (类似 Redis)
- `LPush(key string, values ...string) error` - 从左侧推入列表
- `RPush(key string, values ...string) error` - 从右侧推入列表
- `LPop(key string) (string, error)` - 从左侧弹出元素
- `RPop(key string) (string, error)` - 从右侧弹出元素
- 等等...

#### Set 操作 (类似 Redis)
- `SAdd(key string, members ...string) error` - 添加集合成员
- `SRem(key string, members ...string) error` - 移除集合成员
- `SMembers(key string) ([]string, error)` - 获取所有集合成员
- 等等...

#### SQL 操作
- `Exec(query string, args ...interface{}) (sql.Result, error)` - 执行 SQL 语句
- `Query(query string, args ...interface{}) (*sql.Rows, error)` - 执行查询
- `QueryRow(query string, args ...interface{}) *sql.Row` - 执行单行查询

#### 表管理操作
- `CreateTable(tableName, schema string) error` - 创建表
- `TableExists(tableName string) (bool, error)` - 检查表是否存在
- `DropTable(tableName string) error` - 删除表
- `AddColumn(tableName, columnDef string) error` - 添加列
- `GetTableSchema(tableName string) ([]ColumnInfo, error)` - 获取表结构信息
- `BeginTx() (ITransaction, error)` - 开始事务

### 实现细节

#### baseStore 基类
- 提供所有存储实现的公共逻辑
- 包含表初始化、SQL 执行等通用方法
- 通过嵌入方式被 [LocalStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/core/store.go#L174-L174) 和 [MemStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/core/store.go#L185-L185) 复用

#### LocalStore 本地存储
- 基于 SQLite 实现持久化存储
- 数据库文件默认命名为 `duola.dat`
- 支持自定义文件路径和文件名
- 使用 `modernc.org/sqlite` 驱动，避免 CGO 依赖

#### MemStore 内存存储
- 基于内存的 SQLite 实现
- 线程安全，通过读写锁保护并发访问
- 适用于临时数据存储场景

### 数据表结构

存储模块自动创建以下数据表：

1. **kv_store** - 键值存储表
   ```sql
   CREATE TABLE IF NOT EXISTS kv_store (
       key TEXT PRIMARY KEY,
       value TEXT,
       expire_at INTEGER
   )
   ```

2. **hash_store** - 哈希存储表
   ```sql
   CREATE TABLE IF NOT EXISTS hash_store (
       key TEXT NOT NULL,
       field TEXT NOT NULL,
       value TEXT,
       PRIMARY KEY (key, field)
   )
   ```

3. **list_store** - 列表存储表
   ```sql
   CREATE TABLE IF NOT EXISTS list_store (
       key TEXT NOT NULL,
       "index" INTEGER NOT NULL,
       value TEXT,
       PRIMARY KEY (key, "index")
   )
   ```

4. **set_store** - 集合存储表
   ```sql
   CREATE TABLE IF NOT EXISTS set_store (
       key TEXT NOT NULL,
       member TEXT NOT NULL,
       PRIMARY KEY (key, member)
   )
   ```

### 使用示例

#### 基础存储操作

```go
// 创建存储实例
logs := core.NewLogger(&core.LoggerOption{
    Level: "debug",
})

store := NewStore(core.StoreOption{
    Logger: logs,
    FilePath: "/path/to/store",
    FileName: "mydata.db",
})

// 使用 KV 存储
store.Set("mykey", "myvalue")
value, _ := store.Get("mykey")

// 使用 Hash 存储
store.HSet("user:1001", "name", "Alice")
name, _ := store.HGet("user:1001", "name")

// 使用 List 存储
store.LPush("mylist", "item1", "item2")
length, _ := store.LLen("mylist")

// 使用 Set 存储
store.SAdd("myset", "member1", "member2")
members, _ := store.SMembers("myset")
```

#### 表管理操作

```go
// 创建表
err := store.CreateTable("certificates", `
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
`)

// 检查表是否存在
exists, err := store.TableExists("certificates")
if !exists {
    // 表不存在，创建表
    err = store.CreateTable("certificates", "...")
}

// 添加新列（用于数据库升级）
err = store.AddColumn("certificates", "updated_at DATETIME")

// 获取表结构信息
columns, err := store.GetTableSchema("certificates")
for _, col := range columns {
    fmt.Printf("Column: %s, Type: %s, NotNull: %t\n", 
        col.Name, col.Type, col.NotNull)
}

// 删除表
err = store.DropTable("old_table")
```

#### 事务操作

```go
// 开始事务
tx, err := store.BeginTx()
if err != nil {
    log.Fatal(err)
}

// 在事务中执行多个操作
_, err = tx.Exec("INSERT INTO certificates (name, status) VALUES (?, ?)", 
    "example.com", "active")
if err != nil {
    tx.Rollback()
    return err
}

_, err = tx.Exec("UPDATE certificates SET updated_at = ? WHERE name = ?", 
    time.Now(), "example.com")
if err != nil {
    tx.Rollback()
    return err
}

// 提交事务
err = tx.Commit()
if err != nil {
    return err
}
```

## 网络模块

网络模块提供了一个功能丰富的 HTTP 客户端，基于 Go 标准库实现。

### 主要特性
- 支持 GET、POST、PUT、DELETE 等 HTTP 方法
- 支持 Cookie 管理
- 支持请求和响应拦截器
- 支持 JSON 和表单数据发送
- 支持验证码图片获取

### 使用示例

```go
// 创建 HTTP 客户端
httpCli := core.NewHttp(logs)

// 发送 GET 请求
response, err := httpCli.Get("https://api.example.com/users", core.Options{
    Query: map[string]string{
        "page": "1",
        "size": "10",
    },
})

// 发送 POST 请求（JSON）
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

## 日志模块

日志模块基于 `github.com/labstack/gommon/log` 实现，提供统一的日志记录功能。

### 主要特性
- 支持多种日志级别（DEBUG、INFO、WARN、ERROR）
- 支持文件和控制台输出
- 支持自定义日志格式
- 线程安全

### 使用示例

```go
// 创建日志实例
logs := core.NewLogger(&core.LoggerOption{
    Type:     "file",
    FileName: "/path/to/app.log",
    Level:    "debug",
    Prefix:   "DuolaApp",
})

// 记录日志
logs.Debug("这是一条调试日志")
logs.Info("这是一条信息日志")
logs.Warn("这是一条警告日志")
logs.Error("这是一条错误日志")
```

## AppStore 封装

[AppStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/app_store.go#L11-L14) 是对核心存储模块的进一步封装，同时管理本地和内存存储实例。

### 主要特性
- 同时管理 [LocalStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/core/store.go#L174-L174) 和 [MemStore](file:///home/fzxs/workspaces/demo/duola/duola-desktop/duolasdk/core/store.go#L185-L185) 实例
- 通过 `persistent` 参数选择存储类型
- 提供统一的接口访问两种存储

### 使用示例

```go
// 创建 AppStore 实例
appStore := duolasdk.NewStore(core.StoreOption{
    Logger: logs,
})

// 使用持久化存储
appStore.Set("key", "value", true)  // 第三个参数 true 表示使用持久化存储
value, _ := appStore.Get("key", true)

// 使用内存存储
appStore.Set("key", "value", false)  // 第三个参数 false 表示使用内存存储
value, _ := appStore.Get("key", false)

// 默认使用持久化存储
appStore.Set("key", "value")  // 不传第三个参数，默认使用持久化存储
value, _ := appStore.Get("key")
```

### 数据类型定义

#### ColumnInfo 结构体

表示数据库表列的信息：

```go
type ColumnInfo struct {
    Name         string `json:"name"`         // 列名
    Type         string `json:"type"`         // 数据类型
    NotNull      bool   `json:"notNull"`      // 是否非空
    DefaultValue string `json:"defaultValue"` // 默认值
    PrimaryKey   bool   `json:"primaryKey"`   // 是否主键
}
```

#### ITransaction 接口

事务操作接口：

```go
type ITransaction interface {
    Exec(query string, args ...interface{}) (sql.Result, error)
    Query(query string, args ...interface{}) (*sql.Rows, error)
    QueryRow(query string, args ...interface{}) *sql.Row
    Commit() error
    Rollback() error
}
```

### 数据库升级与兼容性

#### 向下兼容性保证

1. **接口扩展** - 只在 IStore 接口中添加新方法，不修改现有方法
2. **现有功能不变** - 所有原有的 Redis 风格操作和 SQL 操作保持不变
3. **可选使用** - 新功能是可选的，现有代码无需修改即可继续工作
4. **安全操作** - 使用 `IF NOT EXISTS` 和 `IF EXISTS` 确保操作安全

#### 数据库升级示例

```go
// 检查并升级数据库结构
func upgradeDatabase(store core.IStore) error {
    // 检查新表是否存在
    exists, err := store.TableExists("new_feature_table")
    if err != nil {
        return err
    }
    
    if !exists {
        // 创建新表
        err = store.CreateTable("new_feature_table", `
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            feature_name TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        `)
        if err != nil {
            return err
        }
    }
    
    // 检查现有表是否需要添加新列
    columns, err := store.GetTableSchema("certificates")
    if err != nil {
        return err
    }
    
    hasUpdatedAt := false
    for _, col := range columns {
        if col.Name == "updated_at" {
            hasUpdatedAt = true
            break
        }
    }
    
    if !hasUpdatedAt {
        // 添加新列
        err = store.AddColumn("certificates", "updated_at DATETIME")
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## 最佳实践

### 存储模块使用建议
1. 对于需要持久化的数据，使用 LocalStore
2. 对于临时或会话数据，使用 MemStore
3. 合理使用过期功能管理数据生命周期
4. 注意在应用退出时调用 Close 方法释放资源
5. 使用表管理功能进行数据库结构升级
6. 在复杂操作中使用事务确保数据一致性

### 网络模块使用建议
1. 合理使用拦截器处理通用逻辑（如认证、日志等）
2. 注意处理网络错误和超时
3. 对于需要保持会话的请求，使用同一个 HttpCli 实例

### 日志模块使用建议
1. 合理设置日志级别，避免生产环境输出过多调试日志
2. 在关键路径添加日志记录，便于问题排查
3. 注意日志文件大小管理，避免占用过多磁盘空间

### 数据库设计建议
1. 使用 `INTEGER PRIMARY KEY AUTOINCREMENT` 作为主键
2. 重要字段添加 `NOT NULL` 约束
3. 使用 `DATETIME DEFAULT CURRENT_TIMESTAMP` 记录创建时间
4. 合理使用索引提高查询性能
5. 在升级时使用 `AddColumn` 而不是重建表