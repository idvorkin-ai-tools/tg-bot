package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v4"
)

var sendVoiceCmd = &cobra.Command{
	Use:   "send-voice <chat_id> <ogg_file>",
	Short: "Send a voice message to a chat",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := botToken()
		if token == "" {
			return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
		}

		chatID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid chat_id: %w", err)
		}
		filePath := args[1]

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		bot, err := tele.NewBot(tele.Settings{
			Token:   token,
			Offline: true,
		})
		if err != nil {
			return fmt.Errorf("create bot: %w", err)
		}

		chat := &tele.Chat{ID: chatID}
		voice := &tele.Voice{File: tele.FromDisk(filePath)}
		msg, err := bot.Send(chat, voice)
		if err != nil {
			return fmt.Errorf("send voice: %w", err)
		}

		out, _ := json.Marshal(map[string]int64{
			"message_id": int64(msg.ID),
			"chat_id":    chatID,
		})
		fmt.Println(string(out))

		// Store outgoing voice message in database
		db, err := OpenDB(dbPath())
		if err != nil {
			log.Printf("warning: could not open db to store sent voice message: %v", err)
			return nil
		}
		defer db.Close()

		if err := db.InsertSentMessage(Message{
			TelegramMsgID: int64(msg.ID),
			ChatID:        chatID,
			SenderName:    "tg-bot",
			SenderID:      0,
			Content:       "[voice message]",
			VoicePath:     filePath,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			log.Printf("warning: could not store sent voice message: %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sendVoiceCmd)
}
