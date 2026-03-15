package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	tele "gopkg.in/telebot.v4"
)

var sendTopicID int
var sendReplyTo int

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
		if sendReplyTo != 0 {
			opts.ReplyTo = &tele.Message{ID: sendReplyTo}
		}

		msg, err := bot.Send(chat, text, opts)
		if err != nil {
			return err
		}
		fmt.Printf("{\"message_id\": %d, \"chat_id\": %d}\n", msg.ID, chatID)
		return nil
	},
}

func init() {
	sendCmd.Flags().IntVar(&sendTopicID, "topic", 0, "forum topic ID")
	sendCmd.Flags().IntVar(&sendReplyTo, "reply-to", 0, "message ID to reply to")
	rootCmd.AddCommand(sendCmd)
}
