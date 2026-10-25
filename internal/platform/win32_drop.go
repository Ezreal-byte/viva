package platform

// DropEffect is the answer an OLE drop target gives a source that offers the
// allowed effects (DROPEFFECT_COPY 1, MOVE 2, LINK 4), for a window that does or
// does not take files at the pointer. Files are taken as a copy of their paths
// whatever the source calls the operation, so a copy is preferred, then a link,
// then a move; none is the "no drop here" cursor.
func DropEffect(accepts bool, allowed uint32) uint32 {
	switch {
	case !accepts:
		return 0
	case allowed&1 != 0:
		return 1
	case allowed&4 != 0:
		return 4
	case allowed&2 != 0:
		return 2
	}
	return 0
}
