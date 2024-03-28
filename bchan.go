package bchan

import "sync"

// UnsubscribeToken can be provided to BroadcastChannel.Unsubscribe in order to unsubscribe from
// the broadcast channel.
// Internally, this is simply a channel that would be closed before holding the lock for the registered channels.
type UnsubscribeToken chan struct{}

// BroadcastChannel is a generic type which represents a broadcast channel
type BroadcastChannel[T any] struct {
	// List of registered channels.
	// The key is the channel which user can read from to get the broadcast messages.
	// The value is a dummy channel which closes just before the sender channel is closed.
	// The reason that we do this, is that we could skip the closed channels while iterating.
	registeredChannels map[UnsubscribeToken]chan T
	// A mutex to lock the map
	mu sync.RWMutex
}

// NewBroadcastChannel will create a new BroadcastChannel
func NewBroadcastChannel[T any]() *BroadcastChannel[T] {
	return &BroadcastChannel[T]{
		registeredChannels: map[UnsubscribeToken]chan T{},
	}
}

// Subscribe to this broadcast channel to get the updates.
// Gets a buffer size to create buffered channels
func (bchan *BroadcastChannel[T]) Subscribe(buffer int) (chan T, UnsubscribeToken) {
	c := make(chan T, buffer)
	cancelChan := make(chan struct{})
	bchan.mu.Lock()
	bchan.registeredChannels[cancelChan] = c
	bchan.mu.Unlock()
	return c, cancelChan
}

// Unsubscribe from the broadcaster.
// This method must be called once and only once on each channel.
func (bchan *BroadcastChannel[T]) Unsubscribe(token UnsubscribeToken) {
	close(token)
	bchan.mu.Lock()
	c := bchan.registeredChannels[token]
	delete(bchan.registeredChannels, token)
	bchan.mu.Unlock()
	close(c)
}

// Broadcast will send a data in all subscribed channels.
// It will wait until all channels have received the data (blocks on send).
func (bchan *BroadcastChannel[T]) Broadcast(data T) {
	bchan.mu.RLock()
	for canceled, c := range bchan.registeredChannels {
		select {
		case c <- data:
			// data sent
		case <-canceled:
			// channel is going to close...
		}
	}
	bchan.mu.RUnlock()
}

// TryBroadcast will try to send a data in all subscribed channels.
// It will skip the channels which are not ready to receive data.
func (bchan *BroadcastChannel[T]) TryBroadcast(data T) {
	bchan.mu.RLock()
	for canceled, c := range bchan.registeredChannels {
		select {
		case c <- data:
			// data sent
		case <-canceled:
			// channel is going to close
		default:
			// buffer full or receiver not ready
		}
	}
	bchan.mu.RUnlock()
}
