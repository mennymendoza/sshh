package sshserver

import (
	"bufio"
	"encoding/json"

	"golang.org/x/crypto/ssh"

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
	default:
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "unknown message type"})
	}
}

func (sess *session) handleJoin(msg protocol.ClientMessage) {
	if msg.Room == "" {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: "room is required"})
		return
	}
	if _, err := sess.server.db.CreateRoom(msg.Room); err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}

	if sess.leaveRoom != nil {
		sess.leaveRoom()
	}
	sub, leave := sess.server.rooms.Join(msg.Room)
	sess.currentRoom = msg.Room

	room, username := msg.Room, sess.username
	sess.leaveRoom = func() {
		leave()
		sess.broadcastUserEvent(protocol.MsgUserLeft, room, username)
	}
	go sess.relayLoop(sub)

	sess.broadcastUserEvent(protocol.MsgUserJoined, room, username)
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

	roomRow, err := sess.server.db.CreateRoom(msg.Room)
	if err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}
	m, err := sess.server.db.InsertMessage(roomRow.ID, sess.username, msg.Body)
	if err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}

	payload, err := json.Marshal(protocol.ServerMessage{
		Type:      protocol.MsgMessage,
		Room:      msg.Room,
		Sender:    m.Sender,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
	})
	if err != nil {
		return
	}
	sess.server.rooms.Broadcast(msg.Room, append(payload, '\n'))
}

func (sess *session) handleListRooms() {
	rooms, err := sess.server.db.ListRooms()
	if err != nil {
		sess.send(protocol.ServerMessage{Type: protocol.MsgError, Error: err.Error()})
		return
	}
	names := make([]string, len(rooms))
	for i, r := range rooms {
		names[i] = r.Name
	}
	sess.send(protocol.ServerMessage{Type: protocol.MsgRooms, Rooms: names})
}
