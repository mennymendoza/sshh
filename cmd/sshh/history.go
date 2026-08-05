package main

import (
	"github.com/spf13/cobra"

	"github.com/mennymendoza/sshh/internal/historyclient"
)

func newHistoryCmd() *cobra.Command {
	var (
		addr    string
		user    string
		room    string
		keyPath string
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Fetch and decrypt a room's stored message history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return historyclient.Run(addr, user, room, keyPath)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "localhost:2222", "server address")
	cmd.Flags().StringVar(&user, "user", "", "only show messages from this user (optional)")
	cmd.Flags().StringVar(&room, "room", "general", "room to fetch history for")
	cmd.Flags().StringVar(&keyPath, "key", "", "path to the X25519 private key PEM file used to decrypt history (required)")
	cmd.MarkFlagRequired("key")

	return cmd
}
