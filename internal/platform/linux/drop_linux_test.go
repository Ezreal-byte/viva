//go:build linux && !android

package linux

import (
	"encoding/binary"
	"os"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/vivaui/viva/internal/platform"
)

func TestURIListsBecomeLocalPaths(t *testing.T) {
	got := parseURIList("# a comment\r\nfile:///tmp/a%20b.txt\r\nfile://localhost/tmp/dir\nhttp://example.com/x\nfile://other-host/tmp/y\nfile:relative\n\nfile:///tmp/caf%C3%A9\n\x00")
	want := []string{"/tmp/a b.txt", "/tmp/dir", "/tmp/café"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %q, want %q", got, want)
	}
	if parseURIList("") != nil {
		t.Fatal("an empty list has paths")
	}
}

// source is another X client playing the program a drag starts from.
type source struct {
	t           *testing.T
	api         *xcbAPI
	connection  uintptr
	screen      *xcbScreen
	id          uint32
	beforeReply func()
}

func (s *source) message(target, kind uint32, longs ...uint32) {
	event := xcbClientEvent{ResponseType: xcbClientMessage, Format: 32, Window: target, Type: kind}
	for index, value := range longs {
		binary.LittleEndian.PutUint32(event.Data[index*4:], value)
	}
	s.api.sendEvent(s.connection, 0, target, 0, (*byte)(unsafe.Pointer(&event)))
	s.api.flush(s.connection)
}

// next waits for an event, answering the target's request for the dragged data.
func (s *source) next(selectionAtom, listAtom uint32, list string) (unsafe.Pointer, bool) {
	deadline := time.Now().Add(5 * time.Second)
	file := pollFD{FD: s.api.fileDescriptor(s.connection), Events: xcbPollInput}
	for time.Now().Before(deadline) {
		if event := s.api.pollForEvent(s.connection); event != nil {
			if (*xcbGenericEvent)(event).ResponseType&0x7f == xcbSelectionRequest {
				request := (*xcbSelectionRequestEvent)(event)
				if s.beforeReply != nil {
					s.beforeReply()
				}
				data := []byte(list)
				s.api.changeProperty(s.connection, xcbPropertyReplace, request.Requestor, request.Property, listAtom, 8, uint32(len(data)), unsafe.Pointer(&data[0]))
				notify := xcbSelectionNotifyWire{xcbSelectionNotifyEvent: xcbSelectionNotifyEvent{
					ResponseType: xcbSelectionNotify, Requestor: request.Requestor, Selection: request.Selection,
					Target: request.Target, Property: request.Property,
				}}
				s.api.sendEvent(s.connection, 0, request.Requestor, 0, (*byte)(unsafe.Pointer(&notify)))
				s.api.flush(s.connection)
				s.api.free(event)
				continue
			}
			return event, true
		}
		s.api.poll(&file, 1, 20)
	}
	return nil, false
}

// reply waits for the target's XdndStatus or XdndFinished and returns its flags.
func (s *source) reply(kind, selectionAtom, listAtom uint32, list string) (flags uint32, found bool) {
	for {
		event, ok := s.next(selectionAtom, listAtom, list)
		if !ok {
			return 0, false
		}
		message := (*xcbClientEvent)(event)
		isReply := (*xcbGenericEvent)(event).ResponseType&0x7f == xcbClientMessage && message.Type == kind
		flags = binary.LittleEndian.Uint32(message.Data[4:])
		s.api.free(event)
		if isReply {
			return flags, true
		}
	}
}

