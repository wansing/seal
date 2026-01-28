package seal

import (
	"testing"
)

func TestBroker(t *testing.T) {
	var got string
	b := NewBroker()
	b.Subscribe("/foo", func(data any) {
		got = data.(string)
	})

	b.Publish("/foo", "hello-world")

	// before ready
	if want := ""; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	b.Ready()

	// after ready
	if want := "hello-world"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	b.Publish("/foo", "goodbye")

	// updated
	if want := "goodbye"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
