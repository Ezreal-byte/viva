package dbusmenu

import (
	"reflect"
	"testing"

	"github.com/vivaui/viva/internal/dbus"
	"github.com/vivaui/viva/internal/platform"
)

type node struct {
	id       int32
	props    map[string]any
	children []node
}

// readNode decodes one (ia{sv}av) the way a host does.
func readNode(t *testing.T, d *dbus.Decoder) node {
	t.Helper()
	d.Align(8)
	n := node{id: d.Int32(), props: map[string]any{}}
	end := d.ArrayLength(8)
	for d.Pos() < end && d.Err() == nil {
		d.Align(8)
		key := d.String()
		switch sig := d.Signature(); sig {
		case "s":
			n.props[key] = d.String()
		case "b":
			n.props[key] = d.Bool()
		case "i":
			n.props[key] = d.Int32()
		default:
			t.Fatalf("property %q has signature %q", key, sig)
		}
	}
	end = d.ArrayLength(1)
	for d.Pos() < end && d.Err() == nil {
		if sig := d.Signature(); sig != nodeType {
			t.Fatalf("child signature %q", sig)
		}
		n.children = append(n.children, readNode(t, d))
	}
	if d.Err() != nil {
		t.Fatal(d.Err())
	}
	return n
}

func call(member, signature string, args func(e *dbus.Encoder)) *dbus.Message {
	var e dbus.Encoder
	if args != nil {
		args(&e)
	}
	return &dbus.Message{Type: dbus.TypeCall, Path: Path, Interface: Interface, Member: member, Signature: signature, Body: e.Bytes()}
}

func layoutCall(parent, depth int32, names ...string) *dbus.Message {
	return call("GetLayout", "iias", func(e *dbus.Encoder) {
		e.Int32(parent)
		e.Int32(depth)
		mark := e.BeginArray(4)
		for _, name := range names {
			e.String(name)
		}
		e.EndArray(mark)
	})
}

func newTestObject(chosen *[]string, builds *int) *Object {
	pick := func(name string) func() { return func() { *chosen = append(*chosen, name) } }
	return New(func() []platform.MenuItem {
		*builds++
		return []platform.MenuItem{
			{Title: "Header", Disabled: true},
			{Title: "Sessions", Submenu: []platform.MenuItem{
				{Title: "alpha", OnSelect: pick("alpha")},
				{Title: "beta", OnSelect: pick("beta")},
			}},
			{Separator: true},
			{Title: "New_terminal", Checked: true, OnSelect: pick("terminal")},
			{Title: "Quit", OnSelect: pick("quit")},
		}
	})
}

func succeeded(t *testing.T, result Result) []byte {
	t.Helper()
	if result.ErrName != "" {
		t.Fatalf("error %s: %s", result.ErrName, result.ErrText)
	}
	return result.Body
}

func TestLayoutIsTheWholeMenuWithItsPropertiesAndIDsInEntryOrder(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	result := object.Call(layoutCall(0, -1))
	if result.Signature != "u"+nodeType {
		t.Fatalf("signature %q", result.Signature)
	}
	d := dbus.NewDecoder(succeeded(t, result))
	if revision := d.Uint32(); revision != 1 {
		t.Fatalf("revision %d", revision)
	}
	root := readNode(t, d)
	if !d.Done() {
		t.Fatal("bytes left over after the layout")
	}
	if root.id != 0 || root.props["children-display"] != "submenu" || len(root.children) != 5 {
		t.Fatalf("root = %+v", root)
	}
	titles := []string{}
	for _, child := range root.children {
		label, _ := child.props["label"].(string)
		titles = append(titles, label)
	}
	if !reflect.DeepEqual(titles, []string{"Header", "Sessions", "", "New__terminal", "Quit"}) {
		t.Fatalf("labels %q", titles)
	}
	header, sessions, separator, terminal := root.children[0], root.children[1], root.children[2], root.children[3]
	if header.id != 1 || header.props["enabled"] != false {
		t.Fatalf("header %+v", header)
	}
	if sessions.id != 2 || sessions.props["children-display"] != "submenu" || len(sessions.children) != 2 || sessions.children[0].id != 3 || sessions.children[1].id != 4 {
		t.Fatalf("sessions %+v", sessions)
	}
	if separator.id != 5 || separator.props["type"] != "separator" {
		t.Fatalf("separator %+v", separator)
	}
	if terminal.id != 6 || terminal.props["toggle-type"] != "checkmark" || terminal.props["toggle-state"] != int32(1) {
		t.Fatalf("terminal %+v", terminal)
	}
	// Every property is stated, defaults included, so a host that keeps what it
	// was told does not carry a separator's look over to the entry that takes
	// its id when the menu is built again.
	quit := root.children[4]
	want := map[string]any{"type": "standard", "label": "Quit", "enabled": true, "visible": true, "toggle-type": "", "toggle-state": int32(0), "children-display": ""}
	if quit.id != 7 || !reflect.DeepEqual(quit.props, want) {
		t.Fatalf("quit = %+v", quit)
	}
	if separator.props["label"] != "" || separator.props["enabled"] != true {
		t.Fatalf("a separator leaves a property unsaid: %+v", separator.props)
	}
}

