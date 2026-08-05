package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "sshh",
		Short: "SSH chat server and client",
	}
	root.AddCommand(newServerCmd())
	root.AddCommand(newClientCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
