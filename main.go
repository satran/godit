package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"unicode"
	"unsafe"
)

var errExit = errors.New("exit")

func main() {
	oldTermios := enableRawMode()
	defer disableRawMode(oldTermios)
	for {
		editorRefreshScreen()
		if err := editorProcessKeyPress(); err != nil {
			if err == errExit {
				return
			}
			log.Println(err)
			return
		}
	}
}

func enableRawMode() *syscall.Termios {
	origTermios := tcGetAttr(os.Stdin.Fd())
	var raw syscall.Termios
	raw = *origTermios

	// IXON disables ^s ^q
	// ICRNL disables ^m to return enter
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK |
		syscall.ISTRIP | syscall.IXON

	// disable carriage returns
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8

	// ECHO is to ensure characters are not echoed to the prompt
	// ICANON turns of canonical mode
	// ISIG is to ensure SIGINT SIGSTOP is ignored when pressing ^c ^d
	// IEXTEN disables terminal to wait for input after pressing a ctrl key.
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN |
		syscall.ISIG

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

func editorRefreshScreen() {
	// Clear screen
	os.Stdout.Write([]byte("\x1b[2J"))
	// move cursor to top
	os.Stdout.Write([]byte("\x1b[H"))
}

func editorProcessKeyPress() error {
	c := editorReadKey()
	switch c {
	case ctrlKey(c):
		os.Stdout.Write([]byte("\x1b[2J"))
		os.Stdout.Write([]byte("\x1b[H"))
		return errExit
	default:
		if unicode.IsControl(rune(c)) {
			fmt.Printf("^%d\r\n", c)
		} else {
			fmt.Printf("%c:%d \r\n", c, c)
		}
	}
	return nil
}

func editorReadKey() uint8 {
	var d [1]byte
	_, err := os.Stdin.Read(d[:])
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	return d[0]
}

func ctrlKey(c uint8) uint8 {
	return c & 0x1f
}
