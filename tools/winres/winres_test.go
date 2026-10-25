package winres

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"testing"
)

// ico makes an .ico of PNG-looking entries whose payloads are easy to recognize.
func ico(payloads ...string) []byte {
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(payloads))})
	offset := 6 + 16*len(payloads)
	for index, payload := range payloads {
		out.Write([]byte{byte(16 << index), byte(16 << index), 0, 0})
		binary.Write(&out, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&out, binary.LittleEndian, [2]uint32{uint32(len(payload)), uint32(offset)})
		offset += len(payload)
	}
	for _, payload := range payloads {
		out.WriteString(payload)
	}
	return out.Bytes()
}

// program is the smallest 64-bit Windows image the headers allow: one .text
// section and a header area with room for more.
func program() []byte {
	exe := make([]byte, 0x400)
	le := binary.LittleEndian
	copy(exe, "MZ")
	le.PutUint32(exe[0x3c:], 0x40)
	copy(exe[0x40:], "PE\x00\x00")
	le.PutUint16(exe[0x44:], 0x8664)
	le.PutUint16(exe[0x46:], 1)   // sections
	le.PutUint16(exe[0x54:], 240) // optional header size
	optional := 0x40 + 24
	le.PutUint16(exe[optional:], 0x20b)
	le.PutUint32(exe[optional+8:], 0x200)
	le.PutUint32(exe[optional+32:], 0x1000) // section alignment
	le.PutUint32(exe[optional+36:], 0x200)  // file alignment
	le.PutUint32(exe[optional+56:], 0x2000) // size of image
	le.PutUint32(exe[optional+60:], 0x200)  // size of headers
	le.PutUint32(exe[optional+108:], 16)
	text := optional + 240
	copy(exe[text:], ".text")
	le.PutUint32(exe[text+8:], 0x100)
	le.PutUint32(exe[text+12:], 0x1000)
	le.PutUint32(exe[text+16:], 0x200)
	le.PutUint32(exe[text+20:], 0x200)
	return exe
}

// walk reads the resource tree of a patched program the way Windows does: from
// the resource directory in the optional header, through type, id and language,
// to a data entry whose address is a virtual one.
func walk(t *testing.T, patched []byte) map[[2]int][]byte {
	t.Helper()
	file, err := pe.NewFile(bytes.NewReader(patched))
	if err != nil {
		t.Fatalf("the patched program is not a PE file: %v", err)
	}
	section := file.Section(".rsrc")
	if section == nil {
		t.Fatal("no .rsrc section")
	}
	directory := file.OptionalHeader.(*pe.OptionalHeader64).DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE]
	if directory.VirtualAddress != section.VirtualAddress || directory.Size != section.VirtualSize {
		t.Fatalf("the resource directory (%+v) does not name the section (%#x, %#x)", directory, section.VirtualAddress, section.VirtualSize)
	}
	if got := file.OptionalHeader.(*pe.OptionalHeader64).SizeOfImage; got < section.VirtualAddress+section.VirtualSize || got%0x1000 != 0 {
		t.Errorf("size of image %#x does not cover the new section or its alignment", got)
	}
	data, err := section.Data()
	if err != nil {
		t.Fatal(err)
	}
	u16 := func(at int) int { return int(binary.LittleEndian.Uint16(data[at:])) }
	u32 := func(at int) int { return int(binary.LittleEndian.Uint32(data[at:])) }
	children := func(at int) (ids, offsets []int) {
		count := u16(at+12) + u16(at+14)
		for index := 0; index < count; index++ {
			ids = append(ids, u32(at+16+8*index))
			offsets = append(offsets, u32(at+16+8*index+4)&0x7fffffff)
		}
		return
	}
	found := map[[2]int][]byte{}
	kinds, kindOffsets := children(0)
	for index, kind := range kinds {
		ids, idOffsets := children(kindOffsets[index])
		for position, id := range ids {
			_, languageOffsets := children(idOffsets[position])
			entry := languageOffsets[0]
			start, length := u32(entry)-int(section.VirtualAddress), u32(entry+4)
			found[[2]int{kind, id}] = data[start : start+length]
		}
	}
	return found
}

func TestIconsAndVersionAreReachableThroughTheResourceTree(t *testing.T) {
	patched, err := Patch(program(), ico("first-image", "second-image-longer"), Info{Name: "Motty", Description: "SSH client", Version: "1.2.3", Publisher: "Acme", File: "motty.exe"})
	if err != nil {
		t.Fatal(err)
	}
	found := walk(t, patched)
	if got := found[[2]int{typeIcon, 1}]; string(got) != "first-image" {
		t.Errorf("icon 1 is %q", got)
	}
	if got := found[[2]int{typeIcon, 2}]; string(got) != "second-image-longer" {
		t.Errorf("icon 2 is %q", got)
	}
	group := found[[2]int{typeGroupIcon, 1}]
	if len(group) != 6+14*2 || binary.LittleEndian.Uint16(group[4:]) != 2 {
		t.Fatalf("the icon group is %d bytes: %v", len(group), group)
	}
	if id := binary.LittleEndian.Uint16(group[6+14+12:]); id != 2 {
		t.Errorf("the second group entry names icon %d, want 2", id)
	}
	version := found[[2]int{typeVersion, 1}]
	if length := int(binary.LittleEndian.Uint16(version)); length != len(version) {
		t.Errorf("the version block says it is %d bytes but is %d", length, len(version))
	}
	for _, text := range []string{"VS_VERSION_INFO", "ProductName", "Motty", "SSH client", "Acme", "motty.exe", "1.2.3"} {
		if !bytes.Contains(version, utf16Z(text)[:len(utf16Z(text))-2]) {
			t.Errorf("the version information does not hold %q", text)
		}
	}
	if fixed := binary.LittleEndian.Uint32(version[40:]); fixed != 0xFEEF04BD {
		t.Errorf("the fixed file info starts with %#x", fixed)
	}
}

func TestPatchKeepsTheProgramAndRefusesWhatItWouldClobber(t *testing.T) {
	original := program()
	patched, err := Patch(original, ico("x"), Info{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if len(patched) <= len(original) {
		t.Fatal("the patched program must be longer")
	}
	if !bytes.Equal(patched[0x200:0x400], original[0x200:0x400]) {
		t.Error("the existing section's bytes changed")
	}
	if _, err := Patch(patched, ico("y"), Info{Name: "A"}); !errors.Is(err, ErrHasResources) {
		t.Errorf("patching a program that has resources gave %v, want ErrHasResources", err)
	}
	if _, err := Patch([]byte("MZ but nothing else at all, not a program................"), ico("x"), Info{}); err == nil {
		t.Error("a file that is not a PE image was patched")
	}
}

func TestVersionNumbersTakeTheLeadingDigitsOfEachPart(t *testing.T) {
	if got := versionNumbers("1.2.3-beta"); got != [4]uint32{1, 2, 3, 0} {
		t.Errorf("got %v", got)
	}
	if got := versionNumbers("2025.10.19.7.9"); got != [4]uint32{2025, 10, 19, 7} {
		t.Errorf("got %v", got)
	}
}

func TestPatchRejectsWhatItCannotWrite(t *testing.T) {
	if _, err := Patch(program(), []byte("not an icon"), Info{}); err == nil {
		t.Error("garbage was taken for an .ico")
	}
	if _, err := Patch(program(), nil, Info{}); err == nil {
		t.Error("a patch with nothing in it was written")
	}
}
