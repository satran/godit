package main

import (
	"bytes"
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

const (
	arrowLeft = iota + 1000
	arrowRight
	arrowUp
	arrowDown
)

func main() {
	oldTermios := enableRawMode()
	defer disableRawMode(oldTermios)
	initEditor()
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

func initEditor() error {
	r, c, err := getWindowSize()
	if err != nil {
		return err
	}
	e.screenrows = r
	e.screencols = c
	return nil
}

func editorRefreshScreen() {
	// hide cursor while repainting
	io.WriteString(abuf, "\x1b[?25l")
	// move cursor to top
	io.WriteString(abuf, "\x1b[H")
	editorDrawRows()

	fmt.Fprintf(abuf, "\x1b[%d;%dH", e.cy+1, e.cx+1)
	// show cursor again
	io.WriteString(abuf, "\x1b[?25h")
	io.Copy(os.Stdout, abuf)
	abuf.Reset()
}

func editorProcessKeyPress() error {
	c := editorReadKey()
	switch c {
	case ctrlKey(c):
		write("\x1b[2J")
		write("\x1b[H")
		return errExit
	case arrowUp, arrowDown, arrowLeft, arrowRight:
		editorMoveCursor(c)
	default:
		if unicode.IsControl(rune(c)) {
			write(fmt.Sprintf("^%d\r\n", c))
		} else {
			write(fmt.Sprintf("%c:%d \r\n", c, c))
		}
	}
	return nil
}

func editorReadKey() int {
	var c [3]byte
	_, err := os.Stdin.Read(c[:1])
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}

	if c[0] != '\x1b' {
		return int(c[0])
	}

	if _, err := os.Stdin.Read(c[1:3]); err != nil && err != io.EOF {
		panic(err)
	}
	if c[1] == '[' {
		switch c[2] {
		case 'A':
			return arrowUp
		case 'B':
			return arrowDown
		case 'C':
			return arrowRight
		case 'D':
			return arrowLeft
		}
	}

	return '\x1b'
}

func editorMoveCursor(key int) {
	switch key {
	case arrowLeft:
		if e.cx != 0 {
			e.cx--
		}
	case arrowRight:
		if e.cx != e.screencols-1 {
			e.cx++
		}
	case arrowUp:
		if e.cy != 0 {
			e.cy--
		}
	case arrowDown:
		if e.cy != e.screenrows-1 {
			e.cy++
		}
	}
}

func ctrlKey(c int) int {
	return c & 0x1f
}

func editorDrawRows() {
	for i := 0; i < e.screenrows; i++ {
		io.WriteString(abuf, "~")
		// Clear the line
		io.WriteString(abuf, "\x1b[K")
		if i < e.screenrows-1 {
			io.WriteString(abuf, "\r\n")
		}
	}
}

func getWindowSize() (int, int, error) {
	w := struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL,
		os.Stdout.Fd(),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&w)),
	)
	if err != 0 { // type syscall.Errno
		// This is a hack to get the position. We move the
		// cursor all the way to the bottom right corner and
		// find cursor position.
		io.WriteString(os.Stdout, "\x1b[999C\x1b[999B")
		return getCursorPosition()
	}
	return int(w.Row), int(w.Col), nil
}

func getCursorPosition() (int, int, error) {
	write("\x1b[6n")
	var buffer [1]byte
	var buf []byte
	var cc int
	for cc, _ = os.Stdin.Read(buffer[:]); cc == 1; cc, _ = os.Stdin.Read(buffer[:]) {
		if buffer[0] == 'R' {
			break
		}
		buf = append(buf, buffer[0])
	}
	if string(buf[0:2]) != "\x1b[" {
		return 0, 0, errors.New("failed to read rows and cols from tty")
	}
	var rows, cols int
	if n, err := fmt.Sscanf(string(buf[2:]), "%d;%d", rows, cols); n != 2 || err != nil {
		if err != nil {
			return 0, 0, fmt.Errorf("getCursorPosition: fmt.Sscanf() failed: %s\n", err)
		}
		if n != 2 {
			return 0, 0, fmt.Errorf("getCursorPosition: got %d items, wanted 2\n", n)
		}
		return 0, 0, errors.New("unknown error")
	}
	return rows, cols, nil
}

func write(in string) error {
	_, err := io.WriteString(os.Stdout, in)
	return err
}

var e = struct {
	origTermios            *syscall.Termios
	screenrows, screencols int
	cx, cy                 int
}{}

var abuf = bytes.NewBuffer(nil)
