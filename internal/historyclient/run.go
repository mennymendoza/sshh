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
)

func Run(addr, sender, room, keyPath string, asJSON bool, page, pageSize uint) error {
	priv, err := cryptox.LoadPrivateKey(keyPath)
	if err != nil {
		return fmt.Errorf("load private key: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            "history",
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer client.Close()

	channel, requests, err := client.OpenChannel(protocol.ChannelType, nil)
	if err != nil {
		return fmt.Errorf("open chat channel: %w", err)
	}
	defer channel.Close()
	go ssh.DiscardRequests(requests)

	if err := writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgHistory, Room: room, Page: page, PageSize: pageSize}); err != nil {
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
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
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
	if sender != "" {
		filtered := entries[:0]
		for _, entry := range entries {
			if entry.Sender == sender {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}

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
