package main

import (
	"encoding/json"
	"fmt"

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
