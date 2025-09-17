package core

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ErrKeyNotFound 当在存储中找不到键时返回
var ErrKeyNotFound = errors.New("key not found")

// StoreOption 存储配置选项
type StoreOption struct {
	// FilePath 文件保存路径（默认程序运行路径）
	FilePath string
	// FileName 文件保存名称（默认duola.dat）
	FileName string
	// Logger 日志记录器
	Logger *AppLog
	// InitSQL 启动时执行的初始化SQL语句
	InitSQL []string
	// MaxOpenConns 设置最大打开连接数
	MaxOpenConns int
	// MaxIdleConns 设置最大空闲连接数
	MaxIdleConns int
	// OnKeyExpired 当key过期时的回调函数
	OnKeyExpired func(key, value string)
}

// IStore 定义存储接口
type IStore interface {
	// KeyValue operations
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
	List() (map[string]string, error)

	// Expiration operations
	Expire(key string, milliseconds int64) error
	TTL(key string) (int64, error)

	// Hash operations (like redis)
	HSet(key, field, value string) error
	HGet(key, field string) (string, error)
	HGetAll(key string) (map[string]string, error)
	HMGet(key string, fields ...string) ([]string, error)
	HMSet(key string, fieldValue map[string]string) error
	HDel(key string, fields ...string) error
	HExists(key, field string) (bool, error)
	HKeys(key string) ([]string, error)
	HLen(key string) (int, error)

	// List operations (like redis)
	LPush(key string, values ...string) error
	RPush(key string, values ...string) error
	LPop(key string) (string, error)
	RPop(key string) (string, error)
	LRange(key string, start, stop int) ([]string, error)
	LLen(key string) (int, error)

	// Set operations (like redis)
	SAdd(key string, members ...string) error
	SRem(key string, members ...string) error
	SMembers(key string) ([]string, error)
	SIsMember(key, member string) (bool, error)
	SCard(key string) (int, error)

	// SQL operations
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	// Close 关闭存储连接
	Close() error
}

// baseStore 基础存储实现
type baseStore struct {
	db     *sql.DB
	logger *AppLog
	// onKeyExpired 当key过期时的回调函数
	onKeyExpired func(key, value string)
}

