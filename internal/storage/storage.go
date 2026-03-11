package storage

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/CyberPigeon/internal/modem"
	_ "modernc.org/sqlite"
)

// Message 存储的短信消息
type Message struct {
	ID        string    `json:"id"`
	Modem     string    `json:"modem"`     // 调制解调器 IMEI
	Number    string    `json:"number"`    // 发送方号码
	Text      string    `json:"text"`      // 短信内容
	Timestamp time.Time `json:"timestamp"` // 接收时间
	Saved     time.Time `json:"saved"`     // 保存时间
}

// Storage 短信存储（SQLite）
type Storage struct {
	db             *sql.DB
	mu             sync.RWMutex
	messageHandler func(Message) // 新消息回调
}

// New 创建存储实例，自动建表并迁移旧 JSON 数据
func New(path string) (*Storage, error) {
	// 确保目录存在
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建存储目录: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}

	// 启用 WAL 模式提升并发性能
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 WAL 模式: %w", err)
	}

	// 创建表
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id        TEXT PRIMARY KEY,
		modem     TEXT NOT NULL,
		number    TEXT NOT NULL,
		text      TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		saved     DATETIME NOT NULL
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("创建表: %w", err)
	}

	// 创建索引加速按时间排序查询
	db.Exec("CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp)")

	s := &Storage{db: db}

	// 自动迁移旧 JSON 数据
	s.migrateFromJSON(path)

	return s, nil
}

// migrateFromJSON 检测并迁移旧的 JSON 存储文件
func (s *Storage) migrateFromJSON(dbPath string) {
	// 构建候选路径（去重）
	seen := map[string]bool{}
	var candidates []string
	addCandidate := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			candidates = append(candidates, p)
		}
	}

	// .db → .json（最可能的旧路径）
	if strings.HasSuffix(dbPath, ".db") {
		addCandidate(strings.TrimSuffix(dbPath, ".db") + ".json")
	}
	// 同目录 sms.json
	dir := filepath.Dir(dbPath)
	addCandidate(filepath.Join(dir, "sms.json"))

	for _, jsonPath := range candidates {
		if _, err := os.Stat(jsonPath); err != nil {
			continue
		}

		slog.Info("检测到旧 JSON 存储文件，开始迁移", "path", jsonPath)

		data, err := os.ReadFile(jsonPath)
		if err != nil {
			slog.Warn("读取旧 JSON 文件失败", "error", err)
			continue
		}

		var messages []Message
		if err := json.Unmarshal(data, &messages); err != nil {
			slog.Warn("解析旧 JSON 文件失败", "error", err)
			continue
		}

		migrated := 0
		for _, msg := range messages {
			_, err := s.db.Exec(
				"INSERT OR IGNORE INTO messages (id, modem, number, text, timestamp, saved) VALUES (?, ?, ?, ?, ?, ?)",
				msg.ID, msg.Modem, msg.Number, msg.Text, msg.Timestamp.UTC(), msg.Saved.UTC(),
			)
			if err == nil {
				migrated++
			}
		}

		slog.Info("JSON 数据迁移完成", "total", len(messages), "migrated", migrated)

		// 重命名旧文件为 .bak 备份（避免已是 .bak 后缀时重复追加）
		backupPath := jsonPath + ".bak"
		if strings.HasSuffix(jsonPath, ".bak") {
			backupPath = jsonPath // 已经是备份名，不再追加
		}
		if backupPath != jsonPath {
			if err := os.Rename(jsonPath, backupPath); err != nil {
				slog.Warn("备份旧 JSON 文件失败", "error", err)
			} else {
				slog.Info("旧 JSON 文件已备份", "path", backupPath)
			}
		}
		return // 只迁移第一个找到的文件
	}
}

// GenerateID 生成消息唯一 ID
func GenerateID(modemIMEI string, sms *modem.SMS) string {
	data := fmt.Sprintf("%s|%s|%d|%s|%s", modemIMEI, sms.Path(), sms.Timestamp.UnixNano(), sms.Number, sms.Text)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Save 保存短信
func (s *Storage) Save(modemIMEI string, sms *modem.SMS) error {
	id := GenerateID(modemIMEI, sms)
	now := time.Now().UTC()

	msg := Message{
		ID:        id,
		Modem:     modemIMEI,
		Number:    sms.Number,
		Text:      sms.Text,
		Timestamp: sms.Timestamp,
		Saved:     now,
	}

	result, err := s.db.Exec(
		"INSERT OR IGNORE INTO messages (id, modem, number, text, timestamp, saved) VALUES (?, ?, ?, ?, ?, ?)",
		msg.ID, msg.Modem, msg.Number, msg.Text, msg.Timestamp.UTC(), msg.Saved,
	)
	if err != nil {
		return fmt.Errorf("保存消息: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil // 已存在，跳过
	}

	s.mu.RLock()
	handler := s.messageHandler
	s.mu.RUnlock()

	if handler != nil {
		go handler(msg)
	}

	return nil
}

// Has 检查消息是否存在
func (s *Storage) Has(modemIMEI string, sms *modem.SMS) bool {
	id := GenerateID(modemIMEI, sms)
	var exists int
	err := s.db.QueryRow("SELECT 1 FROM messages WHERE id = ?", id).Scan(&exists)
	return err == nil
}

// SetMessageHandler 设置新消息处理器
func (s *Storage) SetMessageHandler(handler func(Message)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageHandler = handler
}

// List 列出所有短信
func (s *Storage) List() []Message {
	rows, err := s.db.Query("SELECT id, modem, number, text, timestamp, saved FROM messages ORDER BY timestamp ASC")
	if err != nil {
		slog.Error("查询消息列表失败", "error", err)
		return []Message{}
	}
	defer rows.Close()

	return scanMessages(rows)
}

// ListWithPagination 分页获取短信 (按时间倒序)
func (s *Storage) ListWithPagination(limit, offset int) ([]Message, int64) {
	var total int64
	s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&total)
	if total == 0 {
		return []Message{}, 0
	}

	rows, err := s.db.Query(
		"SELECT id, modem, number, text, timestamp, saved FROM messages ORDER BY timestamp DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return []Message{}, total
	}
	defer rows.Close()

	return scanMessages(rows), total
}

// Delete 删除短信
func (s *Storage) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM messages WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除消息: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("消息不存在")
	}
	return nil
}

// Close 关闭存储
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// scanMessages 从 rows 中读取消息列表
func scanMessages(rows *sql.Rows) []Message {
	var messages []Message
	for rows.Next() {
		var msg Message
		var ts, saved time.Time
		if err := rows.Scan(&msg.ID, &msg.Modem, &msg.Number, &msg.Text, &ts, &saved); err != nil {
			continue
		}
		msg.Timestamp = ts.Local()
		msg.Saved = saved.Local()
		messages = append(messages, msg)
	}
	if messages == nil {
		return []Message{}
	}
	return messages
}
