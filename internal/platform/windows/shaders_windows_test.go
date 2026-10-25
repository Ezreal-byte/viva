package windows

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"testing"
)

//go:embed shaders/scene.hlsl
var testShaderSource []byte

//go:embed shaders/manifest.json
var testShaderManifest []byte

func TestPrecompiledShaders(t *testing.T) {
	var manifest struct {
		SourceSHA256 string `json:"source_sha256"`
		Compiler     string `json:"compiler"`
		Entries      []struct {
			Name    string `json:"name"`
			Profile string `json:"profile"`
			SHA256  string `json:"sha256"`
			Bytes   int    `json:"bytes"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(testShaderManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(testShaderSource)
	if hex.EncodeToString(digest[:]) != manifest.SourceSHA256 {
		t.Fatal("HLSL source changed: regenerate Windows shaders")
	}
	shaders := [...]struct {
		name, profile string
		code          []byte
	}{
		{"rectangleVertex", "vs_5_0", rectangleVertexCode},
		{"rectangleFragment", "ps_5_0", rectangleFragmentCode},
		{"textVertex", "vs_5_0", textVertexCode},
		{"textFragment", "ps_5_0", textFragmentCode},
	}
	if len(manifest.Entries) != len(shaders) {
		t.Fatal("incomplete shader manifest")
	}
	for index, shader := range shaders {
		entry := manifest.Entries[index]
		digest := sha256.Sum256(shader.code)
		if entry.Name != shader.name || entry.Profile != shader.profile || entry.Bytes != len(shader.code) || entry.SHA256 != hex.EncodeToString(digest[:]) {
			t.Fatalf("%s: bytecode does not match manifest", shader.name)
		}
		if len(shader.code) < 32 || !bytes.Equal(shader.code[:4], []byte("DXBC")) || binary.LittleEndian.Uint32(shader.code[24:28]) != uint32(len(shader.code)) {
			t.Fatalf("%s: invalid DXBC container", shader.name)
		}
	}
}

func TestRectangleShaderAppliesBatchOffset(t *testing.T) {
	if !bytes.Contains(testShaderSource, []byte("rectangles[instanceOffset + instance]")) {
		t.Fatal("rectangle shader ignores the compositing batch's first instance")
	}
}
