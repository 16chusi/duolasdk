# 存储模块 (store.go) API 文档

## 1. 核心思想

`store.go` 文件为应用程序提供了一个统一的数据存储层。其核心是 `IStore` 接口，该接口抽象并统一了所有数据操作。这使得上层业务代码无需关心底层是哪种存储介质。

该模块提供了两种具体的存储实现：

1.  **`LocalStore`**: 基于 SQLite 的 **持久化** 文件存储。数据在程序关闭后依然存在。通过 `NewLocalStore()` 创建。
2.  **`MemStore`**: 基于 SQLite 的 **内存** 存储。速度快，但数据在程序关闭后会丢失。它是线程安全的。通过 `NewMemStore()` 创建。

API 的设计风格大量借鉴了 Redis，提供了对多种数据结构（如键值对、哈希、列表、集合）的友好操作。同时，它也保留了直接执行原生 SQL 和管理表结构的能力，提供了极大的灵活性。

## 2. 配置 (StoreOption)

在创建存储实例时，可以通过 `StoreOption` 结构体进行配置：

-   `FilePath`: 数据库文件存放的目录路径（仅 `LocalStore` 有效）。
-   `FileName`: 数据库文件的名称，默认为 `duola.dat`（仅 `LocalStore` 有效）。
-   `Logger`: 用于记录内部操作的日志记录器实例。
-   `InitSQL`: 一个字符串切片，包含在数据库初始化时需要执行的自定义 SQL 语句。
-   `OnKeyExpired`: 当一个设置了过期时间的键到期时触发的回调函数。

## 3. 核心接口 `IStore`

这是所有存储操作的入口，定义了模块的核心能力。

### 3.1. 键值 (Key-Value) 操作

用于基本的字符串键值对存储。

-   `Set(key, value string) error`: 设置一个键值对。如果键已存在，则覆盖。
-   `Get(key string) (string, error)`: 根据键获取其对应的值。如果键不存在或已过期，返回 `ErrKeyNotFound` 错误。
-   `Delete(key string) error`: 根据键删除一个键值对。
-   `List() (map[string]string, error)`: 列出所有未过期的键值对。

### 3.2. 过期 (Expiration) 操作

可以为顶层键（Key-Value）设置生命周期。

-   `Expire(key string, milliseconds int64) error`: 为一个已存在的键设置过期时间（单位：毫秒）。
-   `TTL(key string) (int64, error)`: 获取一个键的剩余生存时间（单位：毫秒）。
    -   返回 `-1` 表示键存在但没有设置过期时间。
    -   返回 `-2` 表示键不存在或已过期。

### 3.3. 哈希 (Hash) 操作

用于存储结构化的对象，类似于 Redis 的 Hash。一个键可以包含多个字段和值的映射。

-   `HSet(key, field, value string) error`: 在指定 `key` 的哈希表中设置一个 `field` 和 `value`。
-   `HGet(key, field string) (string, error)`: 获取哈希表中指定 `field` 的值。
-   `HGetAll(key string) (map[string]string, error)`: 获取哈希表中所有的字段和值。
-   `HMSet(key string, fieldValue map[string]string) error`: 同时设置多个字段和值。
-   `HMGet(key string, fields ...string) ([]string, error)`: 同时获取多个字段的值。
-   `HDel(key string, fields ...string) error`: 删除一个或多个哈希字段。
-   `HExists(key, field string) (bool, error)`: 判断哈希表中是否存在指定的 `field`。
-   `HKeys(key string) ([]string, error)`: 获取哈希表中所有的字段名。
-   `HLen(key string) (int, error)`: 获取哈希表中字段的数量。

### 3.4. 列表 (List) 操作

用于存储有序的字符串序列，类似于 Redis 的 List。

