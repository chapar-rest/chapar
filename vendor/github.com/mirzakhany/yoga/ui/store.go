package ui

type widgetStore struct {
	items map[string]any
	used  map[string]struct{}
}

func newStore() *widgetStore {
	return &widgetStore{
		items: make(map[string]any),
		used:  make(map[string]struct{}),
	}
}

// Widget returns the retained micro-state for id, allocating it with alloc on
// first use. Ids must be unique per window. App data (labels, documents) is
// not stored here.
func (c *Ctx) Widget(id string, alloc func() any) any {
	if c.store == nil {
		c.store = newStore()
	}
	c.store.used[id] = struct{}{}
	if v, ok := c.store.items[id]; ok {
		return v
	}
	v := alloc()
	c.store.items[id] = v
	return v
}

// EndFrame drops widget-store entries that were not visited this drawn frame.
func (c *Ctx) EndFrame() {
	if c.store == nil {
		return
	}
	for id := range c.store.items {
		if _, ok := c.store.used[id]; !ok {
			delete(c.store.items, id)
		}
	}
	clear(c.store.used)
}

func (c *Ctx) beginStorePass() {
	if c.store == nil {
		c.store = newStore()
		return
	}
	// Two BuildFrame calls share one drawn frame (input then paint). Keep
	// used-set accumulating across both; EndFrame GCs after paint.
}
