// Command unigen reads the Unicode Character Database and writes the property tables
// internal/uniprops uses for line breaking (UAX #14), bidirectional text (UAX #9),
// grapheme clusters (UAX #29) and Arabic joining.
//
//	go run ./tools/unigen -ucd /path/to/ucd -out internal/uniprops/tables.go
//
// The directory holds the files of https://www.unicode.org/Public/UCD/latest/ucd/ under
// their own names: LineBreak.txt, EastAsianWidth.txt, BidiBrackets.txt, BidiMirroring.txt,
// DerivedBidiClass.txt, DerivedGeneralCategory.txt, DerivedJoiningType.txt,
// DerivedCoreProperties.txt, GraphemeBreakProperty.txt and emoji-data.txt. Every
// property is stored as a two stage table (a block index and deduplicated blocks of 128
// code points, one byte each) so a lookup is two array reads.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const max = 0x110000

type table []uint8

func newTable(def uint8) table {
	t := make(table, max)
	for i := range t {
		t[i] = def
	}
	return t
}

func (t table) set(lo, hi int, v uint8) {
	for i := lo; i <= hi; i++ {
		t[i] = v
	}
}

type entry struct {
	lo, hi int
	value  string
	line   string
}

var rangeRE = regexp.MustCompile(`^([0-9A-F]+)(?:\.\.([0-9A-F]+))?$`)

// parse reads "range ; fields... # comment" lines (and "# @missing:" lines when missing
// is true) from path.
func parse(path string, missing bool) (data, defaults []entry) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		isMissing := false
		if missing && strings.HasPrefix(line, "# @missing:") {
			line = strings.TrimPrefix(line, "# @missing:")
			isMissing = true
		} else if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Split(line, ";")
		if len(fields) < 2 {
			continue
		}
		m := rangeRE.FindStringSubmatch(strings.TrimSpace(fields[0]))
		if m == nil {
			continue
		}
		lo, _ := strconv.ParseInt(m[1], 16, 32)
		hi := lo
		if m[2] != "" {
			hi, _ = strconv.ParseInt(m[2], 16, 32)
		}
		e := entry{lo: int(lo), hi: int(hi), value: strings.TrimSpace(fields[len(fields)-1]), line: line}
		if len(fields) > 2 && !isMissing {
			e.value = strings.TrimSpace(fields[1])
			if strings.TrimSpace(fields[1]) == "InCB" {
				e.value = strings.TrimSpace(fields[2])
			}
		}
		if isMissing {
			defaults = append(defaults, e)
		} else {
			data = append(data, e)
		}
	}
	return
}

type gen struct {
	out bytes.Buffer
}

// combined packs the properties of every code point into one tuple, numbers the distinct
// tuples and emits a two stage table of tuple numbers for code points below limit, plus
// the tuple of everything above it (except the tag block, which has a block of its own).
func (g *gen) combined(tables ...table) {
	const shift = 7
	const limit = 0x40000
	const tagFirst, tagEnd = 0xE0000, 0xE1000
	type tuple [5]uint8
	ids := map[tuple]int{}
	var list []tuple
	id := func(r int) int {
		var k tuple
		for i, t := range tables {
			k[i] = t[r]
		}
		n, ok := ids[k]
		if !ok {
			n = len(list)
			ids[k] = n
			list = append(list, k)
		}
		return n
	}
	other := id(0x50000)
	for r := limit; r < 0xF0000; r++ { // the private use planes differ from the unassigned ones only in the Cn flag
		if r >= tagFirst && r < tagEnd || r&0xFFFE == 0xFFFE { // tags and noncharacters
			continue
		}
		if id(r) != other {
			panic(fmt.Sprintf("U+%04X differs from the default above the third plane", r))
		}
	}
	if len(list) > 256 {
		panic("more than 256 distinct property tuples")
	}
	seen := map[string]int{}
	var blocks []byte
	var index []uint16
	for b := 0; b < limit>>shift; b++ {
		block := make([]byte, 1<<shift)
		for i := range block {
			block[i] = byte(id(b<<shift + i))
		}
		n, ok := seen[string(block)]
		if !ok {
			n = len(seen)
			seen[string(block)] = n
			blocks = append(blocks, block...)
		}
		index = append(index, uint16(n))
	}
	fmt.Fprintf(&g.out, "// otherTuple is the tuple of the code points above the third plane that are not tags.\nconst otherTuple = %d\n\n", other)
	fmt.Fprintf(&g.out, "// tagTuples is the tuple of each code point of U+E0000 to U+E0FFF.\nvar tagTuples = [%d]uint8{", tagEnd-tagFirst)
	for r := tagFirst; r < tagEnd; r++ {
		if (r-tagFirst)%32 == 0 {
			g.out.WriteString("\n\t")
		} else {
			g.out.WriteByte(' ')
		}
		fmt.Fprintf(&g.out, "%d,", id(r))
	}
	g.out.WriteString("\n}\n\n")
	fmt.Fprintf(&g.out, "// tuples are the distinct combinations of line break class, bidi class, grapheme class,\n// flags and joining type; the table below numbers them for every code point.\nvar tuples = [%d][5]uint8{\n", len(list))
	for _, k := range list {
		fmt.Fprintf(&g.out, "\t{%d, %d, %d, %d, %d},\n", k[0], k[1], k[2], k[3], k[4])
	}
	g.out.WriteString("}\n\n")
	fmt.Fprintf(&g.out, "// tupleIndex maps code point>>%d to a block of tupleBlocks.\nvar tupleIndex = [%d]uint16{", shift, len(index))
	for i, v := range index {
		if i%24 == 0 {
			g.out.WriteString("\n\t")
		} else {
			g.out.WriteByte(' ')
		}
		fmt.Fprintf(&g.out, "%d,", v)
	}
	fmt.Fprintf(&g.out, "\n}\n\n// tupleBlocks holds %d deduplicated blocks of %d tuple numbers.\nvar tupleBlocks = [%d]uint8{", len(seen), 1<<shift, len(blocks))
	for i, v := range blocks {
		if i%32 == 0 {
			g.out.WriteString("\n\t")
		} else {
			g.out.WriteByte(' ')
		}
		fmt.Fprintf(&g.out, "%d,", v)
	}
	g.out.WriteString("\n}\n\n")
}

