package bus

var bus = NewBus()

type Bus struct {
	subscribers map[string][]chan any
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string][]chan any),
	}
}

func Subscribe(topic string) chan any {
	return bus.Subscribe(topic)
}

func (b *Bus) Subscribe(topic string) chan any {
	ch := make(chan any)
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	return ch
}

func Publish(topic string, data any) {
	bus.Publish(topic, data)
}

func (b *Bus) Publish(topic string, data any) {
	for _, ch := range b.subscribers[topic] {
		ch <- data
	}
}

func Unsubscribe(topic string, ch chan any) {
	bus.Unsubscribe(topic, ch)
}

func (b *Bus) Unsubscribe(topic string, ch chan any) {
	subscribers := b.subscribers[topic]
	for i, subscriber := range subscribers {
		if subscriber == ch {
			subscribers = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
	b.subscribers[topic] = subscribers
}

func (b *Bus) Close() {
	for _, subscribers := range b.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
	}
}

func (b *Bus) CloseTopic(topic string) {
	for _, ch := range b.subscribers[topic] {
		close(ch)
	}
	delete(b.subscribers, topic)
}

func (b *Bus) CloseAll() {
	for topic := range b.subscribers {
		b.CloseTopic(topic)
	}
}

func (b *Bus) SubscribersCount(topic string) int {
	return len(b.subscribers[topic])
}

func (b *Bus) SubscribersCountAll() int {
	count := 0
	for _, subscribers := range b.subscribers {
		count += len(subscribers)
	}
	return count
}

func (b *Bus) Topics() []string {
	topics := make([]string, 0, len(b.subscribers))
	for topic := range b.subscribers {
		topics = append(topics, topic)
	}
	return topics
}

func (b *Bus) TopicsCount() int {
	return len(b.subscribers)
}
