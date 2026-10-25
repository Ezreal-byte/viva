//go:build unix

package dbus

import (
	"testing"
	"time"
)

func TestWaitReturnsNothingWhenTheTimeRunsOut(t *testing.T) {
	_, conn := newFakeBus(t)
	start := time.Now()
	messages, err := conn.Wait(40 * time.Millisecond)
	if err != nil || len(messages) != 0 {
		t.Fatalf("Wait = %v, %v; want nothing and no error", messages, err)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond || elapsed > 2*time.Second {
		t.Fatalf("Wait took %v", elapsed)
	}
}

func TestWaitReturnsASignalThatArrivesWhileItBlocks(t *testing.T) {
	bus, conn := newFakeBus(t)
	go func() {
		time.Sleep(30 * time.Millisecond)
		bus.write((&Message{Type: TypeSignal, Serial: 3, Path: "/p", Interface: "org.p", Member: "Late"}).encode())
	}()
	messages, err := conn.Wait(5 * time.Second)
	if err != nil || len(messages) != 1 || messages[0].Member != "Late" {
		t.Fatalf("Wait = %v, %v", messages, err)
	}
	// The connection still answers calls with its usual timeout afterwards.
	if _, err := conn.Receive(); err != nil {
		t.Fatal(err)
	}
}