-   `LPush(key string, values ...string) error`: 将一个或多个值插入到列表的 **头部**。
-   `RPush(key string, values ...string) error`: 将一个或多个值插入到列表的 **尾部**。
-   `LPop(key string) (string, error)`: 移除并返回列表的 **头** 元素。
-   `RPop(key string) (string, error)`: 移除并返回列表的 **尾** 元素。
-   `LRange(key string, start, stop int) ([]string, error)`: 获取列表中指定范围的元素。
-   `LLen(key string) (int, error)`: 获取列表的长度。

### 3.5. 集合 (Set) 操作

用于存储无序且唯一的字符串集合，类似于 Redis 的 Set。

-   `SAdd(key string, members ...string) error`: 向集合中添加一个或多个成员。
-   `SRem(key string, members ...string) error`: 从集合中移除一个或多个成员。
-   `SMembers(key string) ([]string, error)`: 返回集合中的所有成员。
-   `SIsMember(key, member string) (bool, error)`: 判断一个成员是否存在于集合中。
-   `SCard(key string) (int, error)`: 获取集合中元素的数量（基数）。

### 3.6. 原生 SQL 操作

提供了直接操作底层 SQLite 数据库的能力。

-   `Exec(query string, args ...interface{}) (sql.Result, error)`: 执行不返回结果集的 SQL 语句（如 `INSERT`, `UPDATE`, `DELETE`）。
-   `Query(query string, args ...interface{}) (*sql.Rows, error)`: 执行返回多行结果的 SQL 查询（如 `SELECT`）。
-   `QueryRow(query string, args ...interface{}) *sql.Row`: 执行最多返回一行的 SQL 查询。

### 3.7. 表管理 (Table Management) 操作

用于动态管理数据库的表结构，非常适合数据库迁移和升级场景。

-   `CreateTable(tableName string, schema string) error`: 根据指定的表名和结构定义（如 `"id INTEGER PRIMARY KEY, name TEXT"`）创建一个新表。
-   `TableExists(tableName string) (bool, error)`: 检查指定的表是否存在。
-   `DropTable(tableName string) error`: 删除一个表。
-   `AddColumn(tableName, columnDef string) error`: 向一个已存在的表添加一个新的列（如 `"age INTEGER"`）。
-   `GetTableSchema(tableName string) ([]ColumnInfo, error)`: 获取表的结构信息，返回一个 `ColumnInfo` 结构体切片。

### 3.8. 事务 (Transaction) 操作

用于执行一组需要保证原子性的数据库操作。

-   `BeginTx() (ITransaction, error)`: 开始一个事务，并返回一个 `ITransaction` 接口实例。

## 4. 辅助类型

### 4.1. `ITransaction` 接口

事务实例的接口，提供了以下方法：

-   `Exec`, `Query`, `QueryRow`: 在事务上下文中执行 SQL。
-   `Commit() error`: 提交事务，使所有更改生效。
-   `Rollback() error`: 回滚事务，撤销所有更改。

### 4.2. `ColumnInfo` 结构体

用于描述数据库表的列信息。

-   `Name`: 列名。
-   `Type`: 数据类型 (如 `TEXT`, `INTEGER`)。
-   `NotNull`: 是否有 `NOT NULL` 约束。
-   `DefaultValue`: 列的默认值。
-   `PrimaryKey`: 是否是主键。

## 5. 使用示例

### 5.1. 基础操作

