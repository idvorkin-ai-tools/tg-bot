package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
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

func ownerID() int64 {
	s := os.Getenv("TGBOT_OWNER_ID")
	if s == "" {
		return 0
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Fatalf("invalid TGBOT_OWNER_ID %q: %v", s, err)
	}
	return id
}
