# tg-bot Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI binary that bridges Telegram to Claude Code via SQLite.

**Architecture:** Long-running `listen` command polls Telegram API and writes incoming messages to SQLite. Instant CLI subcommands (`poll`, `send`, `chats`, `typing`) read/write the same DB. Claude Code invokes the CLI via bash/hooks.

**Tech Stack:** Go, telebot v4 (gopkg.in/telebot.v4), modernc.org/sqlite (pure Go), cobra (CLI framework)

---

### Task 1: Scaffold Go project

**Files:**
- Create: `~/gits/tg-bot/go.mod`
- Create: `~/gits/tg-bot/main.go`
- Create: `~/gits/tg-bot/.gitignore`

**Step 1: Create repo and initialize Go module**

```bash
mkdir -p ~/gits/tg-bot && cd ~/gits/tg-bot
git init
go mod init github.com/idvorkin/tg-bot
```

**Step 2: Create .gitignore**

```gitignore
tg-bot
*.db
.env
```

**Step 3: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tg-bot <command> [args]")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "not implemented:", os.Args[1])
	os.Exit(1)
}
```

**Step 4: Verify it builds**

Run: `go build -o tg-bot . && ./tg-bot`
Expected: `usage: tg-bot <command> [args]` on stderr, exit 1

**Step 5: Commit**

```bash
git add -A
git commit -m "feat: scaffold Go project"
```

---

### Task 2: SQLite database layer

**Files:**
- Create: `~/gits/tg-bot/db.go`
- Create: `~/gits/tg-bot/db_test.go`

**Step 1: Install SQLite dependency**

```bash
cd ~/gits/tg-bot
go get modernc.org/sqlite
go get github.com/jmoiron/sqlx
```

**Step 2: Write the failing test**

```go
// db_test.go
package main

import (
	"os"
	"testing"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	path := t.TempDir() + "/test.db"
	db, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}

func TestInsertAndPollMessages(t *testing.T) {
	db := testDB(t)

	err := db.InsertMessage(Message{
		TelegramMsgID: 1,
		ChatID:        100,
		TopicID:       nil,
		SenderName:    "Alice",
		SenderID:      42,
		Content:       "hello",
		Timestamp:     "2026-03-07T12:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	msgs, err := db.PollMessages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "hello" {
		t.Fatalf("expected 'hello', got %q", msgs[0].Content)
	}
	if msgs[0].SenderName != "Alice" {
		t.Fatalf("expected 'Alice', got %q", msgs[0].SenderName)
	}

	// polling again should return nothing (marked as read)
	msgs2, err := db.PollMessages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs2) != 0 {
		t.Fatalf("expected 0 messages after poll, got %d", len(msgs2))
	}
}

func TestPollMessagesByChatID(t *testing.T) {
	db := testDB(t)

	db.InsertMessage(Message{TelegramMsgID: 1, ChatID: 100, SenderName: "A", SenderID: 1, Content: "msg1", Timestamp: "2026-03-07T12:00:00Z"})
	db.InsertMessage(Message{TelegramMsgID: 2, ChatID: 200, SenderName: "B", SenderID: 2, Content: "msg2", Timestamp: "2026-03-07T12:01:00Z"})

	msgs, err := db.PollMessages(100)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for chat 100, got %d", len(msgs))
	}
	if msgs[0].Content != "msg1" {
		t.Fatalf("expected 'msg1', got %q", msgs[0].Content)
	}
}

