package main

import (
	"github.com/spf13/cobra"

	"github.com/mennymendoza/sshh/internal/tui"
)

func newClientCmd() *cobra.Command {
	var (
		addr string
		user string
		room string
	)

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Run the minimal sshh chat TUI client",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(addr, user, room)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "localhost:2222", "server address")
	cmd.Flags().StringVar(&user, "user", "", "chat username (required)")
	cmd.Flags().StringVar(&room, "room", "general", "room to join on connect")
	cmd.MarkFlagRequired("user")

	return cmd
}
