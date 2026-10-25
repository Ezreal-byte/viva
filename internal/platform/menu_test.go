package platform

import "testing"

func TestNumberMenuIsDepthFirstFromOne(t *testing.T) {
	var chosen []string
	pick := func(name string) func() { return func() { chosen = append(chosen, name) } }
	entries := NumberMenu([]MenuItem{
		{Title: "a", OnSelect: pick("a")},
		{Title: "b", Submenu: []MenuItem{
			{Title: "b1", OnSelect: pick("b1")},
			{Title: "b2", Submenu: []MenuItem{{Title: "b2x", OnSelect: pick("b2x")}}},
		}},
		{Separator: true, Title: "ignored", Disabled: true, OnSelect: pick("sep")},
		{Title: "off", Disabled: true, OnSelect: pick("off")},
		{Title: "c", Checked: true, OnSelect: pick("c")},
	})
	want := []struct {
		title    string
		children int
	}{{"a", 0}, {"b", 2}, {"b1", 0}, {"b2", 1}, {"b2x", 0}, {"ignored", 0}, {"off", 0}, {"c", 0}}
	if len(entries) != len(want) {
		t.Fatalf("%d entries: %+v", len(entries), entries)
	}
	for index, entry := range entries {
		if entry.ID != index+1 || entry.Title != want[index].title || entry.Children != want[index].children {
			t.Fatalf("entry %d = %+v, want %+v", index, entry, want[index])
		}
	}
	if !entries[5].Separator || entries[5].Disabled || entries[5].OnSelect != nil {
		t.Fatalf("a separator kept fields it must not have: %+v", entries[5])
	}
	if !entries[7].Checked {
		t.Fatal("checked flag lost")
	}
	if TopLevel(entries) != 5 || SubtreeSize(entries, 1) != 3 || SubtreeSize(entries, 3) != 1 || SubtreeSize(entries, 0) != 0 {
		t.Fatalf("TopLevel %d, SubtreeSize(b) %d", TopLevel(entries), SubtreeSize(entries, 1))
	}
	for id := 0; id <= len(entries)+1; id++ {
		Select(entries, id)
	}
	if got := len(chosen); got != 4 || chosen[0] != "a" || chosen[1] != "b1" || chosen[2] != "b2x" || chosen[3] != "c" {
		t.Fatalf("chosen = %v: only reachable entries run", chosen)
	}
}

func TestNumberMenuOfNothingIsEmpty(t *testing.T) {
	if entries := NumberMenu(nil); len(entries) != 0 {
		t.Fatalf("entries = %+v", entries)
	}
}