// getDefaultInitQueries 获取默认的初始化查询
func (b *baseStore) getDefaultInitQueries() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS kv_store (
			key TEXT PRIMARY KEY,
			value TEXT,
			expire_at INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS hash_store (
			key TEXT NOT NULL,
			field TEXT NOT NULL,
			value TEXT,
			PRIMARY KEY (key, field)
		)`,
		`CREATE TABLE IF NOT EXISTS list_store (
			key TEXT NOT NULL,
			"index" INTEGER NOT NULL,
			value TEXT,
			PRIMARY KEY (key, "index")
		)`,
		`CREATE TABLE IF NOT EXISTS set_store (
			key TEXT NOT NULL,
			member TEXT NOT NULL,
			PRIMARY KEY (key, member)
		)`,
	}
}

// initializeTables 初始化表结构
func (b *baseStore) initializeTables(initQueries []string) error {
	for _, query := range initQueries {
		if b.logger != nil {
			b.logger.Debugf("Store: executing init query: %s", query)
		}
		_, err := b.db.Exec(query)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store: failed to execute init query '%s': %v", query, err)
			}
			return err
		}
	}
	return nil
}

// LocalStore 本地存储
type LocalStore struct {
	*baseStore
}

// Ensure that LocalStore implements IStore interface
var _ IStore = (*LocalStore)(nil)

// MemStore 内存存储
type MemStore struct {
	*baseStore
	mutex sync.RWMutex
}

// Ensure that MemStore implements IStore interface
var _ IStore = (*MemStore)(nil)

// NewLocalStore 创建一个新的本地存储实例
func NewLocalStore(opts ...StoreOption) (*LocalStore, error) {
	// 默认配置
	option := StoreOption{
		FileName:     "duola.dat",
		MaxOpenConns: 1,
		MaxIdleConns: 0,
	}

	// 应用传入的配置
	if len(opts) > 0 {
		option = opts[0]
	}

	// 获取当前工作目录
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// 如果指定了文件路径，则使用指定路径，否则使用当前工作目录
	if option.FilePath != "" {
		dir = option.FilePath
	}

	// 构建数据库文件路径
	dbPath := filepath.Join(dir, "data", fmt.Sprintf("%s.dat", option.FileName))

	// 记录日志
	if option.Logger != nil {
		option.Logger.Infof("LocalStore: opening database at %s", dbPath)
	}

	// 打开 SQLite 数据库
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		if option.Logger != nil {
			option.Logger.Errorf("LocalStore: failed to open database: %v", err)
		}
		return nil, err
	}

	// 设置连接池参数
	if option.MaxOpenConns > 0 {
		db.SetMaxOpenConns(option.MaxOpenConns)
	}
	if option.MaxIdleConns > 0 {
		db.SetMaxIdleConns(option.MaxIdleConns)
	}

	// 创建默认的 KV 表
	base := &baseStore{db: db, logger: option.Logger, onKeyExpired: option.OnKeyExpired}
	err = base.initializeTables(base.getDefaultInitQueries())
	if err != nil {
		if option.Logger != nil {
			option.Logger.Errorf("LocalStore: failed to initialize tables: %v", err)
		}
		db.Close()
		return nil, err
	}

	// 执行用户自定义的初始化SQL语句
	for _, queryStmt := range option.InitSQL {
		if option.Logger != nil {
			option.Logger.Infof("LocalStore: executing init SQL: %s", queryStmt)
		}
		_, err = db.Exec(queryStmt)
		if err != nil {
			if option.Logger != nil {
				option.Logger.Errorf("LocalStore: failed to execute init SQL '%s': %v", queryStmt, err)
			}
			db.Close()
			return nil, err
		}
	}

	store := &LocalStore{
		baseStore: &baseStore{
			db:           db,
			logger:       option.Logger,
			onKeyExpired: option.OnKeyExpired,
		},
	}

	if option.Logger != nil {
		option.Logger.Info("LocalStore: successfully created")
	}

	return store, nil
}

// NewMemStore 创建一个新的内存存储实例
func NewMemStore(opts ...StoreOption) (*MemStore, error) {
	// 默认配置
	option := StoreOption{
		MaxOpenConns: 1,
		MaxIdleConns: 0,
	}

	// 应用传入的配置
	if len(opts) > 0 {
		option = opts[0]
	}

	// 记录日志
	if option.Logger != nil {
		option.Logger.Info("MemStore: creating in-memory database")
	}

	// 使用内存模式打开 SQLite 数据库
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		if option.Logger != nil {
			option.Logger.Errorf("MemStore: failed to open in-memory database: %v", err)
		}
		return nil, err
	}

	// 设置连接池参数
	if option.MaxOpenConns > 0 {
		db.SetMaxOpenConns(option.MaxOpenConns)
	}
	if option.MaxIdleConns > 0 {
		db.SetMaxIdleConns(option.MaxIdleConns)
	}

	// 创建默认的 KV 表
	base := &baseStore{db: db, logger: option.Logger, onKeyExpired: option.OnKeyExpired}
	err = base.initializeTables(base.getDefaultInitQueries())
	if err != nil {
		if option.Logger != nil {
			option.Logger.Errorf("MemStore: failed to initialize tables: %v", err)
		}
		db.Close()
		return nil, err
	}

	// 执行用户自定义的初始化SQL语句
	for _, queryStmt := range option.InitSQL {
		if option.Logger != nil {
			option.Logger.Infof("MemStore: executing init SQL: %s", queryStmt)
		}
		_, err = db.Exec(queryStmt)
		if err != nil {
			if option.Logger != nil {
				option.Logger.Errorf("MemStore: failed to execute init SQL '%s': %v", queryStmt, err)
			}
			db.Close()
			return nil, err
		}
	}

	store := &MemStore{
		baseStore: &baseStore{
			db:           db,
			logger:       option.Logger,
			onKeyExpired: option.OnKeyExpired,
		},
	}

	if option.Logger != nil {
		option.Logger.Info("MemStore: successfully created")
	}

	return store, nil
}

// Set 在存储中设置键值对
func (b *baseStore) Set(key, value string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.Set: setting key=%s", key)
	}

	_, err := b.db.Exec(`
		INSERT OR REPLACE INTO kv_store (key, value, expire_at) VALUES (?, ?, NULL)
	`, key, value)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.Set: failed to set key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.Set: successfully set key=%s", key)
	}

	return nil
}

// Get 从存储中获取值
func (b *baseStore) Get(key string) (string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.Get: getting key=%s", key)
	}

	// 检查键是否过期
	expired, err := b.checkKeyExpired(key)
	if err != nil && err != ErrKeyNotFound {
		if b.logger != nil {
			b.logger.Errorf("Store.Get: failed to check expiration for key=%s: %v", key, err)
		}
		return "", err
	}

	// 如果键已过期或不存在
	if expired || err == ErrKeyNotFound {
		if b.logger != nil {
			b.logger.Debugf("Store.Get: key=%s not found or expired", key)
		}
		return "", ErrKeyNotFound
	}

	var value string
	err = b.db.QueryRow(`
		SELECT value FROM kv_store WHERE key = ?
	`, key).Scan(&value)

	if err != nil {
		if err == sql.ErrNoRows {
			if b.logger != nil {
				b.logger.Debugf("Store.Get: key=%s not found", key)
			}
			return "", ErrKeyNotFound
		}
		if b.logger != nil {
			b.logger.Errorf("Store.Get: failed to get key=%s: %v", key, err)
		}
		return "", err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.Get: successfully got key=%s", key)
	}

	return value, nil
}

// Delete 从存储中删除键值对
func (b *baseStore) Delete(key string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.Delete: deleting key=%s", key)
	}

	_, err := b.db.Exec(`
		DELETE FROM kv_store WHERE key = ?
	`, key)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.Delete: failed to delete key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.Delete: successfully deleted key=%s", key)
	}

	return nil
}

// List 列出存储中的所有键值对
func (b *baseStore) List() (map[string]string, error) {
	if b.logger != nil {
		b.logger.Debug("Store.List: listing all keys")
	}

	rows, err := b.db.Query(`
		SELECT key, value, expire_at FROM kv_store
	`)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.List: failed to query kv_store: %v", err)
		}
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var key, value string
		var expireAt sql.NullInt64
		if err := rows.Scan(&key, &value, &expireAt); err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.List: failed to scan row: %v", err)
			}
			return nil, err
		}

		// 检查键是否过期
		if expireAt.Valid {
			now := time.Now().UnixNano() / int64(time.Millisecond)
			if now >= expireAt.Int64 {
				// 键已过期，删除它
				_, err = b.db.Exec(`DELETE FROM kv_store WHERE key = ?`, key)
				if err != nil && b.logger != nil {
					b.logger.Errorf("Store.List: failed to delete expired key %s: %v", key, err)
				}
				// 触发过期回调
				if b.onKeyExpired != nil {
					b.onKeyExpired(key, value)
				}
				continue
			}
		}

		result[key] = value
	}

	if b.logger != nil {
		b.logger.Debugf("Store.List: found %d keys", len(result))
	}

	return result, nil
}

// checkKeyExpired 检查键是否过期，如果过期则删除
func (b *baseStore) checkKeyExpired(key string) (bool, error) {
	var expireAt sql.NullInt64
	var value string
	err := b.db.QueryRow(`
		SELECT value, expire_at FROM kv_store WHERE key = ?
	`, key).Scan(&value, &expireAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrKeyNotFound
		}
		return false, err
	}

	// 如果没有设置过期时间，键未过期
	if !expireAt.Valid {
		return false, nil
	}

	// 检查是否过期
	now := time.Now().UnixNano() / int64(time.Millisecond)
	if now >= expireAt.Int64 {
		// 键已过期，删除它
		_, err = b.db.Exec(`DELETE FROM kv_store WHERE key = ?`, key)
		if err != nil {
			return false, err
		}

		// 触发过期回调
		if b.onKeyExpired != nil {
			b.onKeyExpired(key, value)
		}

		return true, nil // 键已过期
	}

	return false, nil // 键未过期
}

// Expire 设置键的过期时间（毫秒）
func (b *baseStore) Expire(key string, milliseconds int64) error {
	// 检查键是否存在
	var count int
	err := b.db.QueryRow(`
		SELECT COUNT(*) FROM kv_store WHERE key = ?
	`, key).Scan(&count)

	if err != nil {
		return err
	}

	if count == 0 {
		return ErrKeyNotFound
	}

	// 计算过期时间戳
	expireAt := time.Now().UnixNano()/int64(time.Millisecond) + milliseconds

	// 更新键的过期时间
	_, err = b.db.Exec(`
		UPDATE kv_store SET expire_at = ? WHERE key = ?
	`, expireAt, key)

	return err
}

// TTL 获取键的剩余生存时间（毫秒），-1表示没有设置过期时间，-2表示键不存在或已过期
func (b *baseStore) TTL(key string) (int64, error) {
	var expireAt sql.NullInt64
	var value string
	err := b.db.QueryRow(`
		SELECT value, expire_at FROM kv_store WHERE key = ?
	`, key).Scan(&value, &expireAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return -2, nil // 键不存在
		}
		return 0, err
	}

	// 如果没有设置过期时间
	if !expireAt.Valid {
		return -1, nil // 没有设置过期时间
	}

	now := time.Now().UnixNano() / int64(time.Millisecond)

	// 检查是否已经过期
	if now >= expireAt.Int64 {
		// 删除过期的键
		_, err = b.db.Exec(`DELETE FROM kv_store WHERE key = ?`, key)
		if err != nil {
			return 0, err
		}

		// 触发过期回调
		if b.onKeyExpired != nil {
			b.onKeyExpired(key, value)
		}

		return -2, nil // 键已过期
	}

	// 返回剩余时间（毫秒）
	return expireAt.Int64 - now, nil
}

// HSet 设置哈希表key中字段field的值
func (b *baseStore) HSet(key, field, value string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.HSet: setting key=%s, field=%s", key, field)
	}

	_, err := b.db.Exec(`
		INSERT OR REPLACE INTO hash_store (key, field, value) VALUES (?, ?, ?)
	`, key, field, value)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HSet: failed to set key=%s, field=%s: %v", key, field, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HSet: successfully set key=%s, field=%s", key, field)
	}

	return nil
}

// HGet 获取哈希表key中字段field的值
func (b *baseStore) HGet(key, field string) (string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.HGet: getting key=%s, field=%s", key, field)
	}

	var value string
	err := b.db.QueryRow(`
		SELECT value FROM hash_store WHERE key = ? AND field = ?
	`, key, field).Scan(&value)

	if err != nil {
		if err == sql.ErrNoRows {
			if b.logger != nil {
				b.logger.Debugf("Store.HGet: key=%s, field=%s not found", key, field)
			}
			return "", ErrKeyNotFound
		}
		if b.logger != nil {
			b.logger.Errorf("Store.HGet: failed to get key=%s, field=%s: %v", key, field, err)
		}
		return "", err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HGet: successfully got key=%s, field=%s", key, field)
	}

	return value, nil
}

// HGetAll 获取哈希表key中所有的字段和值
func (b *baseStore) HGetAll(key string) (map[string]string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.HGetAll: getting all fields for key=%s", key)
	}

	rows, err := b.db.Query(`
		SELECT field, value FROM hash_store WHERE key = ?
	`, key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HGetAll: failed to query hash_store for key=%s: %v", key, err)
		}
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var field, value string
		if err := rows.Scan(&field, &value); err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.HGetAll: failed to scan row: %v", err)
			}
			return nil, err
		}
		result[field] = value
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HGetAll: found %d fields for key=%s", len(result), key)
	}

	return result, nil
}

// HMGet 获取哈希表key中一个或多个字段的值
func (b *baseStore) HMGet(key string, fields ...string) ([]string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.HMGet: getting %d fields for key=%s", len(fields), key)
	}

	if len(fields) == 0 {
		return []string{}, nil
	}

	// 构建查询语句
	placeholders := make([]string, len(fields))
	args := make([]interface{}, len(fields)+1)
	args[0] = key

	for i, field := range fields {
		placeholders[i] = "?"
		args[i+1] = field
	}

	query := `
		SELECT field, value FROM hash_store 
		WHERE key = ? AND field IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY CASE field `

	for i, field := range fields {
		query += fmt.Sprintf("WHEN '%s' THEN %d ", field, i)
	}
	query += "END"

	rows, err := b.db.Query(query, args...)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HMGet: failed to query hash_store for key=%s: %v", key, err)
		}
		return nil, err
	}
	defer rows.Close()

	// 创建结果映射
	resultMap := make(map[string]string)
	for rows.Next() {
		var field, value string
		if err := rows.Scan(&field, &value); err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.HMGet: failed to scan row: %v", err)
			}
			return nil, err
		}
		resultMap[field] = value
	}

	// 按照请求的字段顺序构建结果
	result := make([]string, len(fields))
	for i, field := range fields {
		if val, exists := resultMap[field]; exists {
			result[i] = val
		} else {
			result[i] = ""
		}
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HMGet: successfully got %d fields for key=%s", len(result), key)
	}

	return result, nil
}

// HMSet 同时设置哈希表key中一个或多个字段的值
func (b *baseStore) HMSet(key string, fieldValue map[string]string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.HMSet: setting %d fields for key=%s", len(fieldValue), key)
	}

	tx, err := b.db.Begin()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HMSet: failed to begin transaction for key=%s: %v", key, err)
		}
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO hash_store (key, field, value) VALUES (?, ?, ?)
	`)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HMSet: failed to prepare statement for key=%s: %v", key, err)
		}
		return err
	}
	defer stmt.Close()

	for field, value := range fieldValue {
		_, err := stmt.Exec(key, field, value)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.HMSet: failed to set field=%s for key=%s: %v", field, key, err)
			}
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HMSet: failed to commit transaction for key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HMSet: successfully set %d fields for key=%s", len(fieldValue), key)
	}

	return nil
}

