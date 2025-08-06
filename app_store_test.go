package duolasdk

import (
	"os"
	"testing"
	"time"

	"github.com/fzxs8/duolasdk/core"
	"github.com/google/uuid"
)

func TestStore(t *testing.T) {
	logs := core.NewLogger(&core.LoggerOption{
		Level: "debug",
	})

	// 生成唯一的测试数据库文件名
	testDBFile := "duola-test-" + uuid.New().String() + ".db"
	testCloseDBFile := "duola-test-close-" + uuid.New().String() + ".db"

	store := NewStore(core.StoreOption{
		Logger:   logs,
		FilePath: os.TempDir(),
		FileName: testDBFile,
	})

	// 测试 KeyValue 操作
	t.Run("KeyValueOperations", func(t *testing.T) {
		// 测试 Set 和 Get
		key := "test_key"
		value := "test_value"

		// 测试持久化存储
		err := store.Set(key, value, true)
		if err != nil {
			t.Errorf("Set with persistent=true failed: %v", err)
		}

		result, err := store.Get(key, true)
		if err != nil {
			t.Errorf("Get with persistent=true failed: %v", err)
		}
		if result != value {
			t.Errorf("Get with persistent=true returned wrong value: got %s, want %s", result, value)
		}

		// 测试内存存储
		memKey := "mem_key"
		memValue := "mem_value"
		err = store.Set(memKey, memValue, false)
		if err != nil {
			t.Errorf("Set with persistent=false failed: %v", err)
		}

		result, err = store.Get(memKey, false)
		if err != nil {
			t.Errorf("Get with persistent=false failed: %v", err)
		}
		if result != memValue {
			t.Errorf("Get with persistent=false returned wrong value: got %s, want %s", result, memValue)
		}

		// 测试 Delete
		err = store.Delete(key, true)
		if err != nil {
			t.Errorf("Delete with persistent=true failed: %v", err)
		}

		_, err = store.Get(key, true)
		if err == nil {
			t.Error("Get after Delete should return error, but got nil")
		}

		// 测试 List
		store.Set("list_key1", "list_value1", true)
		store.Set("list_key2", "list_value2", true)

		list, err := store.List(true)
		if err != nil {
			t.Errorf("List with persistent=true failed: %v", err)
		}
		if len(list) < 2 {
			t.Errorf("List with persistent=true should contain at least 2 items, got %d", len(list))
		}
		if list["list_key1"] != "list_value1" || list["list_key2"] != "list_value2" {
			t.Error("List with persistent=true returned incorrect values")
		}
	})

	// 测试过期操作
	t.Run("ExpirationOperations", func(t *testing.T) {
		key := "expire_key"
		value := "expire_value"

		// 设置键值
		err := store.Set(key, value, true)
		if err != nil {
			t.Errorf("Set failed: %v", err)
		}

		// 设置过期时间
		err = store.Expire(key, 1000, true) // 1秒
		if err != nil {
			t.Errorf("Expire failed: %v", err)
		}

		// 检查 TTL
		ttl, err := store.TTL(key, true)
		if err != nil {
			t.Errorf("TTL failed: %v", err)
		}
		if ttl <= 0 || ttl > 1000 {
			t.Errorf("TTL returned unexpected value: %d", ttl)
		}

		// 等待过期
		time.Sleep(1100 * time.Millisecond)

		// 检查键是否已过期
		_, err = store.Get(key, true)
		if err == nil {
			t.Error("Get should return error for expired key, but got nil")
		}
	})

	// 测试 Hash 操作
	t.Run("HashOperations", func(t *testing.T) {
		key := "hash_key"
		field := "field1"
		value := "hash_value"

		// 测试 HSet
		err := store.HSet(key, field, value, true)
		if err != nil {
			t.Errorf("HSet failed: %v", err)
		}

		// 测试 HGet
		result, err := store.HGet(key, field, true)
		if err != nil {
			t.Errorf("HGet failed: %v", err)
		}
		if result != value {
			t.Errorf("HGet returned wrong value: got %s, want %s", result, value)
		}

		// 测试 HGetAll
		store.HSet(key, "field2", "value2", true)
		all, err := store.HGetAll(key, true)
		if err != nil {
			t.Errorf("HGetAll failed: %v", err)
		}
		if len(all) != 2 {
			t.Errorf("HGetAll should return 2 fields, got %d", len(all))
		}

		// 测试 HExists
		exists, err := store.HExists(key, field, true)
		if err != nil {
			t.Errorf("HExists failed: %v", err)
		}
		if !exists {
			t.Error("HExists should return true for existing field")
		}

		// 测试 HDel (注意：HDel方法没有persistent参数)
		err = store.HDel(key, field)
		if err != nil {
			t.Errorf("HDel failed: %v", err)
		}

		exists, err = store.HExists(key, field, true)
		if err != nil {
			t.Errorf("HExists failed: %v", err)
		}
		if exists {
			t.Error("HExists should return false for deleted field")
		}
	})

	// 测试 List 操作 (Redis风格)
	t.Run("ListOperations", func(t *testing.T) {
		key := "list_key"

		// 测试 LPush (注意：LPush方法没有persistent参数)
		err := store.LPush(key, "value1", "value2")
		if err != nil {
			t.Errorf("LPush failed: %v", err)
		}

		// 测试 RPush (注意：RPush方法没有persistent参数)
		err = store.RPush(key, "value3")
		if err != nil {
			t.Errorf("RPush failed: %v", err)
		}

		// 测试 LLen
		length, err := store.LLen(key, true)
		if err != nil {
			t.Errorf("LLen failed: %v", err)
		}
		if length != 3 {
			t.Errorf("LLen should return 3, got %d", length)
		}

		// 测试 LRange
		values, err := store.LRange(key, 0, -1, true)
		if err != nil {
			t.Errorf("LRange failed: %v", err)
		}
		if len(values) != 3 {
			t.Errorf("LRange should return 3 values, got %d", len(values))
		}

		// 测试 LPop
		popped, err := store.LPop(key, true)
		if err != nil {
			t.Errorf("LPop failed: %v", err)
		}
		if popped != "value2" {
			t.Errorf("LPop should return 'value2', got %s", popped)
		}

		// 测试 RPop
		popped, err = store.RPop(key, true)
		if err != nil {
			t.Errorf("RPop failed: %v", err)
		}
		if popped != "value3" {
			t.Errorf("RPop should return 'value3', got %s", popped)
		}
	})

	// 测试 Set 操作 (Redis风格)
	t.Run("SetOperations", func(t *testing.T) {
		key := "set_key"
		member1 := "member1"
		member2 := "member2"

		// 测试 SAdd (注意：SAdd方法没有persistent参数)
		err := store.SAdd(key, member1, member2)
		if err != nil {
			t.Errorf("SAdd failed: %v", err)
		}

		// 测试 SMembers
		members, err := store.SMembers(key, true)
		if err != nil {
			t.Errorf("SMembers failed: %v", err)
		}
		if len(members) != 2 {
			t.Errorf("SMembers should return 2 members, got %d", len(members))
		}

		// 测试 SIsMember
		isMember, err := store.SIsMember(key, member1, true)
		if err != nil {
			t.Errorf("SIsMember failed: %v", err)
		}
		if !isMember {
			t.Error("SIsMember should return true for existing member")
		}

		// 测试 SCard
		card, err := store.SCard(key, true)
		if err != nil {
			t.Errorf("SCard failed: %v", err)
		}
		if card != 2 {
			t.Errorf("SCard should return 2, got %d", card)
		}

		// 测试 SRem (注意：SRem方法没有persistent参数)
		err = store.SRem(key, member1)
		if err != nil {
			t.Errorf("SRem failed: %v", err)
		}

		isMember, err = store.SIsMember(key, member1, true)
		if err != nil {
			t.Errorf("SIsMember failed: %v", err)
		}
		if isMember {
			t.Error("SIsMember should return false for removed member")
		}
	})

	// 测试 SQL 操作
	t.Run("SQLOperations", func(t *testing.T) {
		// 先清理可能存在的测试数据
		_, err := store.Exec("DELETE FROM kv_store WHERE key = ?", "sql_key")
		if err != nil {
			t.Logf("Warning: failed to clean up existing test data: %v", err)
		}

		// 测试 Exec
		result, err := store.Exec("INSERT INTO kv_store (key, value) VALUES (?, ?)", "sql_key", "sql_value")
		if err != nil {
			t.Errorf("Exec failed: %v", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			t.Errorf("RowsAffected failed: %v", err)
		}
		if rowsAffected != 1 {
			t.Errorf("Exec should affect 1 row, got %d", rowsAffected)
		}

		// 测试 Query
		rows, err := store.Query("SELECT key, value FROM kv_store WHERE key = ?", "sql_key")
		if err != nil {
			t.Errorf("Query failed: %v", err)
		}
		defer rows.Close()

		if !rows.Next() {
			t.Error("Query should return at least one row")
		}

		var key, value string
		err = rows.Scan(&key, &value)
		if err != nil {
			t.Errorf("Scan failed: %v", err)
		}
		if key != "sql_key" || value != "sql_value" {
			t.Errorf("Query returned wrong data: got key=%s, value=%s", key, value)
		}

		// 测试 QueryRow
		row := store.QueryRow("SELECT value FROM kv_store WHERE key = ?", "sql_key")
		var rowValue string
		err = row.Scan(&rowValue)
		if err != nil {
			t.Errorf("QueryRow.Scan failed: %v", err)
		}
		if rowValue != "sql_value" {
			t.Errorf("QueryRow returned wrong value: got %s, want %s", rowValue, "sql_value")
		}
	})

	// 测试 Close 操作
	t.Run("CloseOperation", func(t *testing.T) {
		// 创建一个新的存储实例用于测试Close
		tempStore := NewStore(core.StoreOption{
			Logger:   logs,
			FilePath: os.TempDir(),
			FileName: testCloseDBFile,
		})

		err := tempStore.Close()
		if err != nil {
			t.Errorf("Close failed: %v", err)
		}
	})

	// 测试完成后清理测试文件
	t.Cleanup(func() {
		store.Close()
		testDBPath := os.TempDir() + string(os.PathSeparator) + testDBFile
		testCloseDBPath := os.TempDir() + string(os.PathSeparator) + testCloseDBFile
		os.Remove(testDBPath)
		os.Remove(testCloseDBPath)
	})
}
