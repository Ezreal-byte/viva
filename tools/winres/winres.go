// Package winres writes the resources a Windows program carries in its own
// .exe: its icon, which Explorer, the taskbar and Alt+Tab draw for it, and its
// version information, which Task Manager and the file's properties show. They
// are added to a finished program: a new .rsrc section is appended to the linked
// .exe. Nothing is generated into the source directory, no resource compiler is
// needed, and it works from any host.
package winres

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Resource types.
const (
	typeIcon      = 3
	typeGroupIcon = 14
	typeVersion   = 16
)

// languageEnglishUS is the language of every resource written: a program with
// one set of strings uses the neutral choice of its installer, English.
const languageEnglishUS = 0x0409

// Info is what the file's properties show. Empty fields are left out.
type Info struct {
	// Name is the product's name.
	Name string
	// Description is what Task Manager lists the program as.
	Description string
	// Version is dotted decimal such as 1.2.3; at most four numbers are kept.
	Version   string
	Publisher string
	Copyright string
	// File is the executable's file name.
	File string
}

type resource struct {
	kind, id int
	data     []byte
}

// ErrHasResources is returned by Patch for a program that already carries
// resources of its own, which are not replaced.
var ErrHasResources = errors.New("winres: the program already has resources")

// Patch returns the 64-bit Windows program exe with the icon and the version
// information added. ico is an .ico file whose entries Windows reads directly
// (PNG or bitmap); nil leaves the icon out. The input is not modified.
func Patch(exe, ico []byte, info Info) ([]byte, error) {
	var resources []resource
	if ico != nil {
		icons, err := iconResources(ico)
		if err != nil {
			return nil, err
		}
		resources = append(resources, icons...)
	}
	if version := versionResource(info); version != nil {
		resources = append(resources, resource{typeVersion, 1, version})
	}
	if len(resources) == 0 {
		return nil, errors.New("winres: nothing to write")
	}
	return appendSection(exe, resources)
}

// iconResources splits an .ico into the RT_ICON images and the RT_GROUP_ICON
// directory that lists them, the form an executable stores an icon in.
func iconResources(ico []byte) ([]resource, error) {
	if len(ico) < 6 || binary.LittleEndian.Uint16(ico[0:]) != 0 || binary.LittleEndian.Uint16(ico[2:]) != 1 {
		return nil, errors.New("winres: the icon is not an .ico file")
	}
	count := int(binary.LittleEndian.Uint16(ico[4:]))
	if count == 0 || len(ico) < 6+16*count {
		return nil, errors.New("winres: the .ico directory is truncated")
	}
	var group bytes.Buffer
	binary.Write(&group, binary.LittleEndian, [3]uint16{0, 1, uint16(count)})
	var resources []resource
	for index := 0; index < count; index++ {
		entry := ico[6+16*index:]
		size, offset := int(binary.LittleEndian.Uint32(entry[8:])), int(binary.LittleEndian.Uint32(entry[12:]))
		if size <= 0 || offset < 6+16*count || offset+size > len(ico) {
			return nil, fmt.Errorf("winres: image %d of the .ico lies outside the file", index)
		}
		id := index + 1
		group.Write(entry[:8]) // width, height, colors, reserved, planes, bits
		binary.Write(&group, binary.LittleEndian, uint32(size))
		binary.Write(&group, binary.LittleEndian, uint16(id))
		resources = append(resources, resource{typeIcon, id, ico[offset : offset+size]})
	}
	// The group has id 1: Explorer shows the first group of the lowest id.
	return append(resources, resource{typeGroupIcon, 1, group.Bytes()}), nil
}

