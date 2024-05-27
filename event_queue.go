package euphoria

import "sync"

type EventQueue interface {
	Push(event *Event)
	Pop()
	Front() *Event
	Count() int
	Empty() bool
	Recovery(events []*Event)
	Lock()
	Unlock()
}

type EventQueueImpl struct {
	sync.Mutex
	Queue []*Event
}

func (m *EventQueueImpl) Push(event *Event) {
	m.Queue = append(m.Queue, event)
}

func (m *EventQueueImpl) Pop() {
	m.Queue = m.Queue[1:]
}

func (m *EventQueueImpl) Front() *Event {
	return m.Queue[0]
}

func (m *EventQueueImpl) Count() int {
	return len(m.Queue)
}

func (m *EventQueueImpl) Empty() bool {
	return m.Count() == 0
}

func (m *EventQueueImpl) Recovery(events []*Event) {
	m.Queue = append(events, m.Queue...)
}
