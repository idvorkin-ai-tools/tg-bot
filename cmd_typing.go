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
			Token:   token,
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