func TestLayoutDepthAndParentSelectWhatIsWritten(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	object.Call(layoutCall(0, -1))

	d := dbus.NewDecoder(succeeded(t, object.Call(layoutCall(0, 1))))
	d.Uint32()
	flat := readNode(t, d)
	if len(flat.children) != 5 || len(flat.children[1].children) != 0 {
		t.Fatalf("depth 1 wrote grandchildren: %+v", flat)
	}
	d = dbus.NewDecoder(succeeded(t, object.Call(layoutCall(0, 0))))
	d.Uint32()
	if bare := readNode(t, d); len(bare.children) != 0 {
		t.Fatalf("depth 0 wrote children: %+v", bare)
	}
	d = dbus.NewDecoder(succeeded(t, object.Call(layoutCall(object.base+2, -1))))
	d.Uint32()
	sub := readNode(t, d)
	if sub.id != object.base+2 || len(sub.children) != 2 || sub.children[1].props["label"] != "beta" {
		t.Fatalf("submenu %+v", sub)
	}
	if r := object.Call(layoutCall(99, -1)); r.ErrName == "" {
		t.Fatal("an unknown parent was answered")
	}
	d = dbus.NewDecoder(succeeded(t, object.Call(layoutCall(0, -1, "label"))))
	d.Uint32()
	only := readNode(t, d)
	if _, set := only.props["children-display"]; set || only.children[0].props["label"] != "Header" {
		t.Fatalf("property filter ignored: %+v", only)
	}
	if len(only.children[0].props) != 1 {
		t.Fatalf("properties that were not asked for were written: %+v", only.children[0].props)
	}
}

func TestOpeningTheMenuAsksTheApplicationAgain(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	object.Call(layoutCall(0, -1))
	if builds != 1 {
		t.Fatalf("builds %d", builds)
	}
	// A host that says it is about to show the menu is told to read it again,
	// and reading it then does not build it a second time.
	d := dbus.NewDecoder(succeeded(t, object.Call(call("AboutToShow", "i", func(e *dbus.Encoder) { e.Int32(0) }))))
	if !d.Bool() || builds != 2 {
		t.Fatalf("AboutToShow(0) did not refresh: builds %d", builds)
	}
	d = dbus.NewDecoder(succeeded(t, object.Call(layoutCall(0, -1))))
	if revision := d.Uint32(); revision != 2 || builds != 2 {
		t.Fatalf("revision %d builds %d", revision, builds)
	}
	// A layout read with no announcement is a host that kept the old one.
	object.Call(layoutCall(0, -1))
	if builds != 3 {
		t.Fatalf("builds %d", builds)
	}
	d = dbus.NewDecoder(succeeded(t, object.Call(call("AboutToShow", "i", func(e *dbus.Encoder) { e.Int32(2) }))))
	if d.Bool() || builds != 3 {
		t.Fatal("a submenu's opening rebuilt the menu under its own ids")
	}
}

