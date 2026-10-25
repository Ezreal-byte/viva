//go:build darwin

package darwin

import (
	"reflect"
	"testing"
)

func TestLinksThatArriveBeforeTheHandlerAreDeliveredWhenItIsSet(t *testing.T) {
	openURLs.Lock()
	openURLs.handler, openURLs.pending = nil, nil
	openURLs.Unlock()
	deliverOpenURL("ssh://early@host/")
	var got []string
	application := &nativeApplication{}
	application.SetOpenURLHandler(func(url string) { got = append(got, url) })
	deliverOpenURL("ssh://late@host:22/")
	if want := []string{"ssh://early@host/", "ssh://late@host:22/"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	application.SetOpenURLHandler(nil)
}
