//go:build darwin

package darwin

import "sync"

// Links the system opens the application with arrive on the event-loop thread,
// possibly before the application has said what to do with them (a launch by
// link): those wait here until it has.
var openURLs struct {
	sync.Mutex
	handler func(string)
	pending []string
}

const maxPendingURLs = 32

func deliverOpenURL(url string) {
	openURLs.Lock()
	handler := openURLs.handler
	if handler == nil {
		if len(openURLs.pending) < maxPendingURLs {
			openURLs.pending = append(openURLs.pending, url)
		}
		openURLs.Unlock()
		return
	}
	openURLs.Unlock()
	handler(url)
}

// SetOpenURLHandler sets who receives the links the system opens this
// application with, and delivers any that arrived first. It runs on the
// event-loop thread.
func (application *nativeApplication) SetOpenURLHandler(handler func(string)) {
	openURLs.Lock()
	openURLs.handler = handler
	pending := openURLs.pending
	openURLs.pending = nil
	openURLs.Unlock()
	if handler != nil {
		for _, url := range pending {
			handler(url)
		}
	}
}
