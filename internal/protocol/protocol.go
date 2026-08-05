package protocol

import "time"

const ChannelType = "chat"

const (
	MsgJoin      = "join"
	MsgSend      = "send"
	MsgListRooms = "list_rooms"
)

const (
	MsgAck        = "ack"
	MsgError      = "error"
	MsgMessage    = "message"
	MsgRooms      = "rooms"
	MsgUserJoined = "user_joined"
	MsgUserLeft   = "user_left"
)

type ClientMessage struct {
	Type string `json:"type"`
	Room string `json:"room,omitempty"`
	Body string `json:"body,omitempty"`
}

type ServerMessage struct {
	Type      string    `json:"type"`
	Room      string    `json:"room,omitempty"`
	Sender    string    `json:"sender,omitempty"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Rooms     []string  `json:"rooms,omitempty"`
	Error     string    `json:"error,omitempty"`
}
