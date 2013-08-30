package main

import (
	"testing"
	"time"
)

func Test_Subscribe(t *testing.T) {
	b := NewBroker()
	b.Subscribe("test")

	clients, exists := b.topics["test"]
	if !exists {
		t.Error("Subscribing didn't create topic")
	}

	if clients.Len() != 1 {
		t.Error("Subscribing didn't add client to topic")
	}
}

func Test_Unsubscribe(t *testing.T) {
	b := NewBroker()

	ch1 := b.Subscribe("test")
	ch2 := b.Subscribe("test")

	b.Unsubscribe(ch1, "test")

	clients, _ := b.topics["test"]
	if clients.Len() != 1 {
		t.Error("Unsubscribing didn't remove client from topic")
	}

	b.Unsubscribe(ch2, "test")

	if _, exists := b.topics["test"]; exists {
		t.Error("Unsubscribing didn't remove topic")
	}
}

func Test_Dispatch(t *testing.T) {
	b := NewBroker()

	ch1 := b.Subscribe("a")
	ch2 := b.Subscribe("b")

	done := make(chan bool)

	// Surely there's another way
	go func() {
		select {
		case <-ch1:
		case <-time.After(100):
			t.Error("ch1 didn't receive event")
		}
		done <- true
	}()

	// Don't call me Shirley
	go func() {
		select {
		case <-ch2:
			t.Error("ch2 received a message")
		case <-time.After(100):
		}
		done <- true
	}()

	b.Dispatch("a", Event{"hi"})

	<- done
	<- done
}

