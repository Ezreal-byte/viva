package linux

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

const (
	ximConnect = iota + 1
	ximConnectReply
	ximDisconnect
	ximDisconnectReply
	ximError
	ximOpen
	ximOpenReply
	ximEncodingNegotiation
	ximEncodingNegotiationReply
	ximCreateIC
	ximCreateICReply
	ximSetICFocus
	ximUnsetICFocus
	ximForwardEvent
	ximCommit
	ximPreeditStart
	ximPreeditDraw
	ximPreeditDone
)

const ximHeaderSize = 4

type ximTransport struct {
	api            *xcbAPI
	connection     uintptr
	clientWindow   uint32
	serverWindow   uint32
	property       uint32
	imid, icid     uint16
	focused        bool
	reading        []byte
	receivedKinds  []uint16
	commitValues   []string
	preeditStrings []string
}

func (transport *ximTransport) resolveAtoms() error {
	atom, err := transport.api.atom(transport.connection, "_VIVA_XIM_DATA")
	if err != nil {
		return err
	}
	transport.property = atom
	return nil
}

func (transport *ximTransport) send(message []byte) error {
	if transport.serverWindow == 0 {
		return errors.New("linux: XIM server is not connected")
	}
	transport.api.changeProperty(transport.connection, xcbPropertyReplace, transport.serverWindow, transport.property, xcbAtomString, 8, uint32(len(message)), unsafe.Pointer(&message[0]))
	transport.api.sendEvent(transport.connection, 0, transport.serverWindow, 0, (*byte)(unsafe.Pointer(&xcbClientEvent{
		ResponseType: xcbClientMessage,
		Format:       32,
		Window:       transport.serverWindow,
		Type:         transport.property,
		Data:         [20]byte{byte(len(message))},
	})))
	transport.api.flush(transport.connection)
	return nil
}

func (transport *ximTransport) request(kind uint16, body []byte) error {
	message := make([]byte, ximHeaderSize+len(body))
	message[0], message[1] = byte(kind), 0
	putUint16(message[2:], uint16(len(body)))
	copy(message[ximHeaderSize:], body)
	return transport.send(message)
}

func (transport *ximTransport) handleClientMessage(message *xcbClientEvent) error {
	if transport.clientWindow == 0 || message.Window != transport.clientWindow || message.Type != transport.property {
		return nil
	}
	data, ok, err := transport.api.property(transport.connection, transport.clientWindow, transport.property)
	if err != nil {
		return err
	}
	if ok {
		transport.reading = append(transport.reading, data...)
	} else if message.Format == 8 {
		transport.reading = append(transport.reading, message.Data[:20]...)
	}
	return transport.dispatch()
}

func (transport *ximTransport) dispatch() error {
	for len(transport.reading) >= ximHeaderSize {
		length := int(getUint16(transport.reading[2:]))
		if len(transport.reading) < ximHeaderSize+length {
			return nil
		}
		message := transport.reading[:ximHeaderSize+length]
		transport.reading = transport.reading[ximHeaderSize+length:]
		kind := uint16(message[0])
		transport.receivedKinds = append(transport.receivedKinds, kind)
		switch kind {
		case ximError:
			return fmt.Errorf("linux: XIM error code %d", getUint16(message[ximHeaderSize+4:]))
		case ximOpenReply:
			if len(message) < ximHeaderSize+2 {
				return errors.New("linux: short XIM open reply")
			}
			transport.imid = getUint16(message[ximHeaderSize:])
		case ximCreateICReply:
			if len(message) < ximHeaderSize+4 {
				return errors.New("linux: short XIM context reply")
			}
			transport.icid = getUint16(message[ximHeaderSize+2:])
		case ximCommit:
			if len(message) < ximHeaderSize+4 {
				return errors.New("linux: short XIM commit")
			}
			transport.commitValues = append(transport.commitValues, string(message[ximHeaderSize+4:ximHeaderSize+4+int(getUint16(message[ximHeaderSize+2:]))]))
		case ximPreeditDraw:
			if len(message) < ximHeaderSize+16 {
				return errors.New("linux: short XIM preedit")
			}
			status := getUint32(message[ximHeaderSize+12:])
			if status&1 == 0 {
				valueLength := int(getUint16(message[ximHeaderSize+16:]))
				if len(message) < ximHeaderSize+18+valueLength {
					return errors.New("linux: short XIM preedit text")
				}
				transport.preeditStrings = append(transport.preeditStrings, string(message[ximHeaderSize+18:ximHeaderSize+18+valueLength]))
			}
		}
	}
	return nil
}

