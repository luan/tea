package utils

import (
	"syscall"
	"unsafe"
)

type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16
	y      uint16
}

func SetWinsize(fd uintptr, ws *Winsize) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
	return err
}
