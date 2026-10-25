//go:build darwin || (js && wasm)

package goldens

// DrawList is a serialized, platform-neutral painting instruction stream. Two
// backends rendering the same component tree must produce equal draw lists
// (geometry, colors, order); rasterization may differ, commands may not.
type DrawList struct {
	Commands []Command
}

// Command is one painting instruction in deterministic order.
type Command struct {
	Kind   uint8 // 1=rectangle 2=text 3=shadow
	X      float32
	Y      float32
	Width  float32
	Height float32
	Radius float32
	Red    float32
	Green  float32
	Blue   float32
	Alpha  float32
	Value  string
	// Offset, Blur and Spread place and blur a shadow (Kind 3), cast by the rounded
	// rectangle X, Y, Width, Height, Radius.
	OffsetX float32 `json:",omitempty"`
	OffsetY float32 `json:",omitempty"`
	Blur    float32 `json:",omitempty"`
	Spread  float32 `json:",omitempty"`
}
