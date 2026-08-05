package historyclient

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/cryptox"
	"github.com/mennymendoza/sshh/internal/protocol"
)

func Run(addr, sender, room, keyPath string) error {
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

	if err := writeMessage(channel, protocol.ClientMessage{Type: protocol.MsgHistory, Room: room}); err != nil {
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
			return printHistory(priv, msg, sender)
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

func printHistory(priv *[32]byte, msg protocol.ServerMessage, sender string) error {
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

	if len(entries) == 0 {
		if sender != "" {
			fmt.Fprintf(os.Stdout, "no messages from %s in %s\n", sender, msg.Room)
		} else {
			fmt.Fprintf(os.Stdout, "no messages in %s\n", msg.Room)
		}
		return nil
	}
	for _, entry := range entries {
		ciphertext, err := base64.StdEncoding.DecodeString(entry.Body)
		if err != nil {
			return fmt.Errorf("decode message body from %s: %w", entry.Sender, err)
		}
		plaintext, err := cryptox.Decrypt(priv, ciphertext)
		if err != nil {
			return fmt.Errorf("decrypt message from %s: %w", entry.Sender, err)
		}
		fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"), entry.Sender, plaintext)
	}
	return nil
}
