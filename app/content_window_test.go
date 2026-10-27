//go:build darwin || windows || linux

package app

import (
	"sync"
	"testing"
)

func TestContentWindowDeliversUpdatesAndHonorsClosePolicy(t *testing.T) {
	host := &fakeHost{}
	useHost(t, host)
	err := RunApplication(func(application *Application) error {
		var allow func()
		wakes, closed := 0, 0
		content, err := application.OpenContentWindow(Config{Title: "document", OnWake: func() { wakes++ }, OnClosed: func() { closed++ }, OnCloseRequest: func(next func()) { allow = next }}, nil)
		if err != nil {
			return err
		}
		second, err := application.OpenContentWindow(Config{Title: "inspector"}, nil)
		if err != nil {
			return err
		}
		if host.windows[0].config.Width != 960 || host.windows[0].config.Height != 720 {
			t.Fatal("missing default size")
		}
		var wg sync.WaitGroup
		count := 0
		for range 32 {
			wg.Go(func() {
				if !content.Post(func() { count++ }) {
					t.Error("rejected live update")
				}
			})
		}
		wg.Wait()
		if count != 0 {
			t.Fatal("background worker changed UI state")
		}
		host.windows[0].config.OnWake()
		if count != 32 || wakes != 1 {
			t.Fatal("updates were not delivered before wake", count, wakes)
		}
		content.RequestClose()
		if content.Closed() || allow == nil {
			t.Fatal("close policy was bypassed")
		}
		allow()
		// The fake host records native destruction separately from application bookkeeping.
		host.windows[0].config.OnClosed()
		if !content.Closed() || closed != 1 || second.Closed() {
			t.Fatal("independent lifetime lost")
		}
		if content.Post(func() { t.Error("closed update ran") }) {
			t.Fatal("accepted closed update")
		}
		host.windows[0].config.OnWake()
		if wakes != 1 {
			t.Fatal("closed window woke")
		}
		second.Close()
		host.windows[1].config.OnClosed()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestContentWindowRejectedOpenAndPendingUpdateTeardown(t *testing.T) {
	host := &fakeHost{reject: "rejected"}
	useHost(t, host)
	if err := RunApplication(func(application *Application) error {
		if content, err := application.OpenContentWindow(Config{Title: "rejected"}, nil); err == nil || content != nil {
			t.Fatal("rejected native open retained")
		}
		content, err := application.OpenContentWindow(Config{}, nil)
		if err != nil {
			return err
		}
		content.Post(func() { t.Error("update outlived native content") })
		host.windows[0].config.OnClosed()
		content.Close()
		host.windows[0].config.OnWake()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
