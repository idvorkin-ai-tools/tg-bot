package main

import (
	"fmt"
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

func ownerID() int64 {
	s := os.Getenv("TGBOT_OWNER_ID")
	if s == "" {
		return 0
	}
	var id int64
	fmt.Sscanf(s, "%d", &id)
	return id
}
