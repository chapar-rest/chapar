package events

import (
	"sync"
	"testing"
	"time"
)

func TestBrokerSubscribePublish(t *testing.T) {
	b := NewBroker[int](1)
	sub := b.Subscribe()
	defer sub.Unsubscribe()

	b.Publish(42)

	select {
	case got := <-sub.C:
		if got != 42 {
			t.Fatalf("expected 42, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published message")
	}
}

func TestBrokerFanout(t *testing.T) {
	b := NewBroker[string](1)
	sub1 := b.Subscribe()
	sub2 := b.Subscribe()
	defer sub1.Unsubscribe()
	defer sub2.Unsubscribe()

	b.Publish("hello")

	assertReceive(t, sub1.C, "hello")
	assertReceive(t, sub2.C, "hello")
}

func TestBrokersDifferentTypesDoNotInterfere(t *testing.T) {
	intBroker := NewBroker[int](1)
	stringBroker := NewBroker[string](1)

	intSub := intBroker.Subscribe()
	stringSub := stringBroker.Subscribe()
	defer intSub.Unsubscribe()
	defer stringSub.Unsubscribe()

	intBroker.Publish(7)
	stringBroker.Publish("salam")

	assertReceive(t, intSub.C, 7)
	assertReceive(t, stringSub.C, "salam")
}

func TestBrokerUnsubscribeStopsDeliveryAndClosesChannel(t *testing.T) {
	b := NewBroker[int](1)
	sub := b.Subscribe()

	sub.Unsubscribe()

	select {
	case _, ok := <-sub.C:
		if ok {
			t.Fatal("expected channel to be closed after unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for closed subscription channel")
	}

	b.Publish(1)

	select {
	case _, ok := <-sub.C:
		if ok {
			t.Fatal("expected closed channel after publish on unsubscribed subscription")
		}
	default:
	}
}

func TestBrokerCloseClosesAllSubscriberChannels(t *testing.T) {
	b := NewBroker[int](1)
	sub1 := b.Subscribe()
	sub2 := b.Subscribe()

	b.Close()

	assertClosed(t, sub1.C)
	assertClosed(t, sub2.C)
}

func TestBrokerConcurrentPublish(t *testing.T) {
	const (
		publishers = 8
		perPub     = 50
		total      = publishers * perPub
	)

	b := NewBroker[int](total)
	sub1 := b.Subscribe()
	sub2 := b.Subscribe()
	defer sub1.Unsubscribe()
	defer sub2.Unsubscribe()

	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < perPub; i++ {
				b.Publish(offset*perPub + i)
			}
		}(p)
	}
	wg.Wait()

	received1 := drainCount(t, sub1.C, total)
	received2 := drainCount(t, sub2.C, total)

	if received1 != total {
		t.Fatalf("subscriber 1: expected %d messages, got %d", total, received1)
	}
	if received2 != total {
		t.Fatalf("subscriber 2: expected %d messages, got %d", total, received2)
	}
}

func assertReceive[T comparable](t *testing.T, ch <-chan T, want T) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("expected %v, got %v", want, got)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %v", want)
	}
}

func assertClosed[T any](t *testing.T, ch <-chan T) {
	t.Helper()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func drainCount[T any](t *testing.T, ch <-chan T, expected int) int {
	t.Helper()

	count := 0
	timeout := time.After(2 * time.Second)
	for count < expected {
		select {
		case <-ch:
			count++
		case <-timeout:
			t.Fatalf("timed out while draining messages: got %d, expected %d", count, expected)
		}
	}
	return count
}
