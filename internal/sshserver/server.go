package sshserver

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/db"
	"github.com/mennymendoza/sshh/internal/room"
)

const chatChannelType = "chat"

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

func LoadOrGenerateHostKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return ssh.ParsePrivateKey(data)
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(priv)
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
		if newChannel.ChannelType() != chatChannelType {
			newChannel.Reject(ssh.UnknownChannelType, `only "chat" channels are supported`)
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go ssh.DiscardRequests(requests)
		go s.handleSession(channel, conn.User())
	}
}

type clientMessage struct {
	Type string `json:"type"`
	Room string `json:"room,omitempty"`
	Body string `json:"body,omitempty"`
}

type serverMessage struct {
	Type      string    `json:"type"`
	Room      string    `json:"room,omitempty"`
	Sender    string    `json:"sender,omitempty"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Rooms     []string  `json:"rooms,omitempty"`
	Error     string    `json:"error,omitempty"`
}

func (s *Server) handleSession(channel ssh.Channel, username string) {
	defer channel.Close()

	out := make(chan []byte, 32)
	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case payload, ok := <-out:
				if !ok {
					return
				}
				if _, err := channel.Write(payload); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	send := func(msg serverMessage) {
		payload, err := json.Marshal(msg)
		if err != nil {
			return
		}
		payload = append(payload, '\n')
		select {
		case out <- payload:
		case <-done:
		}
	}

	var (
		currentRoom string
		leaveRoom   func()
	)
	defer func() {
		if leaveRoom != nil {
			leaveRoom()
		}
	}()

	joinRoom := func(name string) {
		if _, err := s.db.CreateRoom(name); err != nil {
			send(serverMessage{Type: "error", Error: err.Error()})
			return
		}
		if leaveRoom != nil {
			leaveRoom()
		}
		sub, leave := s.rooms.Join(name)
		currentRoom = name
		leaveRoom = leave
		go func() {
			for payload := range sub {
				select {
				case out <- payload:
				case <-done:
					return
				}
			}
		}()
		send(serverMessage{Type: "ack"})
	}

	scanner := bufio.NewScanner(channel)
	for scanner.Scan() {
		var msg clientMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			send(serverMessage{Type: "error", Error: "invalid json"})
			continue
		}

		switch msg.Type {
		case "join":
			if msg.Room == "" {
				send(serverMessage{Type: "error", Error: "room is required"})
				continue
			}
			joinRoom(msg.Room)

		case "send":
			if currentRoom == "" || msg.Room != currentRoom {
				send(serverMessage{Type: "error", Error: "not joined to room"})
				continue
			}
			roomRow, err := s.db.CreateRoom(msg.Room)
			if err != nil {
				send(serverMessage{Type: "error", Error: err.Error()})
				continue
			}
			m, err := s.db.InsertMessage(roomRow.ID, username, msg.Body)
			if err != nil {
				send(serverMessage{Type: "error", Error: err.Error()})
				continue
			}
			payload, err := json.Marshal(serverMessage{
				Type:      "message",
				Room:      msg.Room,
				Sender:    m.Sender,
				Body:      m.Body,
				CreatedAt: m.CreatedAt,
			})
			if err != nil {
				continue
			}
			s.rooms.Broadcast(msg.Room, append(payload, '\n'))

		case "list_rooms":
			rooms, err := s.db.ListRooms()
			if err != nil {
				send(serverMessage{Type: "error", Error: err.Error()})
				continue
			}
			names := make([]string, len(rooms))
			for i, r := range rooms {
				names[i] = r.Name
			}
			send(serverMessage{Type: "rooms", Rooms: names})

		default:
			send(serverMessage{Type: "error", Error: "unknown message type"})
		}
	}
}
