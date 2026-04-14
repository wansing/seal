package seal

import (
	"testing"
)

func TestBroker(t *testing.T) {
	b := &Broker[string]{}
	b.Publish("hello-world")

	var got string
	b.Subscribe(func(data string) {
		got = data
	})

	if want := "hello-world"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	b.Publish("goodbye")

	if want := "goodbye"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
