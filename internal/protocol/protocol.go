// Package protocol defines the JSON wire format exchanged between the sshh
// server and client over an SSH "chat" channel.
package protocol

import "time"

// ChannelType is the SSH channel type used for chat sessions.
const ChannelType = "chat"

// Client -> server message types.
const (
	MsgJoin      = "join"
	MsgSend      = "send"
	MsgListRooms = "list_rooms"
)

// Server -> client message types.
const (
	MsgAck     = "ack"
	MsgError   = "error"
	MsgMessage = "message"
	MsgRooms   = "rooms"
)

// ClientMessage is sent from the client to the server.
type ClientMessage struct {
	Type string `json:"type"`
	Room string `json:"room,omitempty"`
	Body string `json:"body,omitempty"`
}

// ServerMessage is sent from the server to the client.
type ServerMessage struct {
	Type      string    `json:"type"`
	Room      string    `json:"room,omitempty"`
	Sender    string    `json:"sender,omitempty"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Rooms     []string  `json:"rooms,omitempty"`
	Error     string    `json:"error,omitempty"`
}