// enum emits a type with constants for names, in the order given, and returns the
// value of each name.
func (g *gen) enum(typ, prefix, doc string, names []string) map[string]uint8 {
	fmt.Fprintf(&g.out, "// %s %s\ntype %s uint8\n\nconst (\n", typ, doc, typ)
	values := map[string]uint8{}
	for i, n := range names {
		if i == 0 {
			fmt.Fprintf(&g.out, "\t%s%s %s = iota\n", prefix, n, typ)
		} else {
			fmt.Fprintf(&g.out, "\t%s%s\n", prefix, n)
		}
		values[n] = uint8(i)
	}
	g.out.WriteString(")\n\n")
	return values
}

func main() {
	ucd := flag.String("ucd", "", "directory with the Unicode Character Database files")
	out := flag.String("out", "tables.go", "output Go file")
	flag.Parse()
	path := func(n string) string { return filepath.Join(*ucd, n) }
	var g gen

	version := "unknown"
	if f, err := os.ReadFile(path("LineBreak.txt")); err == nil {
		if m := regexp.MustCompile(`LineBreak-([0-9.]+)\.txt`).FindSubmatch(f); m != nil {
			version = string(m[1])
		}
	}

	// Line breaking.
	lbData, _ := parse(path("LineBreak.txt"), false)
	lbSet := map[string]bool{"XX": true}
	for _, e := range lbData {
		lbSet[e.value] = true
	}
	lbNames := sortedKeys(lbSet)
	lbIDs := g.enum("LineBreakClass", "LB", "is the Line_Break property of UAX #14.", lbNames)
	lb := newTable(lbIDs["XX"])
	for _, e := range lbData {
		lb.set(e.lo, e.hi, lbIDs[e.value])
	}

	// Bidi classes.
	bidiLong := map[string]string{
		"Left_To_Right": "L", "Right_To_Left": "R", "Arabic_Letter": "AL", "European_Number": "EN",
		"European_Separator": "ES", "European_Terminator": "ET", "Arabic_Number": "AN", "Common_Separator": "CS",
		"Nonspacing_Mark": "NSM", "Boundary_Neutral": "BN", "Paragraph_Separator": "B", "Segment_Separator": "S",
		"White_Space": "WS", "Other_Neutral": "ON", "Left_To_Right_Embedding": "LRE", "Left_To_Right_Override": "LRO",
		"Right_To_Left_Embedding": "RLE", "Right_To_Left_Override": "RLO", "Pop_Directional_Format": "PDF",
		"Left_To_Right_Isolate": "LRI", "Right_To_Left_Isolate": "RLI", "First_Strong_Isolate": "FSI",
		"Pop_Directional_Isolate": "PDI",
	}
	bidiNames := []string{"L", "R", "AL", "EN", "ES", "ET", "AN", "CS", "NSM", "BN", "B", "S", "WS", "ON", "LRE", "LRO", "RLE", "RLO", "PDF", "LRI", "RLI", "FSI", "PDI"}
	bidiIDs := g.enum("BidiClass", "Bidi", "is the Bidi_Class property of UAX #9.", bidiNames)
	bd, bdMissing := parse(path("DerivedBidiClass.txt"), true)
	bidi := newTable(bidiIDs["L"])
	for _, e := range bdMissing {
		bidi.set(e.lo, e.hi, bidiIDs[bidiLong[e.value]])
	}
	for _, e := range bd {
		bidi.set(e.lo, e.hi, bidiIDs[e.value])
	}

	// Grapheme cluster break, Extended_Pictographic, Indic_Conjunct_Break, East Asian
	// width (wide: F, W), general category flags and Arabic joining, packed into one
	// "flags" byte-table per property.
	gbNames := []string{"Other", "CR", "LF", "Control", "Extend", "ZWJ", "Regional_Indicator", "Prepend", "SpacingMark", "L", "V", "T", "LV", "LVT"}
	gbShort := []string{"Other", "CR", "LF", "Control", "Extend", "ZWJ", "RegionalIndicator", "Prepend", "SpacingMark", "L", "V", "T", "LV", "LVT"}
	gbIDs := g.enum("GraphemeClass", "GB", "is the Grapheme_Cluster_Break property of UAX #29.", gbShort)
	gbd, _ := parse(path("GraphemeBreakProperty.txt"), false)
	gb := newTable(0)
	for i, n := range gbNames {
		for _, e := range gbd {
			if e.value == n {
				gb.set(e.lo, e.hi, uint8(i))
			}
		}
	}
	_ = gbIDs

	// Flags: bit 0 Extended_Pictographic, 1-2 InCB (1 Consonant, 2 Linker, 3 Extend),
	// 3 East Asian Wide or Fullwidth, 4 Pi, 5 Pf, 6 Cn, 7 mark (Mn, Mc, Me).
	flags := newTable(0)
	pict, _ := parse(path("emoji-data.txt"), false)
	for _, e := range pict {
		if e.value == "Extended_Pictographic" {
			for i := e.lo; i <= e.hi; i++ {
				flags[i] |= 1
			}
		}
	}
	incb, _ := parse(path("DerivedCoreProperties.txt"), false)
	for _, e := range incb {
		if !strings.Contains(e.line, "InCB") {
			continue
		}
		v := map[string]uint8{"Consonant": 1, "Linker": 2, "Extend": 3}[e.value]
		if v == 0 {
			continue
		}
		for i := e.lo; i <= e.hi; i++ {
			flags[i] = flags[i]&^6 | v<<1
		}
	}
	eaw, eawMissing := parse(path("EastAsianWidth.txt"), true)
	for _, e := range eawMissing {
		if e.value == "W" || e.value == "F" {
			for i := e.lo; i <= e.hi; i++ {
				flags[i] |= 8
			}
		}
	}
	for _, e := range eaw {
		for i := e.lo; i <= e.hi; i++ {
			if e.value == "W" || e.value == "F" {
				flags[i] |= 8
			} else {
				flags[i] &^= 8
			}
		}
	}
	gcd, _ := parse(path("DerivedGeneralCategory.txt"), false)
	for _, e := range gcd {
		var bit uint8
		switch e.value {
		case "Pi":
			bit = 16
		case "Pf":
			bit = 32
		case "Cn":
			bit = 64
		case "Mn", "Mc", "Me":
			bit = 128
		}
		for i := e.lo; bit != 0 && i <= e.hi; i++ {
			flags[i] |= bit
		}
	}

	// Arabic joining type.
	joinNames := []string{"U", "R", "D", "L", "C", "T"}
	g.enum("JoiningType", "Join", "is the Joining_Type property of Arabic shaping (U non-joining, R right, D dual, L left, C causing, T transparent).", joinNames)
	jd, _ := parse(path("DerivedJoiningType.txt"), false)
	join := newTable(0)
	for _, e := range jd {
		for i, n := range joinNames {
			if e.value == n {
				join.set(e.lo, e.hi, uint8(i))
			}
		}
	}

	g.combined(lb, bidi, gb, flags, join)

	// Bracket pairs and mirrored glyphs.
	g.out.WriteString("// Brackets lists the paired brackets of UAX #9 (BidiBrackets.txt): the bracket, its\n// counterpart and whether it opens, sorted by bracket.\nvar brackets = [...]Bracket{\n")
	bb, _ := parse(path("BidiBrackets.txt"), false)
	sort.Slice(bb, func(i, j int) bool { return bb[i].lo < bb[j].lo })
	for _, e := range bb {
		fields := strings.Split(e.line, ";")
		pair, _ := strconv.ParseInt(strings.TrimSpace(fields[1]), 16, 32)
		fmt.Fprintf(&g.out, "\t{0x%04X, 0x%04X, %v},\n", e.lo, pair, strings.TrimSpace(fields[2]) == "o")
	}
	g.out.WriteString("}\n\n")
	g.out.WriteString("// mirrors lists Bidi_Mirroring_Glyph pairs (BidiMirroring.txt), sorted by character.\nvar mirrors = [...][2]rune{\n")
	mm, _ := parse(path("BidiMirroring.txt"), false)
	sort.Slice(mm, func(i, j int) bool { return mm[i].lo < mm[j].lo })
	for _, e := range mm {
		v, _ := strconv.ParseInt(e.value, 16, 32)
		fmt.Fprintf(&g.out, "\t{0x%04X, 0x%04X},\n", e.lo, v)
	}
	g.out.WriteString("}\n")

	header := fmt.Sprintf("// Code generated by tools/unigen from the Unicode Character Database %s; DO NOT EDIT.\n\npackage uniprops\n\n// Version is the Unicode version of the tables.\nconst Version = %q\n\n", version, version)
	src, err := format.Source(append([]byte(header), g.out.Bytes()...))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, src, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
