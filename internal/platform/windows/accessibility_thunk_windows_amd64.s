//go:build windows && amd64

#include "textflag.h"

// UI Automation passes ElementProviderFromPoint's coordinates in XMM1 and
// XMM2. syscall.NewCallback expects each argument in its integer-register
// position, so this ABI-only thunk copies the raw bits before tail-calling it.
TEXT ·uiaFromPointThunk(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ X2, R8
	MOVQ ·uiaFromPointCallback(SB), AX
	JMP  AX

TEXT ·uiaFromPointThunkAddress(SB), NOSPLIT, $0-8
	LEAQ ·uiaFromPointThunk(SB), AX
	MOVQ AX, ret+0(FP)
	RET

// SetValue of the range value pattern takes a double, which UI Automation
// passes in XMM1 after the interface pointer. The thunk moves its bits into the
// integer register the callback reads it from.
TEXT ·uiaRangeValueThunk(SB), NOSPLIT|NOFRAME, $0-0
	MOVQ X1, DX
	MOVQ ·uiaRangeValueCallback(SB), AX
	JMP  AX

TEXT ·uiaRangeValueThunkAddress(SB), NOSPLIT, $0-8
	LEAQ ·uiaRangeValueThunk(SB), AX
	MOVQ AX, ret+0(FP)
	RET
