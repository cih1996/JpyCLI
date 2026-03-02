//go:build !windows

package server

import (
	"io"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
)

func getUserShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "/bin/bash"
	}
	return shell
}

func StartPTY(cmd *exec.Cmd) (io.ReadWriteCloser, error) {
	return pty.Start(cmd)
}

func ResizePTY(ptyIO io.ReadWriteCloser, w, h int) {
	if f, ok := ptyIO.(*os.File); ok {
		syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
			uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
	}
}
