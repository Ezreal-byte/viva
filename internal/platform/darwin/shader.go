//go:build darwin

package darwin

import _ "embed"

//go:embed shaders/scene.metallib
var sceneShaderLibrary []byte
