package sshclient

import (
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/protocol"
)

func Dial(addr, user string) (*ssh.Client, ssh.Channel, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password("test")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	channel, requests, err := client.OpenChannel(protocol.ChannelType, nil)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("open chat channel: %w", err)
	}
	go ssh.DiscardRequests(requests)

	return client, channel, nil
}
