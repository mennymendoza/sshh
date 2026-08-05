package main

import (
	"github.com/spf13/cobra"

	"github.com/mennymendoza/sshh/internal/sshserver"
)

func newServerCmd() *cobra.Command {
	var (
		addr        string
		dbPath      string
		hostKeyPath string
		pubKeyPath  string
	)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the sshh chat server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sshserver.Run(addr, dbPath, hostKeyPath, pubKeyPath)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":2222", "address to listen on")
	cmd.Flags().StringVar(&dbPath, "db", "", "path to the SQLite database file (requires --pub)")
	cmd.Flags().StringVar(&hostKeyPath, "host-key", "host_key", "path to the SSH host key file")
	cmd.Flags().StringVar(&pubKeyPath, "pub", "", "path to an X25519 public key PEM file used to encrypt stored messages (requires --db)")
	cmd.MarkFlagsRequiredTogether("db", "pub")

	return cmd
}
