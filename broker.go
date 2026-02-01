package seal

import "sync"

// A Broker implements local publish/subscribe.
type Broker struct {
	retain map[string]any              // collects data until Ready() is called
	sub    map[string][]func(data any) // key: urlpath
	lock   sync.Mutex
	ready  bool
}

func NewBroker() *Broker {
	return &Broker{
		retain: make(map[string]any),
		sub:    make(map[string][]func(data any)),
	}
}

func (b *Broker) Publish(urlpath string, data any) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.ready {
		for _, fn := range b.sub[urlpath] {
			fn(data)
		}
	} else {
		b.retain[urlpath] = data
	}
}

func (b *Broker) Subscribe(urlpath string, fn func(data any)) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sub[urlpath] = append(b.sub[urlpath], fn)
}

// Ready enables notifications and initially notifies all subscribers.
// No notifications are made before Ready is called.
func (b *Broker) Ready() {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.ready {
		return // Ready has already been called
	}

	b.ready = true
	for urlpath, fns := range b.sub {
		data := b.retain[urlpath]
		for _, fn := range fns {
			fn(data)
		}
	}
	clear(b.retain)
}
