package windows

import _ "embed"

//go:embed shaders/rectangleVertex.dxbc
var rectangleVertexCode []byte

//go:embed shaders/rectangleFragment.dxbc
var rectangleFragmentCode []byte

//go:embed shaders/textVertex.dxbc
var textVertexCode []byte

//go:embed shaders/textFragment.dxbc
var textFragmentCode []byte
