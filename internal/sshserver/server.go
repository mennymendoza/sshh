package sshserver

import (
	"net"

	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/db"
	"github.com/mennymendoza/sshh/internal/protocol"
	"github.com/mennymendoza/sshh/internal/room"
)

type Server struct {
	cfg   *ssh.ServerConfig
	rooms *room.Registry
	db    *db.DB
}

func NewServer(hostKey ssh.Signer, rooms *room.Registry, database *db.DB) *Server {
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}
	cfg.AddHostKey(hostKey)
	return &Server{cfg: cfg, rooms: rooms, db: database}
}

func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(nc net.Conn) {
	conn, chans, reqs, err := ssh.NewServerConn(nc, s.cfg)
	if err != nil {
		nc.Close()
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != protocol.ChannelType {
			newChannel.Reject(ssh.UnknownChannelType, `only "chat" channels are supported`)
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go ssh.DiscardRequests(requests)
		go s.newSession(channel, conn.User()).run()
	}
}
