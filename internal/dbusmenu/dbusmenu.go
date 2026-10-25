// Package dbusmenu serves a menu over the com.canonical.dbusmenu interface, the
// way desktops that host StatusNotifier icons (KDE Plasma, the GNOME Shell
// AppIndicator extension, the panels of Xfce and others) ask for a menu to show
// beside an icon. The host draws the menu; this package answers what it asks.
//
// An Object has no connection of its own. The owner of the bus connection gives
// it each call made to the menu's path and sends back what it returns, so the
// package neither reads nor writes the bus and can be tested without one.
package dbusmenu

import (
	"strings"

	"github.com/vivaui/viva/internal/dbus"
	"github.com/vivaui/viva/internal/platform"
)

// Path is where an icon serves its menu, which the icon names in its Menu
// property.
const Path = "/MenuBar"

const (
	// Interface is the interface a host calls to read and use a menu.
	Interface  = "com.canonical.dbusmenu"
	properties = "org.freedesktop.DBus.Properties"
	version    = 3

	nodeType = "(ia{sv}av)"
)

// Object is one icon's menu. Its content is asked for from the application each
// time the host is about to show it, so it shows the state at that moment.
type Object struct {
	menu     func() []platform.MenuItem
	entries  []platform.MenuEntry
	revision uint32
	// base is added to an entry's number to make the id a host knows it by.
	// Every build takes ids no earlier build used, because a host that keeps
	// what it was told about an id (KDE's keeps that an entry is a separator)
	// would carry it over to whatever entry takes the id next.
	base int32
	// stale is true once a layout has been read: a host that asks for another
	// without first saying it is about to show the menu is reading it again, and
	// gets a new one.
	stale bool
}

// New returns the menu of an icon, built by menu on demand.
func New(menu func() []platform.MenuItem) *Object {
	return &Object{menu: menu, stale: true}
}

// Result is what to send back for a call: a reply of Signature and Body, or the
// error ErrName with ErrText when ErrName is set.
type Result struct {
	Signature string
	Body      []byte
	ErrName   string
	ErrText   string
}

func failure(name, text string) Result { return Result{ErrName: name, ErrText: text} }

func reply(signature string, e *dbus.Encoder) Result {
	return Result{Signature: signature, Body: e.Bytes()}
}

// refresh asks the application for the menu as it is now.
func (object *Object) refresh() {
	if object.base += int32(len(object.entries)); object.base > 1<<30 {
		object.base = 0
	}
	object.entries = platform.NumberMenu(object.menu())
	object.revision++
	object.stale = false
}

// Call answers one call made to the menu's path, on either interface it has.
func (object *Object) Call(call *dbus.Message) Result {
	switch call.Interface {
	case properties:
		return object.properties(call)
	case Interface:
	default:
		return failure("org.freedesktop.DBus.Error.UnknownInterface", call.Interface)
	}
	args := call.Args()
	var e dbus.Encoder
	switch call.Member {
	case "GetLayout":
		parent, depth, names := args.Int32(), args.Int32(), readNames(args)
		if args.Err() != nil || call.Signature != "iias" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (iias)")
		}
		if parent == 0 {
			// Reading the whole menu again, with no word that it is about to
			// show, is a host that kept the last layout; reading a submenu is
			// part of reading the one it has.
			if object.stale || object.entries == nil {
				object.refresh()
			}
			object.stale = true
		}
		e.Uint32(object.revision)
		if !object.writeLayout(&e, parent, depth, names) {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "no such menu item")
		}
		return reply("u"+nodeType, &e)
	case "GetGroupProperties":
		ids := readIDs(args)
		names := readNames(args)
		if args.Err() != nil || call.Signature != "aias" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (aias)")
		}
		object.writeGroup(&e, ids, names)
		return reply("a(ia{sv})", &e)
	case "GetProperty":
		id, name := args.Int32(), args.String()
		if args.Err() != nil || call.Signature != "is" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (is)")
		}
		if !object.writeProperty(&e, id, name) {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "no such property")
		}
		return reply("v", &e)
	case "Event":
		id, kind := args.Int32(), args.String()
		args.Skip(args.Signature())
		args.Uint32()
		if args.Err() != nil || call.Signature != "isvu" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (isvu)")
		}
		object.event(id, kind)
		return Result{}
	case "EventGroup":
		end := args.ArrayLength(8)
		var failed []int32
		for args.Pos() < end && args.Err() == nil {
			args.Align(8)
			id, kind := args.Int32(), args.String()
			args.Skip(args.Signature())
			args.Uint32()
			if args.Err() != nil {
				break
			}
			if !object.known(id) {
				failed = append(failed, id)
				continue
			}
			object.event(id, kind)
		}
		if args.Err() != nil || call.Signature != "a(isvu)" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (a(isvu))")
		}
		mark := e.BeginArray(4)
		for _, id := range failed {
			e.Int32(id)
		}
		e.EndArray(mark)
		return reply("ai", &e)
	case "AboutToShow":
		id := args.Int32()
		if args.Err() != nil || call.Signature != "i" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (i)")
		}
		// Only the menu's own opening is a chance to bring it up to date: the
		// ids of a submenu's entries must not change under a host that is
		// reading them.
		update := id == 0
		if update {
			object.refresh()
			object.stale = false
		}
		e.Bool(update)
		return reply("b", &e)
	case "AboutToShowGroup":
		ids := readIDs(args)
		if args.Err() != nil || call.Signature != "ai" {
			return failure("org.freedesktop.DBus.Error.InvalidArgs", "expected (ai)")
		}
		update := false
		for _, id := range ids {
			if id == 0 {
				object.refresh()
				update = true
			}
		}
		updated := e.BeginArray(4)
		if update {
			e.Int32(0)
		}
		e.EndArray(updated)
		errors := e.BeginArray(4)
		e.EndArray(errors)
		return reply("aiai", &e)
	}
	return failure("org.freedesktop.DBus.Error.UnknownMethod", call.Member)
}

