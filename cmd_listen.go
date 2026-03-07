package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Start listening for Telegram messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "listen: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)
}
