package duolasdk

import (
	"database/sql"

	"github.com/fzxs8/duolasdk/core"
)

// AppStore 实现 IStore 接口，支持持久化和内存存储
type AppStore struct {
	local *core.LocalStore
	mem   *core.MemStore
}

// NewStore 创建并初始化 AppStore 实例
func NewStore(opts ...core.StoreOption) *AppStore {
	// 初始化本地存储
	localStore, err := core.NewLocalStore(opts...)
	if err != nil {
		// 处理错误
		panic(err)
	}

	// 初始化内存存储
	memStore, err := core.NewMemStore(opts...)
	if err != nil {
		// 处理错误
		panic(err)
	}

	return &AppStore{
		local: localStore,
		mem:   memStore,
	}
}

// getStore 根据 persistent 参数选择存储实例
func (s *AppStore) getStore(persistent ...bool) core.IStore {
	isPersistent := true
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	if isPersistent {
		return s.local
	}
	return s.mem
}

// KeyValue operations
func (s *AppStore) Set(key, value string, persistent ...bool) error {
	return s.getStore(persistent...).Set(key, value)
}

func (s *AppStore) Get(key string, persistent ...bool) (string, error) {
	return s.getStore(persistent...).Get(key)
}

func (s *AppStore) Delete(key string, persistent ...bool) error {
	return s.getStore(persistent...).Delete(key)
}

func (s *AppStore) List(persistent ...bool) (map[string]string, error) {
	return s.getStore(persistent...).List()
}

// Expiration operations
func (s *AppStore) Expire(key string, milliseconds int64, persistent ...bool) error {
	return s.getStore(persistent...).Expire(key, milliseconds)
}

func (s *AppStore) TTL(key string, persistent ...bool) (int64, error) {
	return s.getStore(persistent...).TTL(key)
}

// Hash operations (like redis)
func (s *AppStore) HSet(key, field, value string, persistent ...bool) error {
	return s.getStore(persistent...).HSet(key, field, value)
}

func (s *AppStore) HGet(key, field string, persistent ...bool) (string, error) {
	return s.getStore(persistent...).HGet(key, field)
}

func (s *AppStore) HGetAll(key string, persistent ...bool) (map[string]string, error) {
	return s.getStore(persistent...).HGetAll(key)
}

func (s *AppStore) HMGet(key string, fields ...string) ([]string, error) {
	return s.local.HMGet(key, fields...)
}
func (s *AppStore) HMGetMem(key string, fields ...string) ([]string, error) {
	return s.mem.HMGet(key, fields...)
}

func (s *AppStore) HMSet(key string, fieldValue map[string]string, persistent ...bool) error {
	return s.getStore(persistent...).HMSet(key, fieldValue)
}

func (s *AppStore) HDel(key string, fields ...string) error {
	return s.local.HDel(key, fields...)
}

func (s *AppStore) HDelMem(key string, fields ...string) error {

	return s.mem.HDel(key, fields...)
}

func (s *AppStore) HExists(key, field string, persistent ...bool) (bool, error) {
	// 修正返回值类型
	return s.getStore(persistent...).HExists(key, field)
}

func (s *AppStore) HKeys(key string, persistent ...bool) ([]string, error) {
	return s.getStore(persistent...).HKeys(key)
}

func (s *AppStore) HLen(key string, persistent ...bool) (int, error) {
	return s.getStore(persistent...).HLen(key)
}

// List operations (like redis)
func (s *AppStore) LPush(key string, values ...string) error {
	return s.local.LPush(key, values...)
}

func (s *AppStore) LPushMem(key string, values ...string) error {
	return s.mem.LPush(key, values...)
}

func (s *AppStore) RPush(key string, values ...string) error {
	return s.local.RPush(key, values...)
}

func (s *AppStore) RPushMem(key string, values ...string) error {
	return s.mem.RPush(key, values...)
}

func (s *AppStore) LPop(key string, persistent ...bool) (string, error) {
	return s.getStore(persistent...).LPop(key)
}

func (s *AppStore) RPop(key string, persistent ...bool) (string, error) {
	return s.getStore(persistent...).RPop(key)
}

func (s *AppStore) LRange(key string, start, stop int, persistent ...bool) ([]string, error) {
	return s.getStore(persistent...).LRange(key, start, stop)
}

func (s *AppStore) LLen(key string, persistent ...bool) (int, error) {
	return s.getStore(persistent...).LLen(key)
}

// Set operations (like redis)
func (s *AppStore) SAdd(key string, members ...string) error {
	return s.local.SAdd(key, members...)
}

func (s *AppStore) SAddMem(key string, members ...string) error {
	return s.mem.SAdd(key, members...)
}

func (s *AppStore) SRem(key string, members ...string) error {
	return s.local.SRem(key, members...)
}
func (s *AppStore) SRemMem(key string, members ...string) error {
	return s.mem.SRem(key, members...)
}

func (s *AppStore) SMembers(key string, persistent ...bool) ([]string, error) {
	return s.getStore(persistent...).SMembers(key)
}

func (s *AppStore) SIsMember(key, member string, persistent ...bool) (bool, error) {
	return s.getStore(persistent...).SIsMember(key, member)
}

func (s *AppStore) SCard(key string, persistent ...bool) (int, error) {
	return s.getStore(persistent...).SCard(key)
}

// SQL operations
func (s *AppStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.local.Exec(query, args...)
}
func (s *AppStore) ExecMem(query string, args ...interface{}) (sql.Result, error) {
	return s.mem.Exec(query, args...)
}

func (s *AppStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.local.Query(query, args...)
}
func (s *AppStore) QueryMem(query string, args ...interface{}) (*sql.Rows, error) {
	return s.mem.Query(query, args...)
}

func (s *AppStore) QueryRow(query string, args ...interface{}) *sql.Row {
	// SQL操作默认使用持久化存储
	return s.local.QueryRow(query, args...)
}
func (s *AppStore) QueryRowMem(query string, args ...interface{}) *sql.Row {
	return s.mem.QueryRow(query, args...)
}

// Close 关闭存储连接
func (s *AppStore) Close() error {
	// 关闭所有存储连接
	localErr := s.local.Close()
	memErr := s.mem.Close()

	if localErr != nil {
		return localErr
	}
	return memErr
}