// HDel 删除哈希表key中的一个或多个字段
func (b *baseStore) HDel(key string, fields ...string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.HDel: deleting %d fields from key=%s", len(fields), key)
	}

	if len(fields) == 0 {
		return nil
	}

	// 构建删除语句
	placeholders := make([]string, len(fields))
	args := make([]interface{}, len(fields)+1)
	args[0] = key

	for i, field := range fields {
		placeholders[i] = "?"
		args[i+1] = field
	}

	query := `
		DELETE FROM hash_store 
		WHERE key = ? AND field IN (` + strings.Join(placeholders, ",") + `)
	`

	_, err := b.db.Exec(query, args...)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HDel: failed to delete fields from key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HDel: successfully deleted fields from key=%s", key)
	}

	return nil
}

// HExists 检查哈希表key中是否存在字段field
func (b *baseStore) HExists(key, field string) (bool, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.HExists: checking if field=%s exists in key=%s", field, key)
	}

	var count int
	err := b.db.QueryRow(`
		SELECT COUNT(*) FROM hash_store WHERE key = ? AND field = ?
	`, key, field).Scan(&count)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HExists: failed to check field=%s in key=%s: %v", field, key, err)
		}
		return false, err
	}

	exists := count > 0

	if b.logger != nil {
		b.logger.Debugf("Store.HExists: field=%s exists in key=%s: %t", field, key, exists)
	}

	return exists, nil
}

