package protocol

import (
	"encoding/json"
	"time"
)

const ChannelType = "chat"

const (
	MsgJoin      = "join"
	MsgSend      = "send"
	MsgListRooms = "list_rooms"
	MsgListUsers = "list_users"
	MsgHistory   = "history"
)

const (
	MsgAck           = "ack"
	MsgError         = "error"
	MsgMessage       = "message"
	MsgRooms         = "rooms"
	MsgUsers         = "users"
	MsgUserJoined    = "user_joined"
	MsgUserLeft      = "user_left"
	MsgHistoryResult = "history"
)

type ClientMessage struct {
	Type     string `json:"type"`
	Room     string `json:"room,omitempty"`
	Body     string `json:"body,omitempty"`
	Quiet    bool   `json:"quiet,omitempty"`
	Sender   string `json:"sender,omitempty"`
	Page     uint   `json:"page,omitempty"`
	PageSize uint   `json:"page_size,omitempty"`
}

type ServerMessage struct {
	Type      string         `json:"type"`
	Room      string         `json:"room,omitempty"`
	Sender    string         `json:"sender,omitempty"`
	Body      string         `json:"body,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
	Rooms     []string       `json:"rooms,omitempty"`
	Users     []string       `json:"users,omitempty"`
	Error     string         `json:"error,omitempty"`
	Messages  []HistoryEntry `json:"messages,omitempty"`
	Page      uint           `json:"page,omitempty"`
	PageSize  uint           `json:"page_size,omitempty"`
}

type HistoryEntry struct {
	Sender    string    `json:"sender"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func Encode(msg any) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append(payload, '\n'), nil
}
