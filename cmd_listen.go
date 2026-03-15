package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v4"
)

var (
	listenJSONLPath  string
	listenTranscribe bool
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Start listening for Telegram messages",
	RunE:  runListen,
}

func init() {
	listenCmd.Flags().StringVar(&listenJSONLPath, "jsonl", "", "path to JSONL file for real-time message output")
	listenCmd.Flags().BoolVar(&listenTranscribe, "transcribe", false, "transcribe voice messages before writing to JSONL")
	rootCmd.AddCommand(listenCmd)
}

func voiceDir() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("warning: cannot determine home directory: %v", err)
			home = "/tmp"
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(dataDir, "tg-bot", "voice")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("warning: cannot create voice directory: %v", err)
	}
	return dir
}

func upsertChatFromMsg(db *DB, msg *tele.Message) {
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
}

func topicFromMsg(msg *tele.Message) *int64 {
	if msg.ThreadID != 0 {
		tid := int64(msg.ThreadID)
		return &tid
	}
	return nil
}

// appendJSONL appends a Message as a JSON line to the file at path.
// Uses file locking (flock) to prevent corruption from concurrent readers.
func appendJSONL(path string, m Message) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open jsonl file: %w", err)
	}
	defer f.Close()

	// Acquire exclusive lock
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	data = append(data, '\n')

	_, err = f.Write(data)
	return err
}

// transcribeVoice runs transcribe.py on the given audio file and returns the
// transcription text. Returns empty string if transcription is unavailable or fails.
func transcribeVoice(voicePath string) string {
	// Find transcribe.py next to the current executable
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("transcribe: cannot find executable path: %v", err)
		return ""
	}
	scriptPath := filepath.Join(filepath.Dir(exePath), "transcribe.py")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// Also try current working directory
		if wd, wdErr := os.Getwd(); wdErr == nil {
			scriptPath = filepath.Join(wd, "transcribe.py")
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				log.Printf("transcribe: transcribe.py not found")
				return ""
			}
		} else {
			log.Printf("transcribe: transcribe.py not found")
			return ""
		}
	}

	cmd := exec.Command("uv", "run", scriptPath, voicePath)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		log.Printf("transcribe: failed: %v", err)
		return ""
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return ""
	}
	return text
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

	owner := ownerID()
	if owner != 0 {
		log.Printf("filtering messages to owner_id=%d", owner)
	}
	if listenJSONLPath != "" {
		log.Printf("writing messages to %s", listenJSONLPath)
	}

	bot.Handle(tele.OnText, func(c tele.Context) error {
		msg := c.Message()
		if owner != 0 && msg.Sender.ID != owner {
			log.Printf("[ignored] %s (id=%d): %s", msg.Sender.FirstName, msg.Sender.ID, msg.Text)
			return nil
		}
		log.Printf("[%s] %s: %s", msg.Chat.Title, msg.Sender.FirstName, msg.Text)

		upsertChatFromMsg(db, msg)

		m := Message{
			TelegramMsgID: int64(msg.ID),
			ChatID:        msg.Chat.ID,
			TopicID:       topicFromMsg(msg),
			SenderName:    msg.Sender.FirstName,
			SenderID:      msg.Sender.ID,
			Content:       msg.Text,
			Timestamp:     msg.Time().UTC().Format(time.RFC3339),
		}

		id, err := db.InsertMessageReturningID(m)
		if err != nil {
			return err
		}

		if listenJSONLPath != "" {
			m.ID = id
			if err := appendJSONL(listenJSONLPath, m); err != nil {
				log.Printf("error writing jsonl: %v", err)
			}
		}

		return nil
	})

	bot.Handle(tele.OnVoice, func(c tele.Context) error {
		msg := c.Message()
		if owner != 0 && msg.Sender.ID != owner {
			log.Printf("[ignored-voice] %s (id=%d)", msg.Sender.FirstName, msg.Sender.ID)
			return nil
		}
		log.Printf("[%s] %s: [voice %ds]", msg.Chat.Title, msg.Sender.FirstName, msg.Voice.Duration)

		// Download voice file
		reader, err := bot.File(&msg.Voice.File)
		if err != nil {
			log.Printf("error downloading voice: %v", err)
			return nil
		}
		defer reader.Close()

		filename := fmt.Sprintf("%d_%d.ogg", msg.Chat.ID, msg.ID)
		outPath := filepath.Join(voiceDir(), filename)
		f, err := os.Create(outPath)
		if err != nil {
			log.Printf("error creating voice file: %v", err)
			return nil
		}
		if _, err := io.Copy(f, reader); err != nil {
			f.Close()
			log.Printf("error saving voice file: %v", err)
			return nil
		}
		f.Close()

		log.Printf("saved voice to %s", outPath)

		upsertChatFromMsg(db, msg)

		content := "[voice message]"
		if msg.Caption != "" {
			content = msg.Caption
		}

		// Transcribe if requested and content is still the default placeholder
		if listenTranscribe && content == "[voice message]" {
			if transcribed := transcribeVoice(outPath); transcribed != "" {
				content = transcribed
				log.Printf("transcribed voice: %s", content)
			}
		}

		m := Message{
			TelegramMsgID: int64(msg.ID),
			ChatID:        msg.Chat.ID,
			TopicID:       topicFromMsg(msg),
			SenderName:    msg.Sender.FirstName,
			SenderID:      msg.Sender.ID,
			Content:       content,
			VoicePath:     outPath,
			Timestamp:     msg.Time().UTC().Format(time.RFC3339),
		}

		id, err := db.InsertMessageReturningID(m)
		if err != nil {
			return err
		}

		if listenJSONLPath != "" {
			m.ID = id
			if err := appendJSONL(listenJSONLPath, m); err != nil {
				log.Printf("error writing jsonl: %v", err)
			}
		}

		return nil
	})

	bot.Handle("/chatid", func(c tele.Context) error {
		return c.Reply(fmt.Sprintf("Chat ID: `%d`", c.Chat().ID), &tele.SendOptions{ParseMode: tele.ModeMarkdown})
	})

	bot.Handle("/ping", func(c tele.Context) error {
		return c.Reply("pong")
	})

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
