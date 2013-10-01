package main

import (
	"container/list"
	"sync"
)

type Event struct {
	Body string
}

type Broker struct {
	topics map[string]*list.List
	sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{topics: make(map[string]*list.List)}
}

func (b *Broker) Subscribe(topic string) <-chan Event {
	ch := make(chan Event)

	// If the topic doesn't exist, create it
	b.Lock()
	defer b.Unlock()
	if _, exists := b.topics[topic]; !exists {
		b.topics[topic] = list.New()
	}
	b.topics[topic].PushBack(ch)

	return ch
}

func (b *Broker) Unsubscribe(ch <-chan Event, topic string) {
	b.Lock()
	defer b.Unlock()
	if clients, exists := b.topics[topic]; exists {
		// Find and remove the chan from the topic's client list
		var next *list.Element
		for e := clients.Front(); e != nil; e = next {
			next = e.Next()
			if e.Value.(chan Event) == ch {
				clients.Remove(e)
				break
			}
		}

		// If the topic doesn't have any more clients, kill it
		if clients.Len() == 0 {
			delete(b.topics, topic)
		}
	}
}

func (b *Broker) Dispatch(topic string, event Event) {
	b.RLock()
	defer b.RUnlock()
	if clients, exists := b.topics[topic]; exists {
		var next *list.Element
		for e := clients.Front(); e != nil; e = next {
			next = e.Next()
			ch := e.Value.(chan Event)
			ch <- event
		}
	}
}

