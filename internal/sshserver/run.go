package sshserver

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mennymendoza/sshh/internal/db"
	"github.com/mennymendoza/sshh/internal/room"
)

func Run(addr, dbPath, hostKeyPath string) error {
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer database.Close()

	hostKey, err := LoadOrGenerateHostKey(hostKeyPath)
	if err != nil {
		return fmt.Errorf("load host key: %w", err)
	}

	rooms := room.NewRegistry()
	server := NewServer(hostKey, rooms, database)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	defer listener.Close()

	go func() {
		log.Printf("listening on %s", addr)
		if err := server.Serve(listener); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("shutting down")
	return nil
}