func event(id int32, kind string) *dbus.Message {
	return call("Event", "isvu", func(e *dbus.Encoder) {
		e.Int32(id)
		e.String(kind)
		e.Variant("s")
		e.String("payload")
		e.Uint32(1234)
	})
}

func TestClickingAnEntryRunsOnlyThatEntry(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	object.Call(layoutCall(0, -1))
	for _, message := range []*dbus.Message{
		event(4, "hovered"), event(1, "clicked"), event(2, "clicked"), event(5, "clicked"),
		event(4, "clicked"), event(7, "clicked"), event(99, "clicked"), event(-1, "clicked"),
	} {
		if r := object.Call(message); r.ErrName != "" || r.Body != nil {
			t.Fatalf("Event answered %+v", r)
		}
	}
	if !reflect.DeepEqual(chosen, []string{"beta", "quit"}) {
		t.Fatalf("chosen %v", chosen)
	}
	group := call("EventGroup", "a(isvu)", func(e *dbus.Encoder) {
		mark := e.BeginArray(8)
		for _, item := range []struct {
			id   int32
			kind string
		}{{3, "clicked"}, {50, "clicked"}} {
			e.Align(8)
			e.Int32(item.id)
			e.String(item.kind)
			e.Variant("i")
			e.Int32(0)
			e.Uint32(0)
		}
		e.EndArray(mark)
	})
	d := dbus.NewDecoder(succeeded(t, object.Call(group)))
	end := d.ArrayLength(4)
	var failed []int32
	for d.Pos() < end {
		failed = append(failed, d.Int32())
	}
	if !reflect.DeepEqual(failed, []int32{50}) || !reflect.DeepEqual(chosen, []string{"beta", "quit", "alpha"}) {
		t.Fatalf("failed %v chosen %v", failed, chosen)
	}
}

func TestPropertiesOfEntriesAreReadableOnTheirOwn(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	object.Call(layoutCall(0, -1))

	d := dbus.NewDecoder(succeeded(t, object.Call(call("GetProperty", "is", func(e *dbus.Encoder) { e.Int32(3); e.String("label") }))))
	if sig := d.Signature(); sig != "s" || d.String() != "alpha" {
		t.Fatal("GetProperty label")
	}
	if r := object.Call(call("GetProperty", "is", func(e *dbus.Encoder) { e.Int32(3); e.String("icon-name") })); r.ErrName == "" {
		t.Fatal("a property the menu does not have was reported")
	}
	group := call("GetGroupProperties", "aias", func(e *dbus.Encoder) {
		ids := e.BeginArray(4)
		e.Int32(1)
		e.Int32(5)
		e.EndArray(ids)
		e.EndArray(e.BeginArray(4))
	})
	d = dbus.NewDecoder(succeeded(t, object.Call(group)))
	end := d.ArrayLength(8)
	got := map[int32]int{}
	for d.Pos() < end && d.Err() == nil {
		d.Align(8)
		id := d.Int32()
		propEnd := d.ArrayLength(8)
		for d.Pos() < propEnd {
			d.Align(8)
			_ = d.String()
			d.Skip(d.Signature())
			got[id]++
		}
	}
	if d.Err() != nil || got[1] != len(propNames) || got[5] != len(propNames) {
		t.Fatalf("group properties %v err %v", got, d.Err())
	}

	properties := func(member string, args func(*dbus.Encoder)) *dbus.Message {
		message := call(member, "", args)
		message.Interface = "org.freedesktop.DBus.Properties"
		return message
	}
	all := properties("GetAll", func(e *dbus.Encoder) { e.String(Interface) })
	d = dbus.NewDecoder(succeeded(t, object.Call(all)))
	end = d.ArrayLength(8)
	names := []string{}
	for d.Pos() < end && d.Err() == nil {
		d.Align(8)
		names = append(names, d.String())
		d.Skip(d.Signature())
	}
	if !reflect.DeepEqual(names, []string{"Version", "TextDirection", "Status", "IconThemePath"}) {
		t.Fatalf("properties %v", names)
	}
	d = dbus.NewDecoder(succeeded(t, object.Call(properties("Get", func(e *dbus.Encoder) { e.String(Interface); e.String("Version") }))))
	if sig := d.Signature(); sig != "u" || d.Uint32() != 3 {
		t.Fatal("Version is not 3")
	}
	if r := object.Call(properties("Set", nil)); r.ErrName == "" {
		t.Fatal("a property was set")
	}
}

