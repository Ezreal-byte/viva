package linux

import (
	"os"
	"unsafe"
)

// xcbAtomWMClientMachine is the predefined atom of WM_CLIENT_MACHINE.
const xcbAtomWMClientMachine = 36

// setIdentity names the window and its process to the window manager: the title
// in both forms (_NET_WM_NAME is UTF-8, WM_NAME can only carry Latin-1 and is
// what older managers read), and the process id with the host it runs on, which
// is how a manager offers to end a window that stopped answering its ping.
func (win *window) setIdentity(title string) {
	api, connection := win.api, win.connection
	if title != "" {
		legacy := latin1(title)
		api.changeProperty(connection, xcbPropertyReplace, win.id, xcbAtomWMName, xcbAtomString, 8, uint32(len(legacy)), unsafe.Pointer(&legacy[0]))
		utf8String, err := api.atom(connection, "UTF8_STRING")
		name, nameErr := api.atom(connection, "_NET_WM_NAME")
		if err == nil && nameErr == nil {
			encoded := []byte(title)
			api.changeProperty(connection, xcbPropertyReplace, win.id, name, utf8String, 8, uint32(len(encoded)), unsafe.Pointer(&encoded[0]))
		}
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		machine := []byte(host)
		api.changeProperty(connection, xcbPropertyReplace, win.id, xcbAtomWMClientMachine, xcbAtomString, 8, uint32(len(machine)), unsafe.Pointer(&machine[0]))
		if pidAtom, err := api.atom(connection, "_NET_WM_PID"); err == nil {
			pid := uint32(os.Getpid())
			api.changeProperty(connection, xcbPropertyReplace, win.id, pidAtom, xcbAtomCardinal, 32, 1, unsafe.Pointer(&pid))
		}
	}
}
