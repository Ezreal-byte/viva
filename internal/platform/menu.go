package platform

// MenuEntry is one MenuItem in the order a native menu is built: the entries of
// a menu, each followed by its submenu's entries, depth first. Its ID is its
// position in that order, counting from 1, so the number a platform hands back
// when the user chooses an entry leads straight to it.
type MenuEntry struct {
	ID        int
	Title     string
	Separator bool
	Disabled  bool
	Checked   bool
	// Children is how many entries the submenu has directly. They follow this
	// entry, each with its own submenu's entries after it.
	Children int
	OnSelect func()
}

// NumberMenu lays a menu out as entries, ID 1 first.
func NumberMenu(items []MenuItem) []MenuEntry {
	entries := make([]MenuEntry, 0, countMenu(items))
	return appendMenu(entries, items)
}

func countMenu(items []MenuItem) int {
	count := len(items)
	for index := range items {
		count += countMenu(items[index].Submenu)
	}
	return count
}

func appendMenu(entries []MenuEntry, items []MenuItem) []MenuEntry {
	for index := range items {
		item := &items[index]
		entry := MenuEntry{
			ID: len(entries) + 1, Title: item.Title, Separator: item.Separator,
			Disabled: item.Disabled, Checked: item.Checked, Children: len(item.Submenu), OnSelect: item.OnSelect,
		}
		if item.Separator {
			entry.Disabled, entry.Checked, entry.Children, entry.OnSelect = false, false, 0, nil
			entries = append(entries, entry)
			continue
		}
		entries = append(entries, entry)
		entries = appendMenu(entries, item.Submenu)
	}
	return entries
}

// Select runs the handler of the entry with this ID, if it has one that the
// user can reach: not a separator, not disabled and not a submenu's parent.
func Select(entries []MenuEntry, id int) {
	if id < 1 || id > len(entries) {
		return
	}
	entry := entries[id-1]
	if entry.OnSelect != nil && !entry.Separator && !entry.Disabled && entry.Children == 0 {
		entry.OnSelect()
	}
}

// SubtreeSize is how many entries follow entries[index] as its submenu, at
// every depth.
func SubtreeSize(entries []MenuEntry, index int) int {
	size := 0
	for child, remaining := index+1, entries[index].Children; remaining > 0; remaining-- {
		below := 1 + SubtreeSize(entries, child)
		size += below
		child += below
	}
	return size
}

// TopLevel is how many entries the menu itself has, not counting its submenus'.
func TopLevel(entries []MenuEntry) int {
	count := 0
	for index := 0; index < len(entries); index += 1 + SubtreeSize(entries, index) {
		count++
	}
	return count
}
