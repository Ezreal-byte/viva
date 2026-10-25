//go:build darwin || windows || linux

package headless

import (
	"testing"

	"github.com/vivaui/viva/app"
)

func TestOpenURLReachesTheApplicationsHandler(t *testing.T) {
	host := newApplicationSession(Options{Scale: 1})
	t.Cleanup(host.closeAll)
	app.UseApplicationRunner(func(start func(app.Host) error) error { return start(host) })
	t.Cleanup(func() { app.UseApplicationRunner(nil) })
	var got []string
	err := app.RunApplication(func(application *app.Application) error {
		application.OnOpenURL(func(url string) { got = append(got, url) })
		_, err := application.OpenWindow(app.Config{Title: "main", Width: 100, Height: 80}, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply, _ := executeApplication(host, "openurl ssh://user@host:2222/"); reply != "ok\n" {
		t.Fatalf("reply %q", reply)
	}
	if len(got) != 1 || got[0] != "ssh://user@host:2222/" {
		t.Fatalf("handler got %q", got)
	}
	if reply, _ := executeApplication(host, "openurl"); reply[:5] != "error" {
		t.Fatalf("reply %q", reply)
	}
}

func TestOpenURLWithoutAHandlerIsAnError(t *testing.T) {
	host := newApplicationSession(Options{Scale: 1})
	t.Cleanup(host.closeAll)
	if reply, _ := executeApplication(host, "openurl ssh://h/"); reply[:5] != "error" {
		t.Fatalf("reply %q", reply)
	}
}