func TestUpsertAndListChats(t *testing.T) {
	db := testDB(t)

	err := db.UpsertChat(Chat{
		ChatID:   100,
		ChatType: "group",
		Title:    "Test Group",
		IsForum:  true,
		LastSeen: "2026-03-07T12:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	chats, err := db.ListChats()
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0].Title != "Test Group" {
		t.Fatalf("expected 'Test Group', got %q", chats[0].Title)
	}
	if !chats[0].IsForum {
		t.Fatal("expected IsForum=true")
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test -run TestInsert -v`
Expected: compilation error — `OpenDB`, `DB`, `Message` not defined

**Step 4: Write implementation**

```go
// db.go
package main

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Message struct {
	ID             int64  `json:"id" db:"id"`
	TelegramMsgID  int64  `json:"telegram_msg_id" db:"telegram_msg_id"`
	ChatID         int64  `json:"chat_id" db:"chat_id"`
	TopicID        *int64 `json:"topic_id,omitempty" db:"topic_id"`
	SenderName     string `json:"sender_name" db:"sender_name"`
	SenderID       int64  `json:"sender_id" db:"sender_id"`
	Content        string `json:"content" db:"content"`
	Timestamp      string `json:"timestamp" db:"timestamp"`
	Read           int    `json:"-" db:"read"`
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
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) InsertMessage(m Message) error {
	_, err := db.conn.Exec(
		`INSERT INTO messages (telegram_msg_id, chat_id, topic_id, sender_name, sender_id, content, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.TelegramMsgID, m.ChatID, m.TopicID, m.SenderName, m.SenderID, m.Content, m.Timestamp,
	)
	return err
}

// PollMessages returns unread messages and marks them as read.
// If chatID is 0, returns all unread messages.
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

	// Mark as read
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
```

**Step 5: Run tests**

Run: `go test -v`
Expected: all 3 tests PASS

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: SQLite database layer with messages and chats"
```

---

### Task 3: CLI command routing with cobra

**Files:**
- Modify: `~/gits/tg-bot/main.go`
- Create: `~/gits/tg-bot/cmd_poll.go`
- Create: `~/gits/tg-bot/cmd_send.go`
- Create: `~/gits/tg-bot/cmd_chats.go`
- Create: `~/gits/tg-bot/cmd_listen.go`
- Create: `~/gits/tg-bot/cmd_typing.go`
- Create: `~/gits/tg-bot/config.go`

**Step 1: Install cobra**

```bash
go get github.com/spf13/cobra
```

**Step 2: Create config helper**

```go
// config.go
package main

import (
	"os"
	"path/filepath"
)

func dbPath() string {
	if p := os.Getenv("TGBOT_DB"); p != "" {
		return p
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(dataDir, "tg-bot")
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "tg-bot.db")
}

func botToken() string {
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}
```

**Step 3: Rewrite main.go with cobra root command**

```go
// main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tg-bot",
	Short: "Telegram bridge for Claude Code",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 4: Create poll command**

```go
// cmd_poll.go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var pollChatID int64

var pollCmd = &cobra.Command{
	Use:   "poll",
	Short: "Return unread messages and mark as read",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := OpenDB(dbPath())
		if err != nil {
			return err
		}
		defer db.Close()

		msgs, err := db.PollMessages(pollChatID)
		if err != nil {
			return err
		}

		out, _ := json.MarshalIndent(msgs, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	pollCmd.Flags().Int64Var(&pollChatID, "chat", 0, "filter to specific chat ID")
	rootCmd.AddCommand(pollCmd)
}
```

**Step 5: Create send command**

```go
// cmd_send.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v4"
)

var sendTopicID int

var sendCmd = &cobra.Command{
	Use:   "send <chat_id> <message>",
	Short: "Send a message to a chat",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := botToken()
		if token == "" {
			return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
		}

		chatID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid chat_id: %w", err)
		}
		text := strings.Join(args[1:], " ")

		bot, err := tele.NewBot(tele.Settings{
			Token:  token,
			Offline: true,
		})
		if err != nil {
			return err
		}

		chat := &tele.Chat{ID: chatID}
		opts := &tele.SendOptions{}
		if sendTopicID != 0 {
			opts.ThreadID = sendTopicID
		}

		msg, err := bot.Send(chat, text, opts)
		if err != nil {
			return err
		}
		fmt.Printf(`{"message_id": %d, "chat_id": %d}%s`, msg.ID, chatID, "\n")
		return nil
	},
}

func init() {
	sendCmd.Flags().IntVar(&sendTopicID, "topic", 0, "forum topic ID")
	rootCmd.AddCommand(sendCmd)
}
```

**Step 6: Create chats command**

```go
// cmd_chats.go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var chatsCmd = &cobra.Command{
	Use:   "chats",
	Short: "List known chats",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := OpenDB(dbPath())
		if err != nil {
			return err
		}
		defer db.Close()

		chats, err := db.ListChats()
		if err != nil {
			return err
		}

		out, _ := json.MarshalIndent(chats, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(chatsCmd)
}
```

**Step 7: Create typing command (stub — needs bot)**

```go
// cmd_typing.go
package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v4"
)

var typingCmd = &cobra.Command{
	Use:   "typing <chat_id>",
	Short: "Send typing indicator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := botToken()
		if token == "" {
			return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
		}

		chatID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid chat_id: %w", err)
		}

		bot, err := tele.NewBot(tele.Settings{
			Token:  token,
			Offline: true,
		})
		if err != nil {
			return err
		}

		return bot.Notify(&tele.Chat{ID: chatID}, tele.Typing)
	},
}

func init() {
	rootCmd.AddCommand(typingCmd)
}
```

**Step 8: Create listen command (placeholder)**

```go
// cmd_listen.go
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Start listening for Telegram messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "listen: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)
}
```

**Step 9: Build and verify subcommands**

Run: `go build -o tg-bot . && ./tg-bot --help`
Expected: Shows help with `chats`, `listen`, `poll`, `send`, `typing` subcommands

**Step 10: Commit**

```bash
git add -A
git commit -m "feat: CLI commands for poll, send, chats, typing"
```

---

### Task 4: Telegram listener

**Files:**
- Modify: `~/gits/tg-bot/cmd_listen.go`

**Step 1: Implement the listener**

```go
// cmd_listen.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v4"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Start listening for Telegram messages",
	RunE:  runListen,
}

func init() {
	rootCmd.AddCommand(listenCmd)
}

func runListen(cmd *cobra.Command, args []string) error {
	token := botToken()
	if token == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	db, err := OpenDB(dbPath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	bot, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 30 * time.Second},
	})
	if err != nil {
		return fmt.Errorf("create bot: %w", err)
	}

	bot.Handle(tele.OnText, func(c tele.Context) error {
		msg := c.Message()
		log.Printf("[%s] %s: %s", msg.Chat.Title, msg.Sender.FirstName, msg.Text)

		// Upsert chat metadata
		chatType := "private"
		if msg.Chat.Type == tele.ChatGroup || msg.Chat.Type == tele.ChatSuperGroup {
			chatType = string(msg.Chat.Type)
		}
		db.UpsertChat(Chat{
			ChatID:   msg.Chat.ID,
			ChatType: chatType,
			Title:    msg.Chat.Title,
			IsForum:  msg.Chat.IsForum,
			LastSeen: msg.Time().UTC().Format(time.RFC3339),
		})

		// Store message
		var topicID *int64
		if msg.ThreadID != 0 {
			tid := int64(msg.ThreadID)
			topicID = &tid
		}

		return db.InsertMessage(Message{
			TelegramMsgID: int64(msg.ID),
			ChatID:        msg.Chat.ID,
			TopicID:       topicID,
			SenderName:    msg.Sender.FirstName,
			SenderID:      msg.Sender.ID,
			Content:       msg.Text,
			Timestamp:     msg.Time().UTC().Format(time.RFC3339),
		})
	})

	// Handle /chatid utility command
	bot.Handle("/chatid", func(c tele.Context) error {
		return c.Reply(fmt.Sprintf("Chat ID: `%d`", c.Chat().ID), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	})

	// Handle /ping
	bot.Handle("/ping", func(c tele.Context) error {
		return c.Reply("pong")
	})

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down...")
		bot.Stop()
	}()

	log.Printf("tg-bot listening (bot: %s)", bot.Me.Username)
	bot.Start()
	return nil
}
```

**Step 2: Build**

Run: `go build -o tg-bot .`
Expected: compiles without errors

**Step 3: Manual test (requires TELEGRAM_BOT_TOKEN)**

Run: `TELEGRAM_BOT_TOKEN=<your-token> ./tg-bot listen`
Expected: `tg-bot listening (bot: YourBotName)` — send a message in Telegram, see it logged

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: Telegram listener with message storage"
```

---

### Task 5: End-to-end manual verification

**Step 1: Build final binary**

```bash
cd ~/gits/tg-bot
go build -o tg-bot .
```

**Step 2: Run listener in background**

```bash
export TELEGRAM_BOT_TOKEN=<your-token>
./tg-bot listen &
```

**Step 3: Send a message in Telegram to the bot**

Send "hello from phone" in a chat with the bot.

**Step 4: Poll messages**

```bash
./tg-bot poll
```

Expected: JSON array with the message you sent.

**Step 5: Send a reply**

```bash
./tg-bot send <chat_id> "hello from Claude Code!"
```

Expected: Message appears in Telegram.

**Step 6: List chats**

```bash
./tg-bot chats
```

Expected: JSON array showing the chat you messaged from.

**Step 7: Run tests one final time**

```bash
go test -v
```

Expected: All pass.

**Step 8: Commit any fixes**

```bash
git add -A
git commit -m "chore: end-to-end verification complete"
```
