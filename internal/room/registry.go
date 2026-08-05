package room

import "sync"

type Registry struct {
	mu    sync.Mutex
	rooms map[string]map[chan []byte]struct{}
}

func NewRegistry() *Registry {
	return &Registry{rooms: make(map[string]map[chan []byte]struct{})}
}

func (r *Registry) Join(roomName string) (sub chan []byte, leave func()) {
	sub = make(chan []byte, 16)

	r.mu.Lock()
	subs, ok := r.rooms[roomName]
	if !ok {
		subs = make(map[chan []byte]struct{})
		r.rooms[roomName] = subs
	}
	subs[sub] = struct{}{}
	r.mu.Unlock()

	leave = func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if subs, ok := r.rooms[roomName]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(r.rooms, roomName)
			}
		}
		close(sub)
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
