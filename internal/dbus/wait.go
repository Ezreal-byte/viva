//go:build unix

package dbus

import (
	"errors"
	"syscall"
	"time"
)

// Wait blocks until a message has arrived or timeout has passed, and returns
// what has arrived: nothing, with no error, when the time ran out. It is for a
// caller that waits on one answer the user is giving (a file dialog) and has no
// event loop to wait in; a loop polls FD and calls Receive instead.
func (c *Conn) Wait(timeout time.Duration) ([]*Message, error) {
	deadline := time.Now().Add(timeout)
	defer c.setReadTimeout(callTimeout)
	for len(c.pending) == 0 {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, nil
		}
		// A zero timeout would mean no timeout at all.
		c.setReadTimeout(max(remaining, time.Millisecond))
		if err := c.fill(true); err != nil && !errors.Is(err, errTimedOut) {
			return nil, err
		}
	}
	messages := c.pending
	c.pending = nil
	return messages, nil
}

func (c *Conn) setReadTimeout(timeout time.Duration) {
	value := syscall.NsecToTimeval(timeout.Nanoseconds())
	_ = syscall.SetsockoptTimeval(c.fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &value)
}
