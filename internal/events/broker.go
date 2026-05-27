package events

import "sync"

const defaultBufferSize = 16

type SubscriptionID uint64

type Subscription[T any] struct {
	ID          SubscriptionID
	C           <-chan T
	unsubscribe func()
}

func (s *Subscription[T]) Unsubscribe() {
	if s == nil || s.unsubscribe == nil {
		return
	}
	s.unsubscribe()
}

type Broker[T any] struct {
	mu          sync.RWMutex
	bufferSize  int
	nextID      SubscriptionID
	subscribers map[SubscriptionID]chan T
	closed      bool
}

func NewBroker[T any](bufferSize int) *Broker[T] {
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}

	return &Broker[T]{
		bufferSize:  bufferSize,
		subscribers: make(map[SubscriptionID]chan T),
	}
}

func (b *Broker[T]) Subscribe() *Subscription[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		ch := make(chan T)
		close(ch)
		return &Subscription[T]{
			C:           ch,
			unsubscribe: func() {},
		}
	}

	b.nextID++
	id := b.nextID
	ch := make(chan T, b.bufferSize)
	b.subscribers[id] = ch

	var once sync.Once
	return &Subscription[T]{
		ID: id,
		C:  ch,
		unsubscribe: func() {
			once.Do(func() {
				b.mu.Lock()
				defer b.mu.Unlock()

				channel, ok := b.subscribers[id]
				if !ok {
					return
				}
				delete(b.subscribers, id)
				close(channel)
			})
		},
	}
}

func (b *Broker[T]) Publish(msg T) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return
	}

	channels := make([]chan T, 0, len(b.subscribers))
	for _, ch := range b.subscribers {
		channels = append(channels, ch)
	}
	b.mu.RUnlock()

	for _, ch := range channels {
		sendSafe(ch, msg)
	}
}

func (b *Broker[T]) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true

	channels := make([]chan T, 0, len(b.subscribers))
	for id, ch := range b.subscribers {
		channels = append(channels, ch)
		delete(b.subscribers, id)
	}
	b.mu.Unlock()

	for _, ch := range channels {
		close(ch)
	}
}

func sendSafe[T any](ch chan T, msg T) {
	defer func() {
		_ = recover()
	}()
	ch <- msg
}