```go
package main

import (
	"fmt"
	"log"

	"github.com/fzxs8/duolasdk/core"
)

func main() {
	// 创建一个日志实例
	appLogger := core.NewLogger(&core.LoggerOption{Level: "debug"})

	// 创建一个持久化存储实例
	// 你也可以使用 NewMemStore() 创建内存存储
	store, err := core.NewLocalStore(core.StoreOption{
		Logger:   appLogger,
		FilePath: "./data", // 指定数据存储目录
		FileName: "myapp", // 指定数据库文件名 (最终为 myapp.dat)
	})
	if err != nil {
		log.Fatalf("创建存储实例失败: %v", err)
	}
	defer store.Close()

	// 1. Key-Value 操作
	fmt.Println("--- Key-Value 操作 ---")
	_ = store.Set("greeting", "Hello, Duola!")
	value, _ := store.Get("greeting")
	fmt.Printf("Get('greeting'): %s
", value)
	_ = store.Expire("greeting", 1000) // 1秒后过期

	// 2. Hash 操作
	fmt.Println("
--- Hash 操作 ---")
	_ = store.HSet("user:1", "name", "Alice")
	_ = store.HSet("user:1", "age", "30")
	name, _ := store.HGet("user:1", "name")
	fmt.Printf("HGet('user:1', 'name'): %s
", name)
	allFields, _ := store.HGetAll("user:1")
	fmt.Printf("HGetAll('user:1'): %v
", allFields)

	// 3. List 操作
	fmt.Println("
--- List 操作 ---")
	_ = store.LPush("tasks", "task1", "task2")
	_ = store.RPush("tasks", "task3")
	tasks, _ := store.LRange("tasks", 0, -1)
	fmt.Printf("LRange('tasks', 0, -1): %v
", tasks)
	task, _ := store.LPop("tasks")
	fmt.Printf("LPop('tasks'): %s
", task)

	// 4. Set 操作
	fmt.Println("
--- Set 操作 ---")
	_ = store.SAdd("tags", "go", "sdk", "go") // "go" 只会添加一次
	tags, _ := store.SMembers("tags")
	fmt.Printf("SMembers('tags'): %v
", tags)
	isMember, _ := store.SIsMember("tags", "go")
	fmt.Printf("SIsMember('tags', 'go'): %t
", isMember)
}
```

### 5.2. 表管理和原生 SQL

```go
func manageTables(store core.IStore) {
	tableName := "users"
	schema := `
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    `

	// 1. 创建表
	fmt.Printf("--- 正在创建表 '%s' ---
", tableName)
	err := store.CreateTable(tableName, schema)
	if err != nil {
		log.Printf("创建表失败: %v", err)
	}

	// 2. 检查表是否存在
	exists, _ := store.TableExists(tableName)
	fmt.Printf("表 '%s' 是否存在: %t
", tableName, exists)

	// 3. 原生 SQL 插入数据
	fmt.Println("--- 正在插入数据 ---")
	res, err := store.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Bob", "bob@example.com")
	if err == nil {
		lastID, _ := res.LastInsertId()
		fmt.Printf("插入成功，ID: %d
", lastID)
	}

	// 4. 添加新列
	fmt.Println("--- 正在添加新列 'status' ---")
	err = store.AddColumn(tableName, "status TEXT DEFAULT 'active'")
	if err != nil {
		log.Printf("添加列失败: %v", err) // 如果列已存在，可能会失败
	}

	// 5. 获取表结构
	fmt.Println("--- 正在获取表结构 ---")
	columns, _ := store.GetTableSchema(tableName)
	for _, col := range columns {
		fmt.Printf("列: %s, 类型: %s, 主键: %t
", col.Name, col.Type, col.PrimaryKey)
	}
}
```

### 5.3. 事务操作

```go
func transactionExample(store core.IStore) {
	fmt.Println("
--- 事务操作 ---")
	// 开始事务
	tx, err := store.BeginTx()
	if err != nil {
		log.Fatalf("开始事务失败: %v", err)
	}

	// 在事务中执行操作
	_, err = tx.Exec("UPDATE users SET name = ? WHERE email = ?", "Robert", "bob@example.com")
	if err != nil {
		fmt.Println("更新失败，正在回滚...")
		_ = tx.Rollback()
		return
	}

	_, err = tx.Exec("INSERT INTO users (name, email) VALUES (?, ?)", "Charlie", "charlie@example.com")
	if err != nil {
		fmt.Println("插入失败，正在回滚...")
		_ = tx.Rollback()
		return
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		log.Fatalf("提交事务失败: %v", err)
	}

	fmt.Println("事务成功提交！")
}
```