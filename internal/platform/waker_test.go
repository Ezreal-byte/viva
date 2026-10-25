package platform

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWakerIsNoOpUntilBoundAndAfterUnbind(t *testing.T) {
	var waker Waker
	waker.Wake()

	var wakes atomic.Int32
	waker.Bind(func() { wakes.Add(1) })
	waker.Wake()
	waker.Wake()
	if wakes.Load() != 2 {
		t.Fatalf("wakes = %d, want 2", wakes.Load())
	}

	waker.Unbind()
	waker.Wake()
	if wakes.Load() != 2 {
		t.Fatalf("wake after Unbind reached the adapter: %d", wakes.Load())
	}
}

func TestWakerUnbindWaitsForInFlightWake(t *testing.T) {
	var waker Waker
	entered, release := make(chan struct{}), make(chan struct{})
	waker.Bind(func() {
		close(entered)
		<-release
	})
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		waker.Wake()
	}()
	<-entered

	var unbindReturned atomic.Bool
	unbound := make(chan struct{})
	go func() {
		waker.Unbind()
		unbindReturned.Store(true)
		close(unbound)
	}()
	runtime.Gosched()
	if unbindReturned.Load() {
		t.Fatal("Unbind returned while the adapter was still posting")
	}
	close(release)
	<-unbound
	group.Wait()
}
