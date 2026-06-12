package mofu

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// enableVTProcessing enables Windows VT100/ANSI escape sequence processing.
// Without this, conhost.exe and older terminals cannot render ANSI colors.
func enableVTProcessing() {
	if runtime.GOOS != "windows" {
		return
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")

	handle := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	getMode.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))

	// ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	// ENABLE_PROCESSED_OUTPUT = 0x0001
	// ENABLE_WRAP_AT_EOL_OUTPUT = 0x0002
	mode |= 0x0007
	setMode.Call(uintptr(handle), uintptr(mode))
}
