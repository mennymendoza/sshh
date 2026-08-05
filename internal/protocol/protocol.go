package protocol

import "time"

const ChannelType = "chat"

const (
	MsgJoin      = "join"
	MsgSend      = "send"
	MsgListRooms = "list_rooms"
	MsgHistory   = "history"
)

const (
	MsgAck           = "ack"
	MsgError         = "error"
	MsgMessage       = "message"
	MsgRooms         = "rooms"
	MsgUserJoined    = "user_joined"
	MsgUserLeft      = "user_left"
	MsgHistoryResult = "history"
)

type ClientMessage struct {
	Type  string `json:"type"`
	Room  string `json:"room,omitempty"`
	Body  string `json:"body,omitempty"`
	Quiet bool   `json:"quiet,omitempty"`
}

type ServerMessage struct {
	Type      string         `json:"type"`
	Room      string         `json:"room,omitempty"`
	Sender    string         `json:"sender,omitempty"`
	Body      string         `json:"body,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
	Rooms     []string       `json:"rooms,omitempty"`
	Error     string         `json:"error,omitempty"`
	Messages  []HistoryEntry `json:"messages,omitempty"`
}

type HistoryEntry struct {
	Sender    string    `json:"sender"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
