package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"unicode"
)

type editor struct {
	screenrows, screencols int
	cx, cy                 int

	bufferLock    sync.Mutex
	buffers       map[string]Buffer
	currentBuffer Buffer
}

func newEditor() (*editor, error) {
	e := &editor{}
	r, c, err := getWindowSize()
	if err != nil {
		return nil, err
	}
	e.screenrows = r
	e.screencols = c
	return e, nil
}

func (e *editor) SetBuffer(name string, b Buffer) {
	e.bufferLock.Lock()
	defer e.bufferLock.Unlock()
	e.currentBuffer = b
	e.buffers[name] = b
}

func (e *editor) Render() {

}

func (e *editor) RefreshScreen() {
	// hide cursor while repainting
	io.WriteString(abuf, "\x1b[?25l")
	// move cursor to top
	io.WriteString(abuf, "\x1b[H")
	e.DrawRows()

	fmt.Fprintf(abuf, "\x1b[%d;%dH", e.cy+1, e.cx+1)
	// show cursor again
	io.WriteString(abuf, "\x1b[?25h")
	io.Copy(os.Stdout, abuf)
	abuf.Reset()
}

func (e *editor) ProcessKeyPress() error {
	c := e.ReadKey()
	switch c {
	case ctrlKey(c):
		write("\x1b[2J")
		write("\x1b[H")
		return errExit
	case arrowUp, arrowDown, arrowLeft, arrowRight:
		e.MoveCursor(c)
	case pageUp, pageDown:
		for i := e.screenrows; i > 0; i-- {
			if c == pageUp {
				e.MoveCursor(arrowUp)
			} else {
				e.MoveCursor(arrowDown)
			}
		}
	case homeKey:
		e.cx = 0
	case endKey:
		e.cx = e.screencols - 1
	default:
		if unicode.IsControl(rune(c)) {
			write(fmt.Sprintf("^%d\r\n", c))
		} else {
			write(fmt.Sprintf("%c:%d \r\n", c, c))
		}
	}
	return nil
}

func (e *editor) ReadKey() int {
	var c [4]byte
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
		if c[2] >= '0' && c[2] <= '9' {
			_, err := os.Stdin.Read(c[3:4])
			if err != nil && err != io.EOF {
				log.Fatal(err)
			}
			if c[3] == '~' {
				switch c[2] {
				case '1':
					return homeKey
				case '3':
					return deleteKey
				case '4':
					return endKey
				case '5':
					return pageUp
				case '6':
					return pageDown
				case '7':
					return homeKey
				case '8':
					return endKey
				}
			}
		} else {
			switch c[2] {
			case 'A':
				return arrowUp
			case 'B':
				return arrowDown
			case 'C':
				return arrowRight
			case 'D':
				return arrowLeft
			case 'H':
				return homeKey
			case 'F':
				return endKey
			}
		}
	} else if c[1] == 'O' {
		switch c[2] {
		case 'H':
			return homeKey
		case 'F':
			return endKey
		}
	}
	return '\x1b'
}

func (e *editor) MoveCursor(key int) {
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

func (e *editor) DrawRows() {
	for y := 0; y < e.screenrows; y++ {
		io.WriteString(abuf, "~")
		// Clear the line
		io.WriteString(abuf, "\x1b[K")
		if y < e.screenrows-1 {
			io.WriteString(abuf, "\r\n")
		}
	}
}

func write(in string) error {
	_, err := io.WriteString(os.Stdout, in)
	return err
}

var abuf = bytes.NewBuffer(nil)
