package bchan

import (
	"github.com/stretchr/testify/assert"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestCancelAndSend(t *testing.T) {
	sender := make(chan struct{})
	cancel := make(chan struct{})
	close(cancel)
	for i := 0; i < 1_000; i++ {
		select {
		case <-cancel:
		case sender <- struct{}{}:
		}
	}
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	const Threads = 16
	const Iterations = 10_000
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		go func() {
			addedSubs := make([]UnsubscribeToken, 0)
			for j := 0; j < Iterations; j++ {
				// Always add a sub if sub count is zero.
				// Otherwise, randomly add another or remove one of the old ones
				if len(addedSubs) == 0 || rng.Int()%2 == 0 {
					_, unsubToken := bchan.Subscribe(0)
					addedSubs = append(addedSubs, unsubToken)
				} else {
					bchan.Unsubscribe(addedSubs[len(addedSubs)-1])
					addedSubs = addedSubs[:len(addedSubs)-1]
				}
			}
			for _, sub := range addedSubs {
				bchan.Unsubscribe(sub)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	assert.Len(t, bchan.registeredChannels, 0)
}

func TestBroadcast1(t *testing.T) {
	const Threads = 8
	const DataCount = 1000
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		sub, unsubToken := bchan.Subscribe(0)
		go func() {
			for expected := 0; expected < DataCount; expected++ {
				assert.Equal(t, expected, <-sub)
			}
			bchan.Unsubscribe(unsubToken)
			wg.Done()
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.Broadcast(data)
	}
	// Wait for them to be received
	wg.Wait()
}

func TestBroadcast2(t *testing.T) {
	// Same as TestBroadcast1 but unsubs halfway through
	const Threads = 8
	const DataCount = 1000
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		sub, unsubToken := bchan.Subscribe(0)
		go func() {
			for expected := 0; expected < DataCount/2; expected++ {
				assert.Equal(t, expected, <-sub)
			}
			bchan.Unsubscribe(unsubToken)
			wg.Done()
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.Broadcast(data)
	}
	// Wait for them to be received
	wg.Wait()
}

func TestBroadcast3(t *testing.T) {
	// Randomly sub and unsub while reading data
	const Threads = 8
	const DataCount = 100_000
	const Timeout = time.Millisecond * 100
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		go func() {
			lastNumber := -1
			for {
				timer := time.NewTimer(Timeout)
				sub, unsubToken := bchan.Subscribe(0)
				select {
				case <-timer.C: // timeout. get out
					wg.Done()
					return
				case data := <-sub: // new data
					timer.Stop()
					assert.Greater(t, data, lastNumber)
					lastNumber = data
				}
				bchan.Unsubscribe(unsubToken)
			}
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.Broadcast(data)
	}
	// Wait for them to be received
	wg.Wait()
}

func TestBroadcast4(t *testing.T) {
	// Randomly sub, don't do anything, then unsub
	const Threads = 8
	const DataCount = 100_000
	bchan := NewBroadcastChannel[int]()
	doneChannel := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		go func() {
			for {
				_, unsubToken := bchan.Subscribe(0)
				select {
				case <-doneChannel: // we are done
					wg.Done()
					return
				default:
					// do nothing
				}
				runtime.Gosched()
				bchan.Unsubscribe(unsubToken)
			}
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.Broadcast(data)
	}
	close(doneChannel)
	// Wait for them to be received
	wg.Wait()
}

func TestTryBroadcast1(t *testing.T) {
	const Threads = 8
	const DataCount = 1000
	const Timeout = time.Millisecond * 100
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		sub, unsubToken := bchan.Subscribe(0)
		go func() {
			lastNumber := -1
			for {
				timer := time.NewTimer(Timeout)
				select {
				case <-timer.C: // timeout. get out
					bchan.Unsubscribe(unsubToken)
					wg.Done()
					return
				case data := <-sub: // new data
					timer.Stop()
					assert.Greater(t, data, lastNumber)
					lastNumber = data
				}

			}
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.TryBroadcast(data)
	}
	// Wait for them to be received
	wg.Wait()
}

func TestTryBroadcast2(t *testing.T) {
	const Threads = 8
	const DataCount = 1000
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		sub, unsubToken := bchan.Subscribe(DataCount)
		go func() {
			for expected := 0; expected < DataCount; expected++ {
				assert.Equal(t, expected, <-sub)
			}
			bchan.Unsubscribe(unsubToken)
			wg.Done()
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.TryBroadcast(data)
	}
	// Wait for them to be received
	wg.Wait()
}

func TestTryBroadcast3(t *testing.T) {
	// Randomly sub and unsub while reading data
	const Threads = 8
	const DataCount = 100_000
	const Timeout = time.Millisecond * 100
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		go func() {
			lastNumber := -1
			for {
				timer := time.NewTimer(Timeout)
				sub, unsubToken := bchan.Subscribe(0)
				select {
				case <-timer.C: // timeout. get out
					wg.Done()
					return
				case data := <-sub: // new data
					timer.Stop()
					assert.Greater(t, data, lastNumber)
					lastNumber = data
				}
				bchan.Unsubscribe(unsubToken)
			}
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.TryBroadcast(data)
	}
	// Wait for them to be received
	wg.Wait()
}

func TestTryBroadcast4(t *testing.T) {
	// Randomly sub, don't do anything, then unsub
	const Threads = 8
	const DataCount = 100_000
	bchan := NewBroadcastChannel[int]()
	doneChannel := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(Threads)
	for i := 0; i < Threads; i++ {
		go func() {
			for {
				_, unsubToken := bchan.Subscribe(0)
				select {
				case <-doneChannel: // we are done
					wg.Done()
					return
				default:
					// do nothing
				}
				runtime.Gosched()
				bchan.Unsubscribe(unsubToken)
			}
		}()
	}
	// Send data
	for data := 0; data < DataCount; data++ {
		bchan.TryBroadcast(data)
	}
	close(doneChannel)
	// Wait for them to be received
	wg.Wait()
}

func Example1() {
	// Create the broadcast channel
	bchan := NewBroadcastChannel[int]()
	wg := new(sync.WaitGroup)
	wg.Add(4)
	// In 4 threads receive the data
	for threadID := 0; threadID < 4; threadID++ {
	    // Subscribe to the broadcast channel
		sub, unsubToken := bchan.Subscribe(0)
		go func(threadID int) {
			for i := 0; i < 5; i++ {
				fmt.Printf("Thread %d received %d\n", threadID, <-sub)
			}
	        // Unsubscribe when we are done
			bchan.Unsubscribe(unsubToken)
			wg.Done()
		}(threadID)
	}
	// Send data to threads
	for data := 0; data < 5; data++ {
		bchan.Broadcast(data)
	}
	// Wait threads to finish
	wg.Wait()
}