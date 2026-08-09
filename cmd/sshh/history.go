package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mennymendoza/sshh/internal/historyclient"
)

func newHistoryCmd() *cobra.Command {
	var (
		addr     string
		user     string
		room     string
		keyPath  string
		asJSON   bool
		page     uint
		pageSize uint
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Fetch and decrypt a room's stored message history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if page == 0 {
				return fmt.Errorf("--page must be at least 1")
			}
			if pageSize == 0 {
				return fmt.Errorf("--pagesize must be at least 1")
			}
			return historyclient.Run(addr, user, room, keyPath, asJSON, page, pageSize)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "localhost:2222", "server address")
	cmd.Flags().StringVar(&user, "user", "", "only show messages from this user (optional)")
	cmd.Flags().StringVar(&room, "room", "general", "room to fetch history for")
	cmd.Flags().StringVar(&keyPath, "key", "", "path to the X25519 private key PEM file used to decrypt history (required)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output messages as indented JSON")
	cmd.Flags().UintVar(&page, "page", 1, "page number to fetch, starting at 1")
	cmd.Flags().UintVar(&pageSize, "pagesize", 100, "number of messages per page")
	cmd.MarkFlagRequired("key")

	return cmd
}
