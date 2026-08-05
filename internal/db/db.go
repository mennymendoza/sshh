package db

import (
	_ "embed"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type DB struct {
	*sqlx.DB
}

type Room struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

type Message struct {
	ID        int64     `db:"id"`
	RoomID    int64     `db:"room_id"`
	Sender    string    `db:"sender"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
}

func Open(path string) (*DB, error) {
	conn, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{conn}, nil
}

const createRoomQuery = `
INSERT INTO rooms (name) VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING id, name, created_at`

func (d *DB) CreateRoom(name string) (Room, error) {
	var r Room
	err := d.Get(&r, createRoomQuery, name)
	return r, err
}

const listRoomsQuery = `SELECT id, name, created_at FROM rooms ORDER BY name`

func (d *DB) ListRooms() ([]Room, error) {
	var rooms []Room
	err := d.Select(&rooms, listRoomsQuery)
	return rooms, err
}

const insertMessageQuery = `
INSERT INTO messages (room_id, sender, body) VALUES (?, ?, ?)
RETURNING id, room_id, sender, body, created_at`

func (d *DB) InsertMessage(roomID int64, sender, body string) (Message, error) {
	var m Message
	err := d.Get(&m, insertMessageQuery, roomID, sender, body)
	return m, err
}

const listMessagesQuery = `
SELECT id, room_id, sender, body, created_at FROM messages
WHERE room_id = ? ORDER BY created_at`

func (d *DB) ListMessages(roomID int64) ([]Message, error) {
	var messages []Message
	err := d.Select(&messages, listMessagesQuery, roomID)
	return messages, err
}