// HKeys 获取哈希表key中的所有字段名
func (b *baseStore) HKeys(key string) ([]string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.HKeys: getting all field names for key=%s", key)
	}

	rows, err := b.db.Query(`
		SELECT field FROM hash_store WHERE key = ? ORDER BY field
	`, key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HKeys: failed to query field names for key=%s: %v", key, err)
		}
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.HKeys: failed to scan field name: %v", err)
			}
			return nil, err
		}
		result = append(result, field)
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HKeys: found %d field names for key=%s", len(result), key)
	}

	return result, nil
}

// HLen 获取哈希表key中字段的数量
func (b *baseStore) HLen(key string) (int, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.HLen: getting field count for key=%s", key)
	}

	var count int
	err := b.db.QueryRow(`
		SELECT COUNT(*) FROM hash_store WHERE key = ?
	`, key).Scan(&count)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.HLen: failed to get field count for key=%s: %v", key, err)
		}
		return 0, err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.HLen: field count for key=%s is %d", key, count)
	}

	return count, nil
}

// LPush 将一个或多个值插入到列表key的表头
func (b *baseStore) LPush(key string, values ...string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.LPush: pushing %d values to key=%s", len(values), key)
	}

	if len(values) == 0 {
		return nil
	}

	tx, err := b.db.Begin()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPush: failed to begin transaction for key=%s: %v", key, err)
		}
		return err
	}
	defer tx.Rollback()

	// 更新现有元素的索引
	_, err = tx.Exec(`
		UPDATE list_store SET "index" = "index" + ? WHERE key = ?
	`, len(values), key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPush: failed to update indices for key=%s: %v", key, err)
		}
		return err
	}

	// 插入新元素
	stmt, err := tx.Prepare(`
		INSERT INTO list_store (key, "index", value) VALUES (?, ?, ?)
	`)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPush: failed to prepare statement for key=%s: %v", key, err)
		}
		return err
	}
	defer stmt.Close()

	for i, value := range values {
		_, err := stmt.Exec(key, len(values)-1-i, value)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.LPush: failed to insert value at index %d for key=%s: %v", len(values)-1-i, key, err)
			}
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPush: failed to commit transaction for key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.LPush: successfully pushed %d values to key=%s", len(values), key)
	}

	return nil
}

