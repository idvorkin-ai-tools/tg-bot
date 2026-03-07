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

	owner := ownerID()
	if owner != 0 {
		log.Printf("filtering messages to owner_id=%d", owner)
	}

	bot.Handle(tele.OnText, func(c tele.Context) error {
		msg := c.Message()
		if owner != 0 && msg.Sender.ID != owner {
			log.Printf("[ignored] %s (id=%d): %s", msg.Sender.FirstName, msg.Sender.ID, msg.Text)
			return nil
		}
		log.Printf("[%s] %s: %s", msg.Chat.Title, msg.Sender.FirstName, msg.Text)

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
