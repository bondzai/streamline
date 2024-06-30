package repositories

import (
	"sse-server/internal/entities"
	"sync"
)

type InMemoryEventRepository interface {
	Add(event *entities.Event)
	GetAll() []*entities.Event
	GetByID(id string) (*entities.Event, bool)
	Remove(id string)
}

type inMemoryEventRepository struct {
	events map[string]*entities.Event
	mu     sync.RWMutex
}

func NewInMemoryJobRepository() *inMemoryEventRepository {
	return &inMemoryEventRepository{
		events: make(map[string]*entities.Event),
	}
}

func (r *inMemoryEventRepository) Add(event *entities.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[event.Id] = event
}

func (r *inMemoryEventRepository) GetAll() []*entities.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var events []*entities.Event
	for _, event := range r.events {
		events = append(events, event)
	}

	return events
}

func (r *inMemoryEventRepository) GetByID(id string) (*entities.Event, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	event, exists := r.events[id]

	return event, exists
}

func (r *inMemoryEventRepository) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.events, id)
}