func readNames(args *dbus.Decoder) []string {
	end := args.ArrayLength(4)
	var names []string
	for args.Pos() < end && args.Err() == nil {
		names = append(names, args.String())
	}
	return names
}

func readIDs(args *dbus.Decoder) []int32 {
	end := args.ArrayLength(4)
	var ids []int32
	for args.Pos() < end && args.Err() == nil {
		ids = append(ids, args.Int32())
	}
	return ids
}

// known reports whether id names the menu itself or one of its entries.
func (object *Object) known(id int32) bool {
	_, ok := object.index(id)
	return id == 0 || ok
}

// event applies what a host did with an entry: choosing one runs its handler.
func (object *Object) event(id int32, kind string) {
	if kind == "clicked" {
		if index, ok := object.index(id); ok {
			platform.Select(object.entries, index+1)
		}
	}
}

func (object *Object) properties(call *dbus.Message) Result {
	args := call.Args()
	var e dbus.Encoder
	switch call.Member {
	case "Get":
		iface, name := args.String(), args.String()
		if args.Err() != nil || iface != Interface || !writeObjectProperty(&e, name) {
			return failure("org.freedesktop.DBus.Error.UnknownProperty", name)
		}
		return reply("v", &e)
	case "GetAll":
		iface := args.String()
		mark := e.BeginArray(8)
		if iface == Interface {
			for _, name := range [...]string{"Version", "TextDirection", "Status", "IconThemePath"} {
				e.Align(8)
				e.String(name)
				writeObjectProperty(&e, name)
			}
		}
		e.EndArray(mark)
		return reply("a{sv}", &e)
	case "Set":
		return failure("org.freedesktop.DBus.Error.PropertyReadOnly", "read-only")
	}
	return failure("org.freedesktop.DBus.Error.UnknownMethod", call.Member)
}

// writeObjectProperty writes one property of the menu itself as a variant.
func writeObjectProperty(e *dbus.Encoder, name string) bool {
	switch name {
	case "Version":
		e.Variant("u")
		e.Uint32(version)
	case "TextDirection":
		e.Variant("s")
		e.String("ltr")
	case "Status":
		e.Variant("s")
		e.String("normal")
	case "IconThemePath":
		e.Variant("as")
		e.EndArray(e.BeginArray(4))
	default:
		return false
	}
	return true
}

// index is the position in the entries of the one with this id.
func (object *Object) index(id int32) (int, bool) {
	number := int64(id) - int64(object.base)
	if number < 1 || number > int64(len(object.entries)) {
		return 0, false
	}
	return int(number) - 1, true
}

// wireID is the id a host knows an entry by.
func (object *Object) wireID(entry *platform.MenuEntry) int32 { return object.base + int32(entry.ID) }

// writeLayout writes the layout node of parent and its descendants down to depth
// levels (all of them when depth is negative), after the revision.
func (object *Object) writeLayout(e *dbus.Encoder, parent, depth int32, names []string) bool {
	if parent == 0 {
		e.Align(8)
		e.Int32(0)
		writeProps(e, nil, object.hasChildren(nil), names)
		mark := e.BeginArray(1)
		if depth != 0 {
			cursor := 0
			for cursor < len(object.entries) {
				e.Variant(nodeType)
				object.writeNode(e, &cursor, depth-1, names)
			}
		}
		e.EndArray(mark)
		return true
	}
	index, ok := object.index(parent)
	if !ok {
		return false
	}
	object.writeNode(e, &index, depth, names)
	return true
}

// writeNode writes the entry at *cursor with its descendants, and leaves the
// cursor after them.
func (object *Object) writeNode(e *dbus.Encoder, cursor *int, depth int32, names []string) {
	entry := &object.entries[*cursor]
	e.Align(8)
	e.Int32(object.wireID(entry))
	writeProps(e, entry, entry.Children > 0, names)
	mark := e.BeginArray(1)
	if depth != 0 && entry.Children > 0 {
		end := *cursor + 1 + platform.SubtreeSize(object.entries, *cursor)
		*cursor++
		for *cursor < end {
			e.Variant(nodeType)
			object.writeNode(e, cursor, depth-1, names)
		}
	} else {
		*cursor += 1 + platform.SubtreeSize(object.entries, *cursor)
	}
	e.EndArray(mark)
}