// TestAFileDragFromAnotherClientIsAcceptedAndDropped runs the whole protocol
// against a real window on a real X server: the source enters, moves, and drops
// a list of files, and the application hears the hover and the paths.
func TestAFileDragFromAnotherClientIsAcceptedAndDropped(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no X server")
	}
	events := make(chan platform.FileDropEvent, 8)
	redrawn := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		dropped := false
		done <- Run(platform.Config{
			Title: "drop", Width: 300, Height: 200,
			Waker: &platform.Waker{},
			OnRedraw: func() (platform.Frame, error) {
				if dropped {
					select {
					case redrawn <- struct{}{}:
					default:
					}
				}
				return platform.Frame{}, nil
			},
			OnFileDrop: func(event platform.FileDropEvent) (platform.Frame, bool, error) {
				dropped = event.Phase == platform.FileDropDrop
				events <- event
				return platform.Frame{}, event.Phase != platform.FileDropLeave, nil
			},
		})
	}()
	var target *window
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		select {
		case err := <-done:
			t.Skipf("no window could be opened here: %v", err)
		default:
		}
		if target = activeWindow.Load(); target != nil {
			break
		}
	}
	if target == nil {
		t.Fatal("the window never showed")
	}
	api, connection, screen, err := openSystemConnection()
	if err != nil {
		t.Fatal(err)
	}
	defer api.disconnect(connection)
	src := &source{t: t, api: api, connection: connection, screen: screen, id: api.generateID(connection)}
	values := [1]uint32{0}
	api.createWindow(connection, xcbCopyFromParent, src.id, screen.Root, 0, 0, 10, 10, 0, xcbWindowClassInput, screen.RootVisual, xcbCWEventMask, &values[0])
	atom := func(name string) uint32 {
		value, err := api.atom(connection, name)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	enter, position, status, drop, finished := atom("XdndEnter"), atom("XdndPosition"), atom("XdndStatus"), atom("XdndDrop"), atom("XdndFinished")
	selection, uriList, copyAction, aware := atom("XdndSelection"), atom("text/uri-list"), atom("XdndActionCopy"), atom("XdndAware")
	api.setSelectionOwner(connection, src.id, selection, 0)
	api.flush(connection)

	// The window said it takes drags, and at which version.
	cookie := api.getProperty(connection, 0, target.id, aware, xcbAtomAtom, 0, 1)
	var nativeError unsafe.Pointer
	reply := api.getPropertyReply(connection, cookie, &nativeError)
	if reply == nil || reply.Format != 32 || *(*uint32)(api.getPropertyValue(reply)) != xdndVersion {
		t.Fatalf("XdndAware not set: %+v", reply)
	}
	api.free(unsafe.Pointer(reply))

	list := "file:///tmp/a%20b.txt\r\nfile:///tmp/dir\r\n"
	src.message(target.id, enter, src.id, xdndVersion<<24, uriList)
	rootCookie := api.translateCoordinates(connection, target.id, screen.Root, 60, 30)
	rootReply := api.translateCoordinatesReply(connection, rootCookie, &nativeError)
	if rootReply == nil {
		t.Fatal("cannot translate coordinates")
	}
	rootX, rootY := uint32(uint16(rootReply.DstX)), uint32(uint16(rootReply.DstY))
	api.free(unsafe.Pointer(rootReply))
	src.message(target.id, position, src.id, 0, rootX<<16|rootY, 0, copyAction)
	flags, ok := src.reply(status, selection, uriList, list)
	if !ok || flags&1 == 0 {
		t.Fatalf("XdndStatus: found %v flags %d: the point should take files", ok, flags)
	}
	hover := <-events
	if hover.Phase != platform.FileDropOver || hover.X < 59 || hover.X > 61 || hover.Y < 29 || hover.Y > 31 || len(hover.Paths) != 0 {
		t.Fatalf("hover = %+v", hover)
	}

	src.beforeReply = func() { src.message(target.id, target.redrawAtom) }
	src.message(target.id, drop, src.id, 0, 0)
	flags, ok = src.reply(finished, selection, uriList, list)
	if !ok || flags&1 == 0 {
		t.Fatalf("XdndFinished: found %v flags %d", ok, flags)
	}
	dropped := <-events
	if dropped.Phase != platform.FileDropDrop || !reflect.DeepEqual(dropped.Paths, []string{"/tmp/a b.txt", "/tmp/dir"}) {
		t.Fatalf("drop = %+v", dropped)
	}
	select {
	case <-redrawn:
	case <-time.After(2 * time.Second):
		t.Error("redraw sent during the drop selection transfer was lost")
	}

	// A drag that leaves ends the hover.
	src.message(target.id, enter, src.id, xdndVersion<<24, uriList)
	src.message(target.id, atom("XdndLeave"), src.id)
	if leave := <-events; leave.Phase != platform.FileDropLeave {
		t.Fatalf("leave = %+v", leave)
	}

	target.postClientMessage(target.closeAtom, 0)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the window did not close")
	}
}

