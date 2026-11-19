//go:build unix

package ibus

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/vivaui/viva/internal/dbus"
)

const (
	fcitx4Method  = "org.fcitx.Fcitx.InputMethod"
	fcitx4Context = "org.fcitx.Fcitx.InputContext"

	fcitx4Preedit          = 1 << 1
	fcitx4FormattedPreedit = 1 << 4
	fcitx4Surrounding      = 1 << 6
)

var errFcitx4PreeditMode = errors.New("fcitx: no client commit preedit protocol")

// DialDesktop reaches the input method configured in the X11 session. Fcitx 4
// (including its Sogou engine) speaks its own protocol on the session bus.
func DialDesktop() (*Bus, error) {
	if os.Getenv("XMODIFIERS") == "@im=fcitx" || os.Getenv("GTK_IM_MODULE") == "fcitx" || os.Getenv("QT_IM_MODULE") == "fcitx" {
		if bus, err := dialFcitx4(); err == nil {
			return bus, nil
		}
		return Dial()
	}
	if bus, err := Dial(); err == nil {
		return bus, nil
	}
	return dialFcitx4()
}

func dialFcitx4() (*Bus, error) {
	conn, err := dbus.DialSession()
	if err != nil {
		return nil, err
	}
	_, display := displayParts(os.Getenv("DISPLAY"))
	var body dbus.Encoder
	body.String("org.fcitx.Fcitx-" + display)
	reply, err := conn.Call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "GetNameOwner", "s", body.Bytes())
	if err == nil && reply.Signature == "s" {
		args := reply.Args()
		if owner := args.String(); strings.HasPrefix(owner, ":") && args.Err() == nil {
			return newInputBus(conn, owner, fcitx4Context), nil
		}
	}
	conn.Close()
	if err != nil {
		return nil, err
	}
	return nil, errors.New("fcitx: no input method owner")
}

func (bus *Bus) newFcitx4Context(name string, handler Handler) (*Context, error) {
	var body dbus.Encoder
	body.String(name)
	body.Int32(int32(os.Getpid()))
	reply, err := bus.conn.Call(bus.destination, "/inputmethod", fcitx4Method, "CreateICv3", "si", body.Bytes())
	if err == nil {
		args := reply.Args()
		id := args.Int32()
		if reply.Sender == bus.destination && reply.Signature == "ibuuuu" && len(reply.Body) == 24 && args.Err() == nil && id >= 0 {
			return bus.registerContext("/inputcontext_"+strconv.Itoa(int(id)), handler,
				"SetCapacity", fcitx4Preedit|fcitx4FormattedPreedit|fcitx4Surrounding)
		}
		err = errors.New("fcitx: invalid input context")
	}
	bus.fail(err)
	return nil, err
}

func (context *Context) fcitx4Signal(message *dbus.Message) {
	args := message.Args()
	switch message.Member {
	case "CommitString":
		value := args.String()
		if message.Signature == "s" && args.Err() == nil && context.handler.Commit != nil {
			context.handler.Commit(value)
		}
	case "UpdateFormattedPreedit":
		if message.Signature != "a(si)i" || context.handler.Preedit == nil {
			return
		}
		var value strings.Builder
		end := args.ArrayLength(8)
		for args.Err() == nil && args.Pos() < end {
			args.Align(8)
			value.WriteString(args.String())
			_ = args.Int32() // Formatting flags. Viva draws its own marked text.
		}
		if args.Pos() != end {
			return
		}
		cursor := int(args.Int32())
		if args.Err() == nil && args.Pos() == len(message.Body) {
			text := value.String()
			cursor = min(max(0, cursor), len(text)) // Fcitx uses a UTF-8 byte offset.
			for cursor > 0 && cursor < len(text) && !utf8.RuneStart(text[cursor]) {
				cursor--
			}
			context.handler.Preedit(PreeditEvent{Text: text, Cursor: utf8.RuneCountInString(text[:cursor]), Visible: text != ""})
		}
	case "DeleteSurroundingText":
		offset, count := args.Int32(), args.Uint32()
		if message.Signature == "iu" && args.Err() == nil && context.handler.DeleteSurrounding != nil {
			context.handler.DeleteSurrounding(int(offset), int(count))
		}
	case "ForwardKey":
		symbol, state, release := args.Uint32(), args.Uint32(), args.Int32()
		if message.Signature == "uui" && args.Err() == nil && release == 0 && context.handler.Forward != nil {
			context.handler.Forward(symbol, 0, state)
		}
	}
}
