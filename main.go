package main

import (
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	oldTermios := enableRawMode()
	defer disableRawMode(oldTermios)

	for {
		data := []byte{0}
		_, err := os.Stdin.Read(data)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		if string(data) == "q" {
			return
		}
	}
}

func enableRawMode() *syscall.Termios {
	origTermios := tcGetAttr(os.Stdin.Fd())
	var raw syscall.Termios
	raw = *origTermios
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN+1] = 0
	raw.Cc[syscall.VTIME+1] = 1
	if e := tcSetAttr(os.Stdin.Fd(), &raw); e != nil {
		log.Fatalf("Problem enabling raw mode: %s\n", e)
	}
	return origTermios
}

func disableRawMode(t *syscall.Termios) {
	if e := tcSetAttr(os.Stdin.Fd(), t); e != nil {
		log.Fatalf("Problem disabling raw mode: %s\n", e)
	}
}
func tcSetAttr(fd uintptr, termios *syscall.Termios) error {
	// TCSETS+1 == TCSETSW, because TCSAFLUSH doesn't exist
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TCSETS+1), uintptr(unsafe.Pointer(termios))); err != 0 {
		return err
	}
	return nil
}

func tcGetAttr(fd uintptr) *syscall.Termios {
	var termios = &syscall.Termios{}
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(termios))); err != 0 {
		log.Fatalf("Problem getting terminal attributes: %s\n", err)
	}
	return termios
}
