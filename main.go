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
	// Clear screen
	write("\x1b[2J")
	// move cursor to top
	write("\x1b[H")
	editorDrawRows()
	write("\x1b[H")
}

func editorProcessKeyPress() error {
	c := editorReadKey()
	switch c {
	case ctrlKey(c):
		write("\x1b[2J")
		write("\x1b[H")
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

func editorDrawRows() {
	for i := 0; i < e.screenrows; i++ {
		write("~\r\n")
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
}{}