// RPush 将一个或多个值插入到列表key的表尾
func (b *baseStore) RPush(key string, values ...string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.RPush: pushing %d values to key=%s", len(values), key)
	}

	if len(values) == 0 {
		return nil
	}

	tx, err := b.db.Begin()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPush: failed to begin transaction for key=%s: %v", key, err)
		}
		return err
	}
	defer tx.Rollback()

	// 获取当前列表长度
	var length int
	err = tx.QueryRow(`
		SELECT COALESCE(MAX("index"), -1) + 1 FROM list_store WHERE key = ?
	`, key).Scan(&length)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPush: failed to get list length for key=%s: %v", key, err)
		}
		return err
	}

	// 插入新元素
	stmt, err := tx.Prepare(`
		INSERT INTO list_store (key, "index", value) VALUES (?, ?, ?)
	`)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPush: failed to prepare statement for key=%s: %v", key, err)
		}
		return err
	}
	defer stmt.Close()

	for i, value := range values {
		_, err := stmt.Exec(key, length+i, value)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.RPush: failed to insert value at index %d for key=%s: %v", length+i, key, err)
			}
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPush: failed to commit transaction for key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.RPush: successfully pushed %d values to key=%s", len(values), key)
	}

	return nil
}

// LPop 移除并返回列表key的头元素
func (b *baseStore) LPop(key string) (string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.LPop: popping value from key=%s", key)
	}

	tx, err := b.db.Begin()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPop: failed to begin transaction for key=%s: %v", key, err)
		}
		return "", err
	}
	defer tx.Rollback()

	var value string
	err = tx.QueryRow(`
		SELECT value FROM list_store WHERE key = ? ORDER BY "index" LIMIT 1
	`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			if b.logger != nil {
				b.logger.Debugf("Store.LPop: key=%s is empty", key)
			}
			return "", ErrKeyNotFound
		}
		if b.logger != nil {
			b.logger.Errorf("Store.LPop: failed to get head element for key=%s: %v", key, err)
		}
		return "", err
	}

	// 删除头元素
	_, err = tx.Exec(`
		DELETE FROM list_store WHERE key = ? AND "index" = (
			SELECT "index" FROM list_store WHERE key = ? ORDER BY "index" LIMIT 1
		)
	`, key, key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPop: failed to delete head element for key=%s: %v", key, err)
		}
		return "", err
	}

	// 更新索引
	_, err = tx.Exec(`
		UPDATE list_store SET "index" = "index" - 1 WHERE key = ?
	`, key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPop: failed to update indices for key=%s: %v", key, err)
		}
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LPop: failed to commit transaction for key=%s: %v", key, err)
		}
		return "", err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.LPop: successfully popped value from key=%s", key)
	}

	return value, nil
}

// RPop 移除并返回列表key的尾元素
func (b *baseStore) RPop(key string) (string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.RPop: popping value from key=%s", key)
	}

	tx, err := b.db.Begin()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPop: failed to begin transaction for key=%s: %v", key, err)
		}
		return "", err
	}
	defer tx.Rollback()

	var value string
	err = tx.QueryRow(`
		SELECT value FROM list_store WHERE key = ? ORDER BY "index" DESC LIMIT 1
	`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			if b.logger != nil {
				b.logger.Debugf("Store.RPop: key=%s is empty", key)
			}
			return "", ErrKeyNotFound
		}
		if b.logger != nil {
			b.logger.Errorf("Store.RPop: failed to get tail element for key=%s: %v", key, err)
		}
		return "", err
	}

	// 删除尾元素
	_, err = tx.Exec(`
		DELETE FROM list_store WHERE key = ? AND "index" = (
			SELECT "index" FROM list_store WHERE key = ? ORDER BY "index" DESC LIMIT 1
		)
	`, key, key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPop: failed to delete tail element for key=%s: %v", key, err)
		}
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.RPop: failed to commit transaction for key=%s: %v", key, err)
		}
		return "", err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.RPop: successfully popped value from key=%s", key)
	}

	return value, nil
}

