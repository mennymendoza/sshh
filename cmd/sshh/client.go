package main

import (
	"fmt"
	"os/user"

	"github.com/spf13/cobra"

	"github.com/mennymendoza/sshh/internal/tui"
)

func newClientCmd() *cobra.Command {
	var (
		addr     string
		username string
		room     string
	)

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Run the minimal sshh chat TUI client",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				current, err := user.Current()
				if err != nil {
					return fmt.Errorf("determine current OS user: %w", err)
				}
				username = current.Username
			}
			return tui.Run(addr, username, room)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "localhost:2222", "server address")
	cmd.Flags().StringVar(&username, "user", "", "chat username (defaults to the current OS user)")
	cmd.Flags().StringVar(&room, "room", "general", "room to join on connect")

	return cmd
}