// versionResource builds a VS_VERSIONINFO block.
func versionResource(info Info) []byte {
	pairs := [][2]string{
		{"CompanyName", info.Publisher}, {"FileDescription", info.Description},
		{"FileVersion", info.Version}, {"InternalName", strings.TrimSuffix(info.File, ".exe")},
		{"LegalCopyright", info.Copyright}, {"OriginalFilename", info.File},
		{"ProductName", info.Name}, {"ProductVersion", info.Version},
	}
	var strs [][]byte
	for _, pair := range pairs {
		if pair[1] != "" {
			value := utf16Z(pair[1])
			strs = append(strs, block(pair[0], 1, len(value)/2, value, nil))
		}
	}
	if len(strs) == 0 {
		return nil
	}
	table := block(fmt.Sprintf("%04X%04X", languageEnglishUS, 1200), 1, 0, nil, strs)
	stringFileInfo := block("StringFileInfo", 1, 0, nil, [][]byte{table})
	translation := make([]byte, 4)
	binary.LittleEndian.PutUint32(translation, 1200<<16|languageEnglishUS)
	variable := block("Translation", 0, len(translation), translation, nil)
	varFileInfo := block("VarFileInfo", 1, 0, nil, [][]byte{variable})

	numbers := versionNumbers(info.Version)
	fixed := make([]byte, 52)
	for index, value := range []uint32{
		0xFEEF04BD, 0x00010000,
		numbers[0]<<16 | numbers[1], numbers[2]<<16 | numbers[3],
		numbers[0]<<16 | numbers[1], numbers[2]<<16 | numbers[3],
		0x3F, 0, 0x00040004 /* VOS_NT_WINDOWS32 */, 1 /* VFT_APP */, 0, 0, 0,
	} {
		binary.LittleEndian.PutUint32(fixed[4*index:], value)
	}
	return block("VS_VERSION_INFO", 0, len(fixed), fixed, [][]byte{stringFileInfo, varFileInfo})
}

// block is one node of a version resource: lengths, type, key, a value and
// children, each part padded to four bytes. valueLength counts the unit the
// node's type says (bytes for binary, characters for text).
func block(key string, kind, valueLength int, value []byte, children [][]byte) []byte {
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, uint16(valueLength), uint16(kind)})
	out.Write(utf16Z(key))
	pad(&out)
	out.Write(value)
	pad(&out)
	for _, child := range children {
		out.Write(child)
		pad(&out)
	}
	data := out.Bytes()
	binary.LittleEndian.PutUint16(data, uint16(len(data)))
	return data
}

func pad(buffer *bytes.Buffer) {
	for buffer.Len()%4 != 0 {
		buffer.WriteByte(0)
	}
}

func utf16Z(text string) []byte {
	units := append(utf16.Encode([]rune(text)), 0)
	out := make([]byte, 2*len(units))
	for index, unit := range units {
		binary.LittleEndian.PutUint16(out[2*index:], unit)
	}
	return out
}

func versionNumbers(version string) (numbers [4]uint32) {
	for index, part := range strings.SplitN(version, ".", 4) {
		digits := part
		for position, r := range part {
			if r < '0' || r > '9' {
				digits = part[:position]
				break
			}
		}
		value, _ := strconv.ParseUint(digits, 10, 16)
		numbers[index] = uint32(value)
	}
	return numbers
}

// tree lays the resources out as the three-level tree Windows reads (type, then
// id, then language) in one section. The data entries hold offsets within the
// section; entries lists where, for the caller to turn into addresses.
func tree(resources []resource) (section []byte, entries []int) {
	sort.SliceStable(resources, func(a, b int) bool {
		if resources[a].kind != resources[b].kind {
			return resources[a].kind < resources[b].kind
		}
		return resources[a].id < resources[b].id
	})
	var kinds []int
	byKind := map[int][]resource{}
	for _, r := range resources {
		if _, seen := byKind[r.kind]; !seen {
			kinds = append(kinds, r.kind)
		}
		byKind[r.kind] = append(byKind[r.kind], r)
	}

	const directory, entry, dataEntry = 16, 8, 16
	size := directory + entry*len(kinds)
	for _, kind := range kinds {
		size += directory + entry*len(byKind[kind]) + (directory+entry)*len(byKind[kind])
	}
	size += dataEntry * len(resources)
	dataStart := (size + 3) &^ 3

	section = make([]byte, dataStart)
	put32 := func(at int, value uint32) { binary.LittleEndian.PutUint32(section[at:], value) }
	put16 := func(at int, value uint16) { binary.LittleEndian.PutUint16(section[at:], value) }
	header := func(at, named, ids int) { put16(at+12, uint16(named)); put16(at+14, uint16(ids)) }

	header(0, 0, len(kinds))
	next := directory + entry*len(kinds) // where the next subdirectory goes
	dataEntries := size - dataEntry*len(resources)
	var payload bytes.Buffer
	for kindIndex, kind := range kinds {
		list := byKind[kind]
		put32(directory+entry*kindIndex, uint32(kind))
		put32(directory+entry*kindIndex+4, 0x80000000|uint32(next))
		idDirectory := next
		header(idDirectory, 0, len(list))
		next += directory + entry*len(list)
		for idIndex, r := range list {
			put32(idDirectory+directory+entry*idIndex, uint32(r.id))
			put32(idDirectory+directory+entry*idIndex+4, 0x80000000|uint32(next))
			languageDirectory := next
			header(languageDirectory, 0, 1)
			put32(languageDirectory+directory, languageEnglishUS)
			put32(languageDirectory+directory+4, uint32(dataEntries))
			next += directory + entry

			put32(dataEntries, uint32(dataStart+payload.Len()))
			entries = append(entries, dataEntries)
			put32(dataEntries+4, uint32(len(r.data)))
			dataEntries += dataEntry
			payload.Write(r.data)
			pad(&payload)
		}
	}
	return append(section, payload.Bytes()...), entries
}

