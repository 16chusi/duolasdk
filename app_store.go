package duolasdk

import (
	"database/sql"

	"github.com/16chusi/duolasdk/core"
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

// GetStore 根据 persistent 参数选择存储实例
func (s *AppStore) GetStore(persistent ...bool) core.IStore {
	isPersistent := true
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	if isPersistent {
		return s.local
	}
	return s.mem
}

// Set stores the key-value pair.
func (s *AppStore) Set(key, value string, persistent ...bool) error {
	return s.GetStore(persistent...).Set(key, value)
}

// Get retrieves the value for a key.
func (s *AppStore) Get(key string, persistent ...bool) (string, error) {
	return s.GetStore(persistent...).Get(key)
}

// Delete removes a key.
func (s *AppStore) Delete(key string, persistent ...bool) error {
	return s.GetStore(persistent...).Delete(key)
}

// List lists all key-value pairs.
func (s *AppStore) List(persistent ...bool) (map[string]string, error) {
	return s.GetStore(persistent...).List()
}

// Expire sets an expiration for a key.
func (s *AppStore) Expire(key string, milliseconds int64, persistent ...bool) error {
	return s.GetStore(persistent...).Expire(key, milliseconds)
}

// TTL gets the time-to-live for a key.
func (s *AppStore) TTL(key string, persistent ...bool) (int64, error) {
	return s.GetStore(persistent...).TTL(key)
}

// HSet sets the value of a field in a hash.
func (s *AppStore) HSet(key, field, value string, persistent ...bool) error {
	return s.GetStore(persistent...).HSet(key, field, value)
}

// HGet gets the value of a field in a hash.
func (s *AppStore) HGet(key, field string, persistent ...bool) (string, error) {
	return s.GetStore(persistent...).HGet(key, field)
}

// HGetAll gets all fields and values in a hash.
func (s *AppStore) HGetAll(key string, persistent ...bool) (map[string]string, error) {
	return s.GetStore(persistent...).HGetAll(key)
}

// HMGet gets the values of multiple fields in a hash.
func (s *AppStore) HMGet(key string, fields ...string) ([]string, error) {
	return s.local.HMGet(key, fields...)
}

// HMGetMem gets the values of multiple fields in a hash from memory.
func (s *AppStore) HMGetMem(key string, fields ...string) ([]string, error) {
	return s.mem.HMGet(key, fields...)
}

// HMSet sets the values of multiple fields in a hash.
func (s *AppStore) HMSet(key string, fieldValue map[string]string, persistent ...bool) error {
	return s.GetStore(persistent...).HMSet(key, fieldValue)
}

// HDel deletes one or more fields from a hash.
func (s *AppStore) HDel(key string, fields ...string) error {
	return s.local.HDel(key, fields...)
}

// HDelMem deletes one or more fields from a hash in memory.
func (s *AppStore) HDelMem(key string, fields ...string) error {

	return s.mem.HDel(key, fields...)
}

// HExists checks if a field exists in a hash.
func (s *AppStore) HExists(key, field string, persistent ...bool) (bool, error) {
	// 修正返回值类型
	return s.GetStore(persistent...).HExists(key, field)
}

// HKeys gets all the fields in a hash.
func (s *AppStore) HKeys(key string, persistent ...bool) ([]string, error) {
	return s.GetStore(persistent...).HKeys(key)
}

// HLen gets the number of fields in a hash.
func (s *AppStore) HLen(key string, persistent ...bool) (int, error) {
	return s.GetStore(persistent...).HLen(key)
}

// LPush prepends one or multiple values to a list.
func (s *AppStore) LPush(key string, values ...string) error {
	return s.local.LPush(key, values...)
}

// LPushMem prepends one or multiple values to a list in memory.
func (s *AppStore) LPushMem(key string, values ...string) error {
	return s.mem.LPush(key, values...)
}

// RPush appends one or multiple values to a list.
func (s *AppStore) RPush(key string, values ...string) error {
	return s.local.RPush(key, values...)
}

// RPushMem appends one or multiple values to a list in memory.
func (s *AppStore) RPushMem(key string, values ...string) error {
	return s.mem.RPush(key, values...)
}

// LPop removes and returns the first element of a list.
func (s *AppStore) LPop(key string, persistent ...bool) (string, error) {
	return s.GetStore(persistent...).LPop(key)
}

// RPop removes and returns the last element of a list.
func (s *AppStore) RPop(key string, persistent ...bool) (string, error) {
	return s.GetStore(persistent...).RPop(key)
}

// LRange returns a range of elements from a list.
func (s *AppStore) LRange(key string, start, stop int, persistent ...bool) ([]string, error) {
	return s.GetStore(persistent...).LRange(key, start, stop)
}

// LLen gets the length of a list.
func (s *AppStore) LLen(key string, persistent ...bool) (int, error) {
	return s.GetStore(persistent...).LLen(key)
}

// SAdd adds one or more members to a set.
func (s *AppStore) SAdd(key string, members ...string) error {
	return s.local.SAdd(key, members...)
}

// SAddMem adds one or more members to a set in memory.
func (s *AppStore) SAddMem(key string, members ...string) error {
	return s.mem.SAdd(key, members...)
}

// SRem removes one or more members from a set.
func (s *AppStore) SRem(key string, members ...string) error {
	return s.local.SRem(key, members...)
}

// SRemMem removes one or more members from a set in memory.
func (s *AppStore) SRemMem(key string, members ...string) error {
	return s.mem.SRem(key, members...)
}

// SMembers returns all the members of a set.
func (s *AppStore) SMembers(key string, persistent ...bool) ([]string, error) {
	return s.GetStore(persistent...).SMembers(key)
}

// SIsMember checks if a member is in a set.
func (s *AppStore) SIsMember(key, member string, persistent ...bool) (bool, error) {
	return s.GetStore(persistent...).SIsMember(key, member)
}

// SCard gets the number of members in a set.
func (s *AppStore) SCard(key string, persistent ...bool) (int, error) {
	return s.GetStore(persistent...).SCard(key)
}

// Exec executes a query without returning any rows.
func (s *AppStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.local.Exec(query, args...)
}

// ExecMem executes a query without returning any rows in memory.
func (s *AppStore) ExecMem(query string, args ...interface{}) (sql.Result, error) {
	return s.mem.Exec(query, args...)
}

// Query executes a query that returns rows.
func (s *AppStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.local.Query(query, args...)
}

// QueryMem executes a query that returns rows in memory.
func (s *AppStore) QueryMem(query string, args ...interface{}) (*sql.Rows, error) {
	return s.mem.Query(query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (s *AppStore) QueryRow(query string, args ...interface{}) *sql.Row {
	// SQL操作默认使用持久化存储
	return s.local.QueryRow(query, args...)
}

// QueryRowMem executes a query that is expected to return at most one row in memory.
func (s *AppStore) QueryRowMem(query string, args ...interface{}) *sql.Row {
	return s.mem.QueryRow(query, args...)
}

// Close closes the storage connection.
func (s *AppStore) Close() error {
	// 关闭所有存储连接
	localErr := s.local.Close()
	memErr := s.mem.Close()

	if localErr != nil {
		return localErr
	}
	return memErr
}

func (s *AppStore) CreateTable(tableName string, schema string) error {
	return s.local.CreateTable(tableName, schema)
}

func (s *AppStore) CreateTableMem(tableName string, schema string) error {
	return s.mem.CreateTable(tableName, schema)
}

func (s *AppStore) DropTable(tableName string) error {
	return s.local.DropTable(tableName)
}
func (s *AppStore) DropTableMem(tableName string) error {
	return s.mem.DropTable(tableName)
}

func (s *AppStore) TableExists(tableName string) (bool, error) {
	return s.local.TableExists(tableName)
}
func (s *AppStore) TableExistsMem(tableName string) (bool, error) {
	return s.mem.TableExists(tableName)
}
