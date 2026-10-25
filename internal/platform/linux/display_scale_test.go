package linux

import (
	"encoding/binary"
	"testing"
)

// settingsBlob builds an XSETTINGS property in the byte order given.
func settingsBlob(order binary.ByteOrder, integers map[string]int32, strings map[string]string) []byte {
	pad := func(b []byte) []byte {
		for len(b)%4 != 0 {
			b = append(b, 0)
		}
		return b
	}
	var body []byte
	count := 0
	add := func(kind byte, name string, value []byte) {
		head := []byte{kind, 0, 0, 0}
		order.PutUint16(head[2:], uint16(len(name)))
		body = append(body, head...)
		body = append(body, pad([]byte(name))...)
		body = append(body, 0, 0, 0, 0) // last change serial
		body = append(body, value...)
		count++
	}
	for name, value := range integers {
		v := make([]byte, 4)
		order.PutUint32(v, uint32(value))
		add(0, name, v)
	}
	for name, value := range strings {
		v := make([]byte, 4)
		order.PutUint32(v, uint32(len(value)))
		add(1, name, append(v, pad([]byte(value))...))
	}
	header := make([]byte, 12)
	if order == binary.BigEndian {
		header[0] = 1
	}
	order.PutUint32(header[8:], uint32(count))
	return append(header, body...)
}

func TestChooseScalePrefersTheMostSpecificSource(t *testing.T) {
	gnome, err := parseXSettings(settingsBlob(binary.LittleEndian, map[string]int32{"Gdk/WindowScalingFactor": 2, "Xft/DPI": 96 * 1024}, nil))
	if err != nil {
		t.Fatal(err)
	}
	kde, err := parseXSettings(settingsBlob(binary.LittleEndian, map[string]int32{"Gdk/WindowScalingFactor": 2, "Xft/DPI": 144 * 1024}, map[string]string{"Net/ThemeName": "breeze"}))
	if err != nil {
		t.Fatal(err)
	}
	flat, err := parseXSettings(settingsBlob(binary.BigEndian, map[string]int32{"Xft/DPI": 96 * 1024}, nil))
	if err != nil {
		t.Fatal(err)
	}
	monitor := monitorGeometry{Width: 3840, Height: 2160, WidthMM: 600, HeightMM: 340, valid: true} // 163 dpi
	for _, test := range []struct {
		name   string
		in     scaleInputs
		want   float32
		origin scaleOrigin
	}{
		{"nothing", scaleInputs{}, 1, scaleDefault},
		{"override", scaleInputs{VivaScale: "2", QtScaleFactor: "1.5", GdkScale: "3"}, 2, scaleFromEnvironment},
		{"qt factor beats gdk", scaleInputs{QtScaleFactor: "1.5", GdkScale: "2", XSettings: gnome}, 1.5, scaleFromEnvironment},
		{"qt screens, named", scaleInputs{QtScreenScaleFactors: "DP-1=2;HDMI-1=1.25", Screen: "HDMI-1"}, 1.25, scaleFromEnvironment},
		{"qt screens, unnamed", scaleInputs{QtScreenScaleFactors: "1.75;1"}, 1.75, scaleFromEnvironment},
		{"qt screens, no match uses the first", scaleInputs{QtScreenScaleFactors: "DP-1=2;HDMI-1=1.25", Screen: "eDP-1"}, 2, scaleFromEnvironment},
		{"gdk scale", scaleInputs{GdkScale: "2", XSettings: kde}, 2, scaleFromEnvironment},
		{"garbage falls through", scaleInputs{VivaScale: "x", QtScaleFactor: "-1", GdkScale: "0", XSettings: kde}, 1.5, scaleFromXSettings},
		{"gnome integer scaling", scaleInputs{XSettings: gnome}, 2, scaleFromXSettings},
		{"kde fractional dpi", scaleInputs{XSettings: kde}, 1.5, scaleFromXSettings},
		{"manager at 96 stops the search", scaleInputs{XSettings: flat, ResourceDPI: 192, Monitor: monitor}, 1, scaleFromXSettings},
		{"resource", scaleInputs{ResourceDPI: 192, Monitor: monitor}, 2, scaleFromResource},
		{"randr", scaleInputs{Monitor: monitor}, 1.75, scaleFromRandR},
		{"dpi below 96 never shrinks", scaleInputs{ResourceDPI: 72}, 1, scaleFromResource},
		{"huge values are limited", scaleInputs{VivaScale: "50"}, 8, scaleFromEnvironment},
	} {
		got, origin := chooseScale(test.in)
		if got != test.want || origin != test.origin {
			t.Errorf("%s: scale %v from %v, want %v from %v", test.name, got, origin, test.want, test.origin)
		}
	}
}