// LRange 返回列表key中指定区间内的元素
func (b *baseStore) LRange(key string, start, stop int) ([]string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.LRange: getting range [%d, %d] for key=%s", start, stop, key)
	}

	// 处理负数索引
	var count int
	err := b.db.QueryRow(`
		SELECT COALESCE(MAX("index"), -1) + 1 FROM list_store WHERE key = ?
	`, key).Scan(&count)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LRange: failed to get list length for key=%s: %v", key, err)
		}
		return nil, err
	}

	if start < 0 {
		start = count + start
	}
	if stop < 0 {
		stop = count + stop
	}

	if start < 0 {
		start = 0
	}
	if stop >= count {
		stop = count - 1
	}
	if start > stop || count == 0 {
		return []string{}, nil
	}

	rows, err := b.db.Query(`
		SELECT value FROM list_store WHERE key = ? AND "index" BETWEEN ? AND ? ORDER BY "index"
	`, key, start, stop)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LRange: failed to query list elements for key=%s: %v", key, err)
		}
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.LRange: failed to scan element: %v", err)
			}
			return nil, err
		}
		result = append(result, value)
	}

	if b.logger != nil {
		b.logger.Debugf("Store.LRange: found %d elements for key=%s", len(result), key)
	}

	return result, nil
}

// LLen 返回列表key的长度
func (b *baseStore) LLen(key string) (int, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.LLen: getting length for key=%s", key)
	}

	var count int
	err := b.db.QueryRow(`
		SELECT COALESCE(MAX("index"), -1) + 1 FROM list_store WHERE key = ?
	`, key).Scan(&count)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.LLen: failed to get list length for key=%s: %v", key, err)
		}
		return 0, err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.LLen: length for key=%s is %d", key, count)
	}

	return count, nil
}