func TestMalformedAndUnknownCallsAreRefused(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	for _, message := range []*dbus.Message{
		call("Nonsense", "", nil),
		call("GetLayout", "i", func(e *dbus.Encoder) { e.Int32(0) }),
		call("Event", "isvu", func(e *dbus.Encoder) { e.Int32(1) }),
		{Interface: "org.example.Other", Member: "X", Path: Path},
	} {
		if r := object.Call(message); r.ErrName == "" {
			t.Fatalf("%s %s was answered", message.Interface, message.Member)
		}
	}
}

func TestAnEmptyMenuHasAnEmptyLayout(t *testing.T) {
	object := New(func() []platform.MenuItem { return nil })
	d := dbus.NewDecoder(succeeded(t, object.Call(layoutCall(0, -1))))
	d.Uint32()
	root := readNode(t, d)
	if root.id != 0 || len(root.children) != 0 || root.props["children-display"] != "" {
		t.Fatalf("root %+v", root)
	}
}

func FuzzCallNeverPanicsOnAnyBody(f *testing.F) {
	members := []string{"GetLayout", "GetGroupProperties", "GetProperty", "Event", "EventGroup", "AboutToShow", "AboutToShowGroup"}
	signatures := []string{"iias", "aias", "is", "isvu", "a(isvu)", "i", "ai", ""}
	f.Add(uint8(0), uint8(0), []byte{0, 0, 0, 0, 255, 255, 255, 255, 0, 0, 0, 0})
	f.Add(uint8(4), uint8(4), []byte{4, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3})
	f.Fuzz(func(t *testing.T, member, signature uint8, body []byte) {
		object := New(func() []platform.MenuItem {
			return []platform.MenuItem{{Title: "a", OnSelect: func() {}}, {Title: "b", Submenu: []platform.MenuItem{{Title: "c", OnSelect: func() {}}}}}
		})
		object.Call(&dbus.Message{
			Path: Path, Interface: Interface, Member: members[int(member)%len(members)],
			Signature: signatures[int(signature)%len(signatures)], Body: body,
		})
	})
}

func TestEachBuildOfTheMenuTakesIDsNoEarlierBuildUsed(t *testing.T) {
	var chosen []string
	var builds int
	object := newTestObject(&chosen, &builds)
	ids := func() []int32 {
		d := dbus.NewDecoder(succeeded(t, object.Call(layoutCall(0, -1))))
		d.Uint32()
		var found []int32
		var walk func(n node)
		walk = func(n node) {
			if n.id != 0 {
				found = append(found, n.id)
			}
			for _, child := range n.children {
				walk(child)
			}
		}
		walk(readNode(t, d))
		return found
	}
	first := ids()
	second := ids() // the whole menu read again with no announcement is built again
	if len(first) != 7 || len(second) != 7 {
		t.Fatalf("ids %v then %v", first, second)
	}
	if second[0] <= first[len(first)-1] {
		t.Fatalf("the second build reused ids: %v then %v", first, second)
	}
	// An id from the earlier build leads nowhere, and one from this build to its entry.
	object.Call(event(first[2], "clicked"))
	if len(chosen) != 0 {
		t.Fatalf("an old id ran %v", chosen)
	}
	object.Call(event(second[2], "clicked"))
	if !reflect.DeepEqual(chosen, []string{"alpha"}) {
		t.Fatalf("chosen %v", chosen)
	}
	if r := object.Call(layoutCall(first[1], -1)); r.ErrName == "" { // a submenu is read from the build the host has
		t.Fatal("a layout was served for an id of an earlier build")
	}
}