// appendSection adds the resources to a PE32+ file as a new last section and
// points the resource directory at it.
func appendSection(exe []byte, resources []resource) ([]byte, error) {
	bad := errors.New("winres: not a 64-bit Windows program")
	if len(exe) < 0x40 || exe[0] != 'M' || exe[1] != 'Z' {
		return nil, bad
	}
	pe := int(binary.LittleEndian.Uint32(exe[0x3c:]))
	if pe < 0 || pe+24+112+16*16 > len(exe) || string(exe[pe:pe+4]) != "PE\x00\x00" {
		return nil, bad
	}
	le := binary.LittleEndian
	sections, optionalSize := int(le.Uint16(exe[pe+6:])), int(le.Uint16(exe[pe+20:]))
	optional := pe + 24
	if le.Uint16(exe[optional:]) != 0x20b || optionalSize < 112+16*8 {
		return nil, bad
	}
	sectionAlignment, fileAlignment := le.Uint32(exe[optional+32:]), le.Uint32(exe[optional+36:])
	headersSize := int(le.Uint32(exe[optional+60:]))
	resourceDirectory := optional + 112 + 2*8
	if le.Uint32(exe[resourceDirectory:]) != 0 {
		return nil, ErrHasResources
	}
	if sectionAlignment == 0 || fileAlignment == 0 {
		return nil, bad
	}
	table := optional + optionalSize
	slot := table + 40*sections
	if slot+40 > headersSize || slot+40 > len(exe) || !allZero(exe[slot:slot+40]) {
		return nil, errors.New("winres: no room in the program's headers for another section")
	}
	end := uint32(0)
	for index := 0; index < sections; index++ {
		header := exe[table+40*index:]
		end = max(end, le.Uint32(header[12:])+max(le.Uint32(header[8:]), le.Uint32(header[16:])))
	}
	align := func(value, to uint32) uint32 { return (value + to - 1) / to * to }

	section, entries := tree(resources)
	virtual := align(end, sectionAlignment)
	for _, at := range entries {
		le.PutUint32(section[at:], le.Uint32(section[at:])+virtual)
	}
	rawSize := align(uint32(len(section)), fileAlignment)
	rawPointer := align(uint32(len(exe)), fileAlignment)

	out := make([]byte, int(rawPointer+rawSize))
	copy(out, exe)
	copy(out[rawPointer:], section)
	header := out[slot : slot+40]
	copy(header, ".rsrc")
	le.PutUint32(header[8:], uint32(len(section)))
	le.PutUint32(header[12:], virtual)
	le.PutUint32(header[16:], rawSize)
	le.PutUint32(header[20:], rawPointer)
	le.PutUint32(header[36:], 0x40000040) // initialized data, readable
	le.PutUint16(out[pe+6:], uint16(sections+1))
	le.PutUint32(out[optional+8:], le.Uint32(out[optional+8:])+rawSize) // SizeOfInitializedData
	le.PutUint32(out[optional+56:], align(virtual+uint32(len(section)), sectionAlignment))
	le.PutUint32(out[resourceDirectory:], virtual)
	le.PutUint32(out[resourceDirectory+4:], uint32(len(section)))
	return out, nil
}

func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}
