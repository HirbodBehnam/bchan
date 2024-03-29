# bchan

[![Run Tests](https://github.com/HirbodBehnam/bchan/actions/workflows/test.yml/badge.svg)](https://github.com/HirbodBehnam/bchan/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/HirbodBehnam/bchan.svg)](https://pkg.go.dev/github.com/HirbodBehnam/bchan)

A dead simple broadcast channel in Golang

## What

In Golang, each data sent in a channel is received once and only once. This makes channels the perfect
choice for creating job distribution channels. However, in some cases you might want different threads to
receive same value (the value is broadcasted to all threads). In this case, this library comes in handy.

I tried to make a general usage library and added features that could come in handy for everyone. Such as

* Generic channels
* Thread safe
* Cancel safe
* Blocking and non-blocking broadcast
* Defining the receive buffer

Something which it does currently lack is the ability to close all subscribed channels.
It will add complexity to the library because I have to prevent double closes and such.
I might try to add it later but right now, I don't need to close the whole broadcast channel in my usage.

## Documentation

The documentation is available at [GoDoc](https://pkg.go.dev/github.com/HirbodBehnam/bchan).

## Example Usage

A very simple usage can be seen in the tests which looks like this:
```go
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
```