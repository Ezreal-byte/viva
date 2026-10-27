//go:build darwin || windows || linux

package app

import "testing"

type quitRequestHost struct {
	fakeHost
	handler func()
}

func (host *quitRequestHost) SetQuitRequestHandler(handler func()) { host.handler = handler }

func TestSystemQuitCanWaitForDocumentConfirmation(t *testing.T) {
	host := &quitRequestHost{}
	application := &Application{host: host}
	var allow func()
	application.OnQuitRequest(func(next func()) { allow = next })
	host.handler()
	if host.quits != 0 || allow == nil {
		t.Fatal("system quit bypassed confirmation")
	}
	allow()
	if host.quits != 1 {
		t.Fatal("confirmed quit did not reach host")
	}
	application.OnQuitRequest(nil)
	if host.handler != nil {
		t.Fatal("quit callback retained")
	}
}
