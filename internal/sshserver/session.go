package sshserver

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/mennymendoza/sshh/internal/cryptox"
	"github.com/mennymendoza/sshh/internal/protocol"
)

type session struct {
	server   *Server
	channel  ssh.Channel
	username string

	out  chan []byte
	done chan struct{}

	currentRoom string
	leaveRoom   func()
}

func (s *Server) newSession(channel ssh.Channel, username string) *session {
	return &session{
		server:   s,
		channel:  channel,
		username: username,
		out:      make(chan []byte, 32),
		done:     make(chan struct{}),
	}
}

func (sess *session) run() {
	defer sess.channel.Close()
	defer close(sess.done)
	defer func() {
		if sess.leaveRoom != nil {
			sess.leaveRoom()
		}
	}()

	go sess.writeLoop()

	scanner := bufio.NewScanner(sess.channel)
	for scanner.Scan() {
		var msg protocol.ClientMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "invalid json"})
			continue
		}
		sess.handleMessage(msg)
	}
}

func (sess *session) writeLoop() {
	for {
		select {
		case payload, ok := <-sess.out:
			if !ok {
				return
			}
			if _, err := sess.channel.Write(payload); err != nil {
				return
			}
		case <-sess.done:
			return
		}
	}
}

func (sess *session) send(msg protocol.ServerMessage) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	payload = append(payload, '\n')
	select {
	case sess.out <- payload:
	case <-sess.done:
	}
}

func (sess *session) handleMessage(msg protocol.ClientMessage) {
	switch msg.Type {
	case protocol.MsgJoin:
		sess.handleJoin(msg)
	case protocol.MsgSend:
		sess.handleSend(msg)
	case protocol.MsgListRooms:
		sess.handleListRooms()
	case protocol.MsgListUsers:
		sess.handleListUsers()
	case protocol.MsgHistory:
		sess.handleHistory(msg)
	default:
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "unknown message type"})
	}
}

func (sess *session) handleJoin(msg protocol.ClientMessage) {
	if msg.Room == "" {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "room is required"})
		return
	}
	if sess.server.db != nil {
		if _, err := sess.server.db.CreateRoom(msg.Room); err != nil {
			sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
			return
		}
	}

	if sess.leaveRoom != nil {
		sess.leaveRoom()
	}
	sub, leave := sess.server.rooms.Join(msg.Room, sess.username)
	sess.currentRoom = msg.Room

	room, username := msg.Room, sess.username
	sess.leaveRoom = func() {
		leave()
		if !msg.Quiet {
			sess.broadcastUserEvent(protocol.MsgUserLeft, room, username)
		}
	}
	go sess.relayLoop(sub)

	if !msg.Quiet {
		sess.broadcastUserEvent(protocol.MsgUserJoined, room, username)
	}
	sess.send(protocol.ServerMessage{Type: protocol.MsgAck})
}

func (sess *session) broadcastUserEvent(msgType, room, username string) {
	payload, err := json.Marshal(protocol.ServerMessage{Type: msgType, Room: room, Sender: username})
	if err != nil {
		return
	}
	sess.server.rooms.Broadcast(room, append(payload, '\n'))
}

func (sess *session) relayLoop(sub chan []byte) {
	for payload := range sub {
		select {
		case sess.out <- payload:
		case <-sess.done:
			return
		}
	}
}

func (sess *session) handleSend(msg protocol.ClientMessage) {
	if sess.currentRoom == "" || msg.Room != sess.currentRoom {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "not joined to room"})
		return
	}

	sender := sess.username
	createdAt := time.Now().UTC()

	if sess.server.db != nil {
		roomRow, err := sess.server.db.CreateRoom(msg.Room)
		if err != nil {
			sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
			return
		}
		ciphertext, err := cryptox.Encrypt(sess.server.pubKey, []byte(msg.Body))
		if err != nil {
			sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
			return
		}
		m, err := sess.server.db.InsertMessage(roomRow.ID, sess.username, base64.StdEncoding.EncodeToString(ciphertext))
		if err != nil {
			sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
			return
		}
		sender = m.Sender
		createdAt = m.CreatedAt
	}

	payload, err := json.Marshal(protocol.ServerMessage{
		Type:      protocol.MsgMessage,
		Room:      msg.Room,
		Sender:    sender,
		Body:      msg.Body,
		CreatedAt: createdAt,
	})
	if err != nil {
		return
	}
	sess.server.rooms.Broadcast(msg.Room, append(payload, '\n'))
}

func (sess *session) handleListRooms() {
	if sess.server.db == nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "room listing unavailable: server was started without --db and --pub"})
		return
	}
	rooms, err := sess.server.db.ListRooms()
	if err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}
	active := sess.server.rooms.ActiveNames()
	names := make([]string, 0, len(rooms))
	for _, r := range rooms {
		if _, ok := active[r.Name]; ok {
			names = append(names, r.Name)
		}
	}
	sess.send(protocol.ServerMessage{Type: protocol.MsgRooms, Rooms: names})
}

func (sess *session) handleListUsers() {
	if sess.currentRoom == "" {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "not joined to room"})
		return
	}
	users := sess.server.rooms.Users(sess.currentRoom)
	sess.send(protocol.ServerMessage{Type: protocol.MsgUsers, Room: sess.currentRoom, Users: users})
}

func (sess *session) handleHistory(msg protocol.ClientMessage) {
	if sess.server.db == nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "no persistence configured on this server"})
		return
	}
	if msg.Room == "" {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "room is required"})
		return
	}

	page := msg.Page
	if page == 0 {
		page = 1
	}
	pageSize := msg.PageSize
	if pageSize == 0 {
		pageSize = 100
	}

	roomRow, err := sess.server.db.CreateRoom(msg.Room)
	if err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}
	messages, err := sess.server.db.ListMessages(roomRow.ID, page, pageSize)
	if err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}

	entries := make([]protocol.HistoryEntry, len(messages))
	for i, m := range messages {
		entries[i] = protocol.HistoryEntry{Sender: m.Sender, Body: m.Body, CreatedAt: m.CreatedAt}
	}
	sess.send(protocol.ServerMessage{Type: protocol.MsgHistoryResult, Room: msg.Room, Messages: entries, Page: page, PageSize: pageSize})
}