func TestSnapScaleRoundsToQuarters(t *testing.T) {
	for _, test := range []struct{ in, want float64 }{
		{1, 1}, {1.1, 1}, {1.13, 1.25}, {1.4, 1.5}, {1.5, 1.5}, {1.7, 1.75}, {2.1, 2}, {2.9, 3}, {0.75, 1}, {100, 8},
	} {
		if got := snapScale(test.in); float64(got) != test.want {
			t.Errorf("snapScale(%v) = %v, want %v", test.in, got, test.want)
		}
	}
}

func TestMonitorScaleTrustsOnlyPlausibleSizes(t *testing.T) {
	for _, test := range []struct {
		name    string
		monitor monitorGeometry
		want    float32
		ok      bool
	}{
		{"1080p 24 inch", monitorGeometry{Width: 1920, Height: 1080, WidthMM: 531, HeightMM: 299, valid: true}, 1, true},
		{"4K 27 inch", monitorGeometry{Width: 3840, Height: 2160, WidthMM: 596, HeightMM: 335, valid: true}, 1.75, true},
		{"4K 32 inch", monitorGeometry{Width: 3840, Height: 2160, WidthMM: 708, HeightMM: 398, valid: true}, 1.5, true},
		{"laptop panel", monitorGeometry{Width: 2880, Height: 1800, WidthMM: 286, HeightMM: 179, valid: true}, 2.75, true},
		{"no size reported", monitorGeometry{Width: 1920, Height: 1080, valid: true}, 0, false},
		{"one millimeter per pixel", monitorGeometry{Width: 1920, Height: 1080, WidthMM: 1, HeightMM: 1, valid: true}, 0, false},
		{"stretched", monitorGeometry{Width: 1920, Height: 1080, WidthMM: 300, HeightMM: 300, valid: true}, 0, false},
		{"unknown monitor", monitorGeometry{}, 0, false},
	} {
		got, ok := monitorScale(test.monitor)
		if ok != test.ok || got != test.want {
			t.Errorf("%s: scale %v, trusted %v; want %v, %v", test.name, got, ok, test.want, test.ok)
		}
	}
}

func TestMonitorAtFollowsTheWindowCenter(t *testing.T) {
	monitors := []monitorGeometry{
		{Name: "left", X: 0, Y: 0, Width: 1920, Height: 1080},
		{Name: "right", X: 1920, Y: 0, Width: 3840, Height: 2160, Primary: true},
	}
	for _, test := range []struct {
		x, y int
		want string
	}{{100, 100, "left"}, {1919, 0, "left"}, {1920, 0, "right"}, {5000, 2000, "right"}, {-50, -50, "right"}} {
		if got, _ := monitorAt(monitors, test.x, test.y); got.Name != test.want {
			t.Errorf("monitorAt(%d, %d) = %s, want %s", test.x, test.y, got.Name, test.want)
		}
	}
}

func TestParseXSettingsRejectsMalformedBlobs(t *testing.T) {
	good := settingsBlob(binary.LittleEndian, map[string]int32{"Xft/DPI": 120 * 1024}, map[string]string{"Net/ThemeName": "Adwaita"})
	parsed, err := parseXSettings(good)
	if err != nil || !parsed.hasDPI || parsed.dpi() != 120 {
		t.Fatalf("parse = %+v, %v; want Xft/DPI 120", parsed, err)
	}
	// Every truncation of a valid blob is an error, never a panic or a partial result.
	for length := range good {
		if _, err := parseXSettings(good[:length]); err == nil {
			t.Errorf("a blob cut to %d of %d bytes was accepted", length, len(good))
		}
	}
	if _, err := parseXSettings([]byte{9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}); err == nil {
		t.Error("an unknown byte order was accepted")
	}
}

func TestResourceDatabase(t *testing.T) {
	database := "Xcursor.size:\t32\nXft.antialias:\t1\nXft.dpi:\t192\nXft.rgba: none\n"
	if got := resourceDPI(database); got != 192 {
		t.Errorf("Xft.dpi = %v, want 192", got)
	}
	if got := resourceCursorSize(database); got != 32 {
		t.Errorf("Xcursor.size = %v, want 32", got)
	}
	if resourceDPI("Xft.dpi: lots") != 0 || resourceDPI("") != 0 {
		t.Error("a missing or non-numeric Xft.dpi was read as a number")
	}
}

func TestDeviceLengthRoundsToTheNearestPixel(t *testing.T) {
	for _, test := range []struct {
		logical int
		scale   float32
		want    uint32
	}{{800, 1, 800}, {800, 2, 1600}, {801, 1.5, 1202}, {1001, 1.25, 1251}, {1, 1.25, 1}, {3, 1.75, 5}} {
		if got := deviceLength(test.logical, test.scale); got != test.want {
			t.Errorf("deviceLength(%d, %v) = %d, want %d", test.logical, test.scale, got, test.want)
		}
	}
}