func (transport *ximTransport) connectBody() []byte {
	body := []byte{'l', 0, 1, 0, 0}
	return body
}

func (transport *ximTransport) openBody(locale string) []byte {
	body := []byte{byte(len(locale))}
	body = append(body, locale...)
	if remainder := len(body) % 4; remainder != 0 {
		body = append(body, make([]byte, 4-remainder)...)
	}
	return body
}

func (transport *ximTransport) start(locale string) error {
	if err := transport.request(ximConnect, transport.connectBody()); err != nil {
		return err
	}
	return transport.request(ximOpen, transport.openBody(locale))
}

func (transport *ximTransport) createContext() error {
	body := make([]byte, 8)
	putUint16(body[0:], transport.imid)
	putUint32(body[4:], transport.clientWindow)
	return transport.request(ximCreateIC, body)
}

func (transport *ximTransport) focus(focused bool) error {
	if transport.focused == focused {
		return nil
	}
	transport.focused = focused
	kind := uint16(ximUnsetICFocus)
	if focused {
		kind = ximSetICFocus
	}
	body := make([]byte, 4)
	putUint16(body[0:], transport.imid)
	putUint16(body[2:], transport.icid)
	return transport.request(kind, body)
}

func (transport *ximTransport) forward(event *xcbInputEvent) error {
	body := make([]byte, 40)
	putUint16(body[0:], transport.imid)
	putUint16(body[2:], transport.icid)
	body[4] = byte(xcbKeyPress)
	body[8] = event.Detail
	putUint16(body[10:], event.State)
	putUint32(body[12:], event.Time)
	putUint32(body[16:], event.Root)
	putUint32(body[20:], event.Event)
	putUint32(body[24:], event.Child)
	putUint16(body[28:], uint16(event.EventX))
	putUint16(body[30:], uint16(event.EventY))
	putUint16(body[32:], uint16(event.RootX))
	putUint16(body[34:], uint16(event.RootY))
	body[36] = event.SameScreen
	return transport.request(ximForwardEvent, body)
}

func openXIMTransport(api *xcbAPI, connection uintptr, root uint32) (*ximTransport, error) {
	transport := &ximTransport{api: api, connection: connection, clientWindow: api.generateID(connection)}
	atom, err := api.atom(connection, "XIM_SERVERS")
	if err != nil {
		return nil, err
	}
	servers, ok, err := api.property(connection, root, atom)
	if err != nil || !ok {
		return nil, err
	}
	if len(servers)%4 != 0 || len(servers) == 0 {
		return nil, errors.New("linux: XIM server list is malformed")
	}
	for offset := 0; offset < len(servers); offset += 4 {
		serverAtom := getUint32(servers[offset:])
		reply := api.selectionOwnerReply(connection, api.getSelectionOwner(connection, serverAtom), nil)
		if reply == nil || reply.Owner == 0 {
			continue
		}
		api.createWindow(connection, xcbCopyFromParent, transport.clientWindow, root, 0, 0, 1, 1, 0, xcbWindowClassInput, 0, 0, nil)
		transport.serverWindow = reply.Owner
		if err := transport.resolveAtoms(); err != nil {
			api.destroyWindow(connection, transport.clientWindow)
			return nil, err
		}
		eventMask := []uint32{xcbEventPropertyChange}
		api.changeWindow(connection, transport.clientWindow, xcbCWEventMask, &eventMask[0])
		if err := transport.start(ximLocale()); err != nil {
			api.destroyWindow(connection, transport.clientWindow)
			return nil, err
		}
		return transport, nil
	}
	return nil, nil
}

func ximLocale() string {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" && value != "C" && value != "POSIX" {
			return value
		}
	}
	return "zh_CN.UTF-8"
}

func putUint16(data []byte, value uint16) {
	data[0], data[1] = byte(value), byte(value>>8)
}

func putUint32(data []byte, value uint32) {
	data[0], data[1], data[2], data[3] = byte(value), byte(value>>8), byte(value>>16), byte(value>>24)
}

func getUint16(data []byte) uint16 {
	return uint16(data[0]) | uint16(data[1])<<8
}

func getUint32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}
