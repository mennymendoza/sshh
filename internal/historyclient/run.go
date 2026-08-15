package historyclient

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/cryptox"
	"github.com/mennymendoza/sshh/internal/protocol"
	"github.com/mennymendoza/sshh/internal/sshclient"
)

func Run(addr, sender, room, keyPath string, asJSON bool, page, pageSize uint) error {
	priv, err := cryptox.LoadPrivateKey(keyPath)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	client, channel, err := sshclient.Dial(addr, "history")
	if err != nil {
		return err
	}
	defer client.Close()
	defer channel.Close()

	if err := writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgHistory, Room: room, Sender: sender, Page: page, PageSize: pageSize}); err != nil {
		return fmt.Errorf("request history: %w", err)
	}

	scanner := bufio.NewScanner(channel)
	for scanner.Scan() {
		var msg protocol.ServerMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			return fmt.Errorf("decode server message: %w", err)
		}
		switch msg.Type {
		case protocol.MsgHistoryResult:
			return printHistory(priv, msg, sender, asJSON)
		case protocol.MsgError:
			return fmt.Errorf("server error: %s", msg.Error)
		}
	}
	return scanner.Err()
}

func writeMessage(channel ssh.Channel, msg protocol.ClientMessage) error {
	payload, err := protocol.Encode(msg)
	if err != nil {
		return err
	}
	_, err = channel.Write(payload)
	return err
}

type decryptedEntry struct {
	Sender    string    `json:"sender"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func printHistory(priv *[32]byte, msg protocol.ServerMessage, sender string, asJSON bool) error {
	entries := msg.Messages

	if !asJSON && len(entries) == 0 {
		if sender != "" {
			fmt.Fprintf(os.Stdout, "no messages from %s in %s (page %d)\n", sender, msg.Room, msg.Page)
		} else {
			fmt.Fprintf(os.Stdout, "no messages in %s (page %d)\n", msg.Room, msg.Page)
		}
		return nil
	}

	decrypted := make([]decryptedEntry, 0, len(entries))
	for _, entry := range entries {
		ciphertext, err := base64.StdEncoding.DecodeString(entry.Body)
		if err != nil {
			return fmt.Errorf("decode message body from %s: %w", entry.Sender, err)
		}
		plaintext, err := cryptox.Decrypt(priv, ciphertext)
		if err != nil {
			return fmt.Errorf("decrypt message from %s: %w", entry.Sender, err)
		}
		decrypted = append(decrypted, decryptedEntry{Sender: entry.Sender, Body: string(plaintext), CreatedAt: entry.CreatedAt})
	}

	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(decrypted)
	}

	for _, entry := range decrypted {
		fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"), entry.Sender, entry.Body)
	}
	return nil
}
