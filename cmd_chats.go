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
