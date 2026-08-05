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
		message  string
		stream   bool
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
			if message != "" {
				return tui.RunOnce(addr, username, room, message)
			}
			if stream {
				return tui.RunStream(addr, username, room)
			}
			return tui.Run(addr, username, room)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "localhost:2222", "server address")
	cmd.Flags().StringVar(&username, "user", "", "chat username (defaults to the current OS user)")
	cmd.Flags().StringVar(&room, "room", "general", "room to join on connect")
	cmd.Flags().StringVar(&message, "message", "", "send a single one-shot message to the room and exit, without opening a session")
	cmd.Flags().BoolVar(&stream, "stream", false, "join the room and print incoming messages to stdout as \"sender: body\", without opening the TUI")

	return cmd
}
