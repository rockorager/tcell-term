package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const tiocgwinsz = 0x5413

func ioctl(fd, op, arg uintptr) error {
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, fd, op, arg)
	if ep != 0 {
		return syscall.Errno(ep)
	}
	return nil
}

type WinSize struct {
	Rows   int16 /* rows, in characters */
	Cols   int16 /* columns, in characters */
	XPixel int16 /* horizontal size, pixels */
	YPixel int16 /* vertical size, pixels */
}

func GetWinSize() (sz WinSize, err error) {
	//TIOCGWINSZ syscall
	for fd := uintptr(0); fd < 3; fd++ {
		if err = ioctl(fd, tiocgwinsz, uintptr(unsafe.Pointer(&sz))); err == nil && sz.XPixel != 0 && sz.YPixel != 0 {
			return
		}
	}
	//if pixels are 0, try CSI
	if sz.XPixel == 0 || sz.YPixel == 0 {
		fmt.Printf("\033[18t")
		fmt.Scanf("\033[%d;%dR", &sz.Rows, &sz.Cols)
		//get terminal resolution
		fmt.Printf("\033[14t")
		fmt.Scanf("\033[4;%d;%dt", &sz.YPixel, &sz.XPixel)
	}
	return
}