// SAdd 将一个或多个成员加入到集合key中
func (b *baseStore) SAdd(key string, members ...string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.SAdd: adding %d members to key=%s", len(members), key)
	}

	if len(members) == 0 {
		return nil
	}

	tx, err := b.db.Begin()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SAdd: failed to begin transaction for key=%s: %v", key, err)
		}
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO set_store (key, member) VALUES (?, ?)
	`)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SAdd: failed to prepare statement for key=%s: %v", key, err)
		}
		return err
	}
	defer stmt.Close()

	for _, member := range members {
		_, err := stmt.Exec(key, member)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.SAdd: failed to add member=%s to key=%s: %v", member, key, err)
			}
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SAdd: failed to commit transaction for key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.SAdd: successfully added members to key=%s", key)
	}

	return nil
}

// SRem 从集合key中移除一个或多个成员
func (b *baseStore) SRem(key string, members ...string) error {
	if b.logger != nil {
		b.logger.Debugf("Store.SRem: removing %d members from key=%s", len(members), key)
	}

	if len(members) == 0 {
		return nil
	}

	// 构建删除语句
	placeholders := make([]string, len(members))
	args := make([]interface{}, len(members)+1)
	args[0] = key

	for i, member := range members {
		placeholders[i] = "?"
		args[i+1] = member
	}

	query := `
		DELETE FROM set_store 
		WHERE key = ? AND member IN (` + strings.Join(placeholders, ",") + `)
	`

	_, err := b.db.Exec(query, args...)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SRem: failed to remove members from key=%s: %v", key, err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.SRem: successfully removed members from key=%s", key)
	}

	return nil
}

// SMembers 返回集合key中的所有成员
func (b *baseStore) SMembers(key string) ([]string, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.SMembers: getting all members for key=%s", key)
	}

	rows, err := b.db.Query(`
		SELECT member FROM set_store WHERE key = ? ORDER BY member
	`, key)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SMembers: failed to query members for key=%s: %v", key, err)
		}
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var member string
		if err := rows.Scan(&member); err != nil {
			if b.logger != nil {
				b.logger.Errorf("Store.SMembers: failed to scan member: %v", err)
			}
			return nil, err
		}
		result = append(result, member)
	}

	if b.logger != nil {
		b.logger.Debugf("Store.SMembers: found %d members for key=%s", len(result), key)
	}

	return result, nil
}

// SIsMember 判断member是否是集合key的成员
func (b *baseStore) SIsMember(key, member string) (bool, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.SIsMember: checking if member=%s is in key=%s", member, key)
	}

	var count int
	err := b.db.QueryRow(`
		SELECT COUNT(*) FROM set_store WHERE key = ? AND member = ?
	`, key, member).Scan(&count)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SIsMember: failed to check member=%s in key=%s: %v", member, key, err)
		}
		return false, err
	}

	isMember := count > 0

	if b.logger != nil {
		b.logger.Debugf("Store.SIsMember: member=%s is in key=%s: %t", member, key, isMember)
	}

	return isMember, nil
}

// SCard 返回集合key的基数(集合中元素的数量)
func (b *baseStore) SCard(key string) (int, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.SCard: getting cardinality for key=%s", key)
	}

	var count int
	err := b.db.QueryRow(`
		SELECT COUNT(*) FROM set_store WHERE key = ?
	`, key).Scan(&count)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.SCard: failed to get cardinality for key=%s: %v", key, err)
		}
		return 0, err
	}

	if b.logger != nil {
		b.logger.Debugf("Store.SCard: cardinality for key=%s is %d", key, count)
	}

	return count, nil
}

// Exec 在存储上执行 SQL 语句
func (b *baseStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.Exec: executing query: %s, args: %v", query, args)
	}

	result, err := b.db.Exec(query, args...)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.Exec: failed to execute query '%s': %v", query, err)
		}
		return nil, err
	}

	if b.logger != nil {
		if rows, err := result.RowsAffected(); err == nil {
			b.logger.Debugf("Store.Exec: query executed successfully, rows affected: %d", rows)
		}
	}

	return result, nil
}

// Query 在存储上执行查询
func (b *baseStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if b.logger != nil {
		b.logger.Debugf("Store.Query: executing query: %s, args: %v", query, args)
	}

	rows, err := b.db.Query(query, args...)

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.Query: failed to execute query '%s': %v", query, err)
		}
		return nil, err
	}

	if b.logger != nil {
		b.logger.Debug("Store.Query: query executed successfully")
	}

	return rows, nil
}

// QueryRow 在存储上执行单行查询
func (b *baseStore) QueryRow(query string, args ...interface{}) *sql.Row {
	if b.logger != nil {
		b.logger.Debugf("Store.QueryRow: executing query: %s, args: %v", query, args)
	}

	row := b.db.QueryRow(query, args...)

	if b.logger != nil {
		b.logger.Debug("Store.QueryRow: query executed")
	}

	return row
}

// Close 关闭存储连接
func (b *baseStore) Close() error {
	if b.logger != nil {
		b.logger.Debug("Store.Close: closing database connection")
	}

	err := b.db.Close()

	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("Store.Close: failed to close database connection: %v", err)
		}
		return err
	}

	if b.logger != nil {
		b.logger.Debug("Store.Close: database connection closed successfully")
	}

	return nil
}

// Set 在本地存储中设置键值对
func (l *LocalStore) Set(key, value string) error {
	return l.baseStore.Set(key, value)
}

// Get 从本地存储中获取值
func (l *LocalStore) Get(key string) (string, error) {
	return l.baseStore.Get(key)
}

// Delete 从本地存储中删除键值对
func (l *LocalStore) Delete(key string) error {
	return l.baseStore.Delete(key)
}

// List 列出本地存储中的所有键值对
func (l *LocalStore) List() (map[string]string, error) {
	return l.baseStore.List()
}

// Close 关闭本地存储连接
func (l *LocalStore) Close() error {
	return l.baseStore.Close()
}

// Expire 设置键的过期时间（毫秒）
func (l *LocalStore) Expire(key string, milliseconds int64) error {
	return l.baseStore.Expire(key, milliseconds)
}

// TTL 获取键的剩余生存时间（毫秒）
func (l *LocalStore) TTL(key string) (int64, error) {
	return l.baseStore.TTL(key)
}

// HSet 设置哈希表key中字段field的值
func (l *LocalStore) HSet(key, field, value string) error {
	return l.baseStore.HSet(key, field, value)
}

// HGet 获取哈希表key中字段field的值
func (l *LocalStore) HGet(key, field string) (string, error) {
	return l.baseStore.HGet(key, field)
}

// HGetAll 获取哈希表key中所有的字段和值
func (l *LocalStore) HGetAll(key string) (map[string]string, error) {
	return l.baseStore.HGetAll(key)
}

// HMGet 获取哈希表key中一个或多个字段的值
func (l *LocalStore) HMGet(key string, fields ...string) ([]string, error) {
	return l.baseStore.HMGet(key, fields...)
}

// HMSet 同时设置哈希表key中一个或多个字段的值
func (l *LocalStore) HMSet(key string, fieldValue map[string]string) error {
	return l.baseStore.HMSet(key, fieldValue)
}

// HDel 删除哈希表key中的一个或多个字段
func (l *LocalStore) HDel(key string, fields ...string) error {
	return l.baseStore.HDel(key, fields...)
}

// HExists 检查哈希表key中是否存在字段field
func (l *LocalStore) HExists(key, field string) (bool, error) {
	return l.baseStore.HExists(key, field)
}

// HKeys 获取哈希表key中的所有字段名
func (l *LocalStore) HKeys(key string) ([]string, error) {
	return l.baseStore.HKeys(key)
}

// HLen 获取哈希表key中字段的数量
func (l *LocalStore) HLen(key string) (int, error) {
	return l.baseStore.HLen(key)
}

// LPush 将一个或多个值插入到列表key的表头
func (l *LocalStore) LPush(key string, values ...string) error {
	return l.baseStore.LPush(key, values...)
}

// RPush 将一个或多个值插入到列表key的表尾
func (l *LocalStore) RPush(key string, values ...string) error {
	return l.baseStore.RPush(key, values...)
}

// LPop 移除并返回列表key的头元素
func (l *LocalStore) LPop(key string) (string, error) {
	return l.baseStore.LPop(key)
}

// RPop 移除并返回列表key的尾元素
func (l *LocalStore) RPop(key string) (string, error) {
	return l.baseStore.RPop(key)
}

// LRange 返回列表key中指定区间内的元素
func (l *LocalStore) LRange(key string, start, stop int) ([]string, error) {
	return l.baseStore.LRange(key, start, stop)
}

// LLen 返回列表key的长度
func (l *LocalStore) LLen(key string) (int, error) {
	return l.baseStore.LLen(key)
}

// SAdd 将一个或多个成员加入到集合key中
func (l *LocalStore) SAdd(key string, members ...string) error {
	return l.baseStore.SAdd(key, members...)
}

// SRem 从集合key中移除一个或多个成员
func (l *LocalStore) SRem(key string, members ...string) error {
	return l.baseStore.SRem(key, members...)
}

// SMembers 返回集合key中的所有成员
func (l *LocalStore) SMembers(key string) ([]string, error) {
	return l.baseStore.SMembers(key)
}

// SIsMember 判断member是否是集合key的成员
func (l *LocalStore) SIsMember(key, member string) (bool, error) {
	return l.baseStore.SIsMember(key, member)
}

// SCard 返回集合key的基数(集合中元素的数量)
func (l *LocalStore) SCard(key string) (int, error) {
	return l.baseStore.SCard(key)
}

// Exec 在本地存储上执行 SQL 语句
func (l *LocalStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	return l.baseStore.Exec(query, args...)
}

// Query 在本地存储上执行查询
func (l *LocalStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return l.baseStore.Query(query, args...)
}

// QueryRow 在本地存储上执行单行查询
func (l *LocalStore) QueryRow(query string, args ...interface{}) *sql.Row {
	return l.baseStore.QueryRow(query, args...)
}

// Set 在内存存储中设置键值对
func (m *MemStore) Set(key, value string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.Set(key, value)
}

// Get 从内存存储中获取值
func (m *MemStore) Get(key string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.Get(key)
}

// Delete 从内存存储中删除键值对
func (m *MemStore) Delete(key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.Delete(key)
}

// List 列出内存存储中的所有键值对
func (m *MemStore) List() (map[string]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.List()
}

// HSet 设置哈希表key中字段field的值
func (m *MemStore) HSet(key, field, value string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.HSet(key, field, value)
}

// HGet 获取哈希表key中字段field的值
func (m *MemStore) HGet(key, field string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.HGet(key, field)
}

// HGetAll 获取哈希表key中所有的字段和值
func (m *MemStore) HGetAll(key string) (map[string]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.HGetAll(key)
}

// HMGet 获取哈希表key中一个或多个字段的值
func (m *MemStore) HMGet(key string, fields ...string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.HMGet(key, fields...)
}

// HMSet 同时设置哈希表key中一个或多个字段的值
func (m *MemStore) HMSet(key string, fieldValue map[string]string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.HMSet(key, fieldValue)
}

// HDel 删除哈希表key中的一个或多个字段
func (m *MemStore) HDel(key string, fields ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.HDel(key, fields...)
}

// HExists 检查哈希表key中是否存在字段field
func (m *MemStore) HExists(key, field string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.HExists(key, field)
}

// HKeys 获取哈希表key中的所有字段名
func (m *MemStore) HKeys(key string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.HKeys(key)
}

// HLen 获取哈希表key中字段的数量
func (m *MemStore) HLen(key string) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.HLen(key)
}

// LPush 将一个或多个值插入到列表key的表头
func (m *MemStore) LPush(key string, values ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.LPush(key, values...)
}

// RPush 将一个或多个值插入到列表key的表尾
func (m *MemStore) RPush(key string, values ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.RPush(key, values...)
}

// LPop 移除并返回列表key的头元素
func (m *MemStore) LPop(key string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.LPop(key)
}

// RPop 移除并返回列表key的尾元素
func (m *MemStore) RPop(key string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.RPop(key)
}

// LRange 返回列表key中指定区间内的元素
func (m *MemStore) LRange(key string, start, stop int) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.LRange(key, start, stop)
}

// LLen 返回列表key的长度
func (m *MemStore) LLen(key string) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.LLen(key)
}

// SAdd 将一个或多个成员加入到集合key中
func (m *MemStore) SAdd(key string, members ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.SAdd(key, members...)
}

// SRem 从集合key中移除一个或多个成员
func (m *MemStore) SRem(key string, members ...string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.SRem(key, members...)
}

// SMembers 返回集合key中的所有成员
func (m *MemStore) SMembers(key string) ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.SMembers(key)
}

// SIsMember 判断member是否是集合key的成员
func (m *MemStore) SIsMember(key, member string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.SIsMember(key, member)
}

// SCard 返回集合key的基数(集合中元素的数量)
func (m *MemStore) SCard(key string) (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.SCard(key)
}

// Exec 在内存存储上执行 SQL 语句
func (m *MemStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.Exec(query, args...)
}

// Query 在内存存储上执行查询
func (m *MemStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.Query(query, args...)
}

// QueryRow 在内存存储上执行单行查询
func (m *MemStore) QueryRow(query string, args ...interface{}) *sql.Row {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.QueryRow(query, args...)
}

// Close 关闭内存存储连接
func (m *MemStore) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.Close()
}

// Expire 设置键的过期时间（毫秒）
func (m *MemStore) Expire(key string, milliseconds int64) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.baseStore.Expire(key, milliseconds)
}

// TTL 获取键的剩余生存时间（毫秒）
func (m *MemStore) TTL(key string) (int64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.baseStore.TTL(key)
}
