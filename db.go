package main

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Message struct {
	ID            int64  `json:"id" db:"id"`
	TelegramMsgID int64  `json:"telegram_msg_id" db:"telegram_msg_id"`
	ChatID        int64  `json:"chat_id" db:"chat_id"`
	TopicID       *int64 `json:"topic_id,omitempty" db:"topic_id"`
	SenderName    string `json:"sender_name" db:"sender_name"`
	SenderID      int64  `json:"sender_id" db:"sender_id"`
	Content       string `json:"content" db:"content"`
	VoicePath     string `json:"voice_path,omitempty" db:"voice_path"`
	Timestamp     string `json:"timestamp" db:"timestamp"`
	Read          int    `json:"-" db:"read"`
}

type Chat struct {
	ChatID   int64  `json:"chat_id" db:"chat_id"`
	ChatType string `json:"chat_type" db:"chat_type"`
	Title    string `json:"title" db:"title"`
	IsForum  bool   `json:"is_forum" db:"is_forum"`
	LastSeen string `json:"last_seen" db:"last_seen"`
}

type DB struct {
	conn *sqlx.DB
}

func OpenDB(path string) (*DB, error) {
	conn, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.MustExec(`
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_msg_id INTEGER,
			chat_id INTEGER,
			topic_id INTEGER,
			sender_name TEXT,
			sender_id INTEGER,
			content TEXT,
			voice_path TEXT DEFAULT '',
			timestamp TEXT,
			read INTEGER DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS chats (
			chat_id INTEGER PRIMARY KEY,
			chat_type TEXT,
			title TEXT,
			is_forum INTEGER DEFAULT 0,
			last_seen TEXT
		);
	`)
	// Migration: add voice_path if missing (existing DBs)
	conn.Exec(`ALTER TABLE messages ADD COLUMN voice_path TEXT DEFAULT ''`)
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) InsertMessage(m Message) error {
	_, err := db.conn.Exec(
		`INSERT INTO messages (telegram_msg_id, chat_id, topic_id, sender_name, sender_id, content, voice_path, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.TelegramMsgID, m.ChatID, m.TopicID, m.SenderName, m.SenderID, m.Content, m.VoicePath, m.Timestamp,
	)
	return err
}

func (db *DB) PollMessages(chatID int64) ([]Message, error) {
	var msgs []Message
	var err error

	if chatID != 0 {
		err = db.conn.Select(&msgs, `SELECT * FROM messages WHERE read = 0 AND chat_id = ? ORDER BY id`, chatID)
	} else {
		err = db.conn.Select(&msgs, `SELECT * FROM messages WHERE read = 0 ORDER BY id`)
	}
	if err != nil {
		return nil, err
	}

	for _, m := range msgs {
		db.conn.Exec(`UPDATE messages SET read = 1 WHERE id = ?`, m.ID)
	}
	return msgs, nil
}

func (db *DB) UpsertChat(c Chat) error {
	_, err := db.conn.Exec(
		`INSERT INTO chats (chat_id, chat_type, title, is_forum, last_seen)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(chat_id) DO UPDATE SET
			chat_type = excluded.chat_type,
			title = excluded.title,
			is_forum = excluded.is_forum,
			last_seen = excluded.last_seen`,
		c.ChatID, c.ChatType, c.Title, c.IsForum, c.LastSeen,
	)
	return err
}

func (db *DB) ListChats() ([]Chat, error) {
	var chats []Chat
	err := db.conn.Select(&chats, `SELECT * FROM chats ORDER BY last_seen DESC`)
	return chats, err
}
