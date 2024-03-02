package bus

var bus *Bus

type Bus struct {
	subscribers map[string][]func(any)
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string][]func(any)),
	}
}

func Subscribe(topic string, fn func(any)) {
	if bus == nil {
		return
	}

	bus.Subscribe(topic, fn)
}

func (b *Bus) Subscribe(topic string, fn func(any)) {
	b.subscribers[topic] = append(b.subscribers[topic], fn)
}

func Publish(topic string, data any) {
	if bus == nil {
		return
	}

	bus.Publish(topic, data)
}

func (b *Bus) Publish(topic string, data any) {
	for _, fn := range b.subscribers[topic] {
		go fn(data)
	}
}

func Init() {
	bus = NewBus()
}
