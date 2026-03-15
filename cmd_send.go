package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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
			Token:   token,
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
		fmt.Printf("{\"message_id\": %d, \"chat_id\": %d}\n", msg.ID, chatID)

		// Store outgoing message in database
		db, err := OpenDB(dbPath())
		if err != nil {
			log.Printf("warning: could not open db to store sent message: %v", err)
			return nil
		}
		defer db.Close()

		var topicID *int64
		if sendTopicID != 0 {
			tid := int64(sendTopicID)
			topicID = &tid
		}

		if err := db.InsertSentMessage(Message{
			TelegramMsgID: int64(msg.ID),
			ChatID:        chatID,
			TopicID:       topicID,
			SenderName:    "tg-bot",
			SenderID:      0,
			Content:       text,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			log.Printf("warning: could not store sent message: %v", err)
		}

		return nil
	},
}

func init() {
	sendCmd.Flags().IntVar(&sendTopicID, "topic", 0, "forum topic ID")
	rootCmd.AddCommand(sendCmd)
}