func TestDropSelectionFallbackToWindowAndPreservesQueuedEvents(t *testing.T) {
	const (
		sourceWin     uint32 = 10
		targetWin     uint32 = 20
		sharedReqWin  uint32 = 30
		dropAtom      uint32 = 40
		finishedAtom  uint32 = 41
		selectionAtom uint32 = 42
		actionCopy    uint32 = 43
		uriListAtom   uint32 = 44
		dropProperty  uint32 = 45
		dropTimestamp uint32 = 777
	)

	selections.mu.Lock()
	prevAPI, prevConn, prevReq := selections.api, selections.connection, selections.requestor
	selections.mu.Unlock()
	t.Cleanup(func() {
		selections.mu.Lock()
		selections.api, selections.connection, selections.requestor = prevAPI, prevConn, prevReq
		selections.mu.Unlock()
	})

	refusal := xcbSelectionNotifyEvent{
		ResponseType: xcbSelectionNotify,
		Requestor:    sharedReqWin,
		Selection:    selectionAtom,
		Target:       xcbAtomNone,
		Property:     xcbAtomNone,
	}
	selReplied := false
	selAPI := &xcbAPI{
		atoms:              map[string]uint32{"XdndSelection": selectionAtom},
		connectionHasError: func(uintptr) int32 { return 0 },
		flush:              func(uintptr) int32 { return 1 },
		fileDescriptor:     func(uintptr) int32 { return 1 },
		free:               func(unsafe.Pointer) {},
		getSelectionOwner:  func(uintptr, uint32) xcbSelectionOwnerCookie { return xcbSelectionOwnerCookie{} },
		selectionOwnerReply: func(uintptr, xcbSelectionOwnerCookie, *unsafe.Pointer) *xcbSelectionOwnerReply {
			return &xcbSelectionOwnerReply{Owner: sourceWin}
		},
		convertSelection: func(uintptr, uint32, uint32, uint32, uint32, uint32) xcbVoidCookie {
			selReplied = false
			return xcbVoidCookie{}
		},
		pollForEvent: func(uintptr) unsafe.Pointer {
			if selReplied {
				return nil
			}
			selReplied = true
			return unsafe.Pointer(&refusal)
		},
		poll: func(*pollFD, uintptr, int32) int32 {
			t.Fatal("shared selection wait polled after Target=None, Property=None refusal")
			return 0
		},
	}
	selections.mu.Lock()
	selections.api, selections.connection, selections.requestor = selAPI, 1, sharedReqWin
	selections.mu.Unlock()

	payload := []byte("file:///tmp/from-qt5.png\r\n")
	queuedExpose := xcbExposeEvent{ResponseType: xcbExpose, Window: targetWin}
	winNotify := xcbSelectionNotifyEvent{
		ResponseType: xcbSelectionNotify,
		Requestor:    targetWin,
		Selection:    selectionAtom,
		Target:       uriListAtom,
		Property:     dropProperty,
	}
	winQueue := []unsafe.Pointer{unsafe.Pointer(&queuedExpose), unsafe.Pointer(&winNotify)}
	var sentReply xcbClientEvent
	var winReqTimestamp uint32
	winAPI := &xcbAPI{
		atoms:              map[string]uint32{"INCR": 99},
		connectionHasError: func(uintptr) int32 { return 0 },
		flush:              func(uintptr) int32 { return 1 },
		fileDescriptor:     func(uintptr) int32 { return 2 },
		free:               func(unsafe.Pointer) {},
		getSelectionOwner:  func(uintptr, uint32) xcbSelectionOwnerCookie { return xcbSelectionOwnerCookie{} },
		selectionOwnerReply: func(uintptr, xcbSelectionOwnerCookie, *unsafe.Pointer) *xcbSelectionOwnerReply {
			return &xcbSelectionOwnerReply{Owner: sourceWin}
		},
		convertSelection: func(_ uintptr, requestor, _, _, _, timestamp uint32) xcbVoidCookie {
			if requestor != targetWin {
				t.Fatalf("fallback convertSelection requestor = %d, want %d", requestor, targetWin)
			}
			winReqTimestamp = timestamp
			return xcbVoidCookie{}
		},
		pollForEvent: func(uintptr) unsafe.Pointer {
			if len(winQueue) == 0 {
				return nil
			}
			ev := winQueue[0]
			winQueue = winQueue[1:]
			return ev
		},
		poll: func(*pollFD, uintptr, int32) int32 {
			t.Fatal("window selection fallback polled unexpectedly")
			return 0
		},
		getProperty: func(uintptr, uint8, uint32, uint32, uint32, uint32, uint32) xcbPropertyCookie {
			return xcbPropertyCookie{}
		},
		getPropertyReply: func(uintptr, xcbPropertyCookie, *unsafe.Pointer) *xcbPropertyReply {
			return &xcbPropertyReply{Format: 8, Type: uriListAtom}
		},
		getPropertyLength: func(*xcbPropertyReply) int32 { return int32(len(payload)) },
		getPropertyValue:  func(*xcbPropertyReply) unsafe.Pointer { return unsafe.Pointer(&payload[0]) },
		sendEvent: func(_ uintptr, _ uint8, _ uint32, _ uint32, data *byte) xcbVoidCookie {
			copy(unsafe.Slice((*byte)(unsafe.Pointer(&sentReply)), unsafe.Sizeof(sentReply)), unsafe.Slice(data, unsafe.Sizeof(sentReply)))
			return xcbVoidCookie{}
		},
	}

	var gotEvents []platform.FileDropEvent
	state := &linuxWindowState{
		win: &window{
			api:        winAPI,
			connection: 2,
			id:         targetWin,
			xdnd: xdndAtoms{
				drop:         dropAtom,
				finished:     finishedAtom,
				selection:    selectionAtom,
				actionCopy:   actionCopy,
				uriList:      uriListAtom,
				dropProperty: dropProperty,
			},
		},
		config: platform.Config{
			OnFileDrop: func(event platform.FileDropEvent) (platform.Frame, bool, error) {
				gotEvents = append(gotEvents, event)
				return platform.Frame{}, true, nil
			},
		},
		router: eventRouter{present: func(platform.Frame) error { return nil }},
		drag:   xdndDrag{source: sourceWin, wantsURIs: true, x: 12, y: 34},
	}

	msg := xcbClientEvent{ResponseType: xcbClientMessage, Format: 32, Window: targetWin, Type: dropAtom}
	binary.LittleEndian.PutUint32(msg.Data[0:], sourceWin)
	binary.LittleEndian.PutUint32(msg.Data[8:], dropTimestamp)

	if err := state.handleDragMessage(&msg); err != nil {
		t.Fatalf("handleDragMessage returned error: %v", err)
	}
	if winReqTimestamp != dropTimestamp {
		t.Fatalf("fallback timestamp = %d, want %d", winReqTimestamp, dropTimestamp)
	}
	if len(gotEvents) != 1 || gotEvents[0].Phase != platform.FileDropDrop || !reflect.DeepEqual(gotEvents[0].Paths, []string{"/tmp/from-qt5.png"}) {
		t.Fatalf("events = %+v, want one drop with /tmp/from-qt5.png", gotEvents)
	}
	if sentReply.Type != finishedAtom || clientLong(&sentReply, 1) != 1 || clientLong(&sentReply, 2) != actionCopy {
		t.Fatalf("XdndFinished reply = type %d flags %d action %d, want accepted copy action in data[2]",
			sentReply.Type, clientLong(&sentReply, 1), clientLong(&sentReply, 2))
	}
	if replayed := state.takeHeld(); replayed != unsafe.Pointer(&queuedExpose) {
		t.Fatalf("replayed event = %p, want queued expose %p", replayed, &queuedExpose)
	}
}
