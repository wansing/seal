package seal

import "sync"

// A Broker implements local publish/subscribe.
type Broker[T any] struct {
	data T // if subscribe is called after publish
	sub  []func(data T)
	lock sync.Mutex
}

func (b *Broker[T]) Publish(data T) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.data = data
	for _, fn := range b.sub {
		fn(data)
	}
}

// Subscribe always calls fn with cached data
func (b *Broker[T]) Subscribe(fn func(data T)) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sub = append(b.sub, fn)
	fn(b.data)
}
