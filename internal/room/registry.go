package room

import (
	"sort"
	"sync"
)

type Registry struct {
	mu    sync.Mutex
	rooms map[string]map[chan []byte]string
}

func NewRegistry() *Registry {
	return &Registry{rooms: make(map[string]map[chan []byte]string)}
}

func (r *Registry) Join(roomName, username string) (sub chan []byte, leave func() (wasLast bool)) {
	sub = make(chan []byte, 16)

	r.mu.Lock()
	subs, ok := r.rooms[roomName]
	if !ok {
		subs = make(map[chan []byte]string)
		r.rooms[roomName] = subs
	}
	subs[sub] = username
	r.mu.Unlock()

	leave = func() (wasLast bool) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if subs, ok := r.rooms[roomName]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(r.rooms, roomName)
				wasLast = true
			}
		}
		close(sub)
		return wasLast
	}
	return sub, leave
}

func (r *Registry) Broadcast(roomName string, payload []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for sub := range r.rooms[roomName] {
		select {
		case sub <- payload:
		default:
		}
	}
}

func (r *Registry) ActiveNames() map[string]struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make(map[string]struct{}, len(r.rooms))
	for name := range r.rooms {
		names[name] = struct{}{}
	}
	return names
}

func (r *Registry) Users(roomName string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	seen := make(map[string]struct{})
	for _, username := range r.rooms[roomName] {
		seen[username] = struct{}{}
	}

	users := make([]string, 0, len(seen))
	for username := range seen {
		users = append(users, username)
	}
	sort.Strings(users)
	return users
}
