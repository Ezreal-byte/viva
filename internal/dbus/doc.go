// Package dbus is the smallest D-Bus client a status icon needs: connect to the
// session bus, authenticate as the current user, call methods, answer the calls
// an object receives, and emit signals. Values are encoded by typed methods, with
// no reflection, and only little-endian messages without file descriptors are
// understood, which is what every desktop bus speaks.
//
// A Conn is not safe for concurrent use; its owner drives it from one thread,
// waiting on FD for the bus to have something to read.
package dbus