// writeGroup writes the properties of each id, or of every entry when none is
// asked for.
func (object *Object) writeGroup(e *dbus.Encoder, ids []int32, names []string) {
	if len(ids) == 0 {
		ids = make([]int32, 0, len(object.entries)+1)
		ids = append(ids, 0)
		for _, entry := range object.entries {
			ids = append(ids, object.wireID(&entry))
		}
	}
	mark := e.BeginArray(8)
	for _, id := range ids {
		var entry *platform.MenuEntry
		if index, ok := object.index(id); ok {
			entry = &object.entries[index]
		} else if id != 0 {
			continue
		}
		e.Align(8)
		e.Int32(id)
		writeProps(e, entry, object.hasChildren(entry), names)
	}
	e.EndArray(mark)
}

func (object *Object) writeProperty(e *dbus.Encoder, id int32, name string) bool {
	var entry *platform.MenuEntry
	if index, ok := object.index(id); ok {
		entry = &object.entries[index]
	} else if id != 0 {
		return false
	}
	if !hasProp(entry, name) || !contains(propNames[:], name) {
		return false
	}
	writeOneProp(e, entry, object.hasChildren(entry), name)
	return true
}

// hasChildren reports whether an entry, or the menu itself when nil, has a submenu.
func (object *Object) hasChildren(entry *platform.MenuEntry) bool {
	if entry == nil {
		return len(object.entries) > 0
	}
	return entry.Children > 0
}

// propNames are the properties an entry carries, in the order they are written.
var propNames = [...]string{"type", "label", "enabled", "visible", "toggle-type", "toggle-state", "children-display"}

// hasProp reports whether the entry (the menu itself when nil) carries a
// property. Every property of an entry is written even when it has the value
// the interface defaults to: a host that keeps what it was last told, as KDE's
// does, would otherwise show a separator's old look on an entry that now takes
// its id when the menu is built again.
func hasProp(entry *platform.MenuEntry, name string) bool {
	return entry != nil || name == "children-display"
}

// writeProps writes the properties of an entry (the menu itself when entry is
// nil) as a{sv}; names, when not empty, limits them.
func writeProps(e *dbus.Encoder, entry *platform.MenuEntry, parent bool, names []string) {
	mark := e.BeginArray(8)
	for _, name := range propNames {
		if hasProp(entry, name) && (len(names) == 0 || contains(names, name)) {
			e.Align(8)
			e.String(name)
			writeOneProp(e, entry, parent, name)
		}
	}
	e.EndArray(mark)
}

func contains(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

// writeOneProp writes a property that hasProp reported as a variant.
func writeOneProp(e *dbus.Encoder, entry *platform.MenuEntry, parent bool, name string) {
	switch name {
	case "type":
		e.Variant("s")
		if entry.Separator {
			e.String("separator")
		} else {
			e.String("standard")
		}
	case "label":
		// '_' marks a shortcut in this interface, and a title is shown as written.
		e.Variant("s")
		e.String(strings.ReplaceAll(entry.Title, "_", "__"))
	case "enabled":
		e.Variant("b")
		e.Bool(!entry.Disabled)
	case "visible":
		e.Variant("b")
		e.Bool(true)
	case "toggle-type":
		e.Variant("s")
		if entry.Checked {
			e.String("checkmark")
		} else {
			e.String("")
		}
	case "toggle-state":
		e.Variant("i")
		if entry.Checked {
			e.Int32(1)
		} else {
			e.Int32(0)
		}
	case "children-display":
		e.Variant("s")
		if parent {
			e.String("submenu")
		} else {
			e.String("")
		}
	}
}

// Introspection describes the interface for a host that asks.
const Introspection = ` <interface name="com.canonical.dbusmenu">
  <property name="Version" type="u" access="read"/>
  <property name="TextDirection" type="s" access="read"/>
  <property name="Status" type="s" access="read"/>
  <property name="IconThemePath" type="as" access="read"/>
  <method name="GetLayout"><arg type="i" direction="in"/><arg type="i" direction="in"/><arg type="as" direction="in"/><arg type="u" direction="out"/><arg type="(ia{sv}av)" direction="out"/></method>
  <method name="GetGroupProperties"><arg type="ai" direction="in"/><arg type="as" direction="in"/><arg type="a(ia{sv})" direction="out"/></method>
  <method name="GetProperty"><arg type="i" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="out"/></method>
  <method name="Event"><arg type="i" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="in"/><arg type="u" direction="in"/></method>
  <method name="EventGroup"><arg type="a(isvu)" direction="in"/><arg type="ai" direction="out"/></method>
  <method name="AboutToShow"><arg type="i" direction="in"/><arg type="b" direction="out"/></method>
  <method name="AboutToShowGroup"><arg type="ai" direction="in"/><arg type="ai" direction="out"/><arg type="ai" direction="out"/></method>
  <signal name="ItemsPropertiesUpdated"><arg type="a(ia{sv})"/><arg type="a(ias)"/></signal>
  <signal name="LayoutUpdated"><arg type="u"/><arg type="i"/></signal>
  <signal name="ItemActivationRequested"><arg type="i"/><arg type="u"/></signal>
 </interface>`
