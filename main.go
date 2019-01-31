package main

import (
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"syscall"
)

var errExit = errors.New("exit")
var debug *log.Logger

const (
	arrowLeft = iota + 1000
	arrowRight
	arrowUp
	arrowDown
	pageUp
	pageDown
	homeKey
	endKey
	deleteKey
)

func main() {
	dbg := flag.Bool("debug", true, "write debug logs to debug.log")
	flag.Parse()
	var w io.Writer
	if *dbg {
		f, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			log.Printf("can't open log file:%s", err)
			return
		}
		defer f.Close()
		w = f
	} else {
		w = ioutil.Discard
	}
	debug = log.New(w, "", log.Lshortfile)

	oldTermios := enableRawMode()
	defer disableRawMode(oldTermios)
	e, err := newEditor()
	if err != nil {
		fatal(oldTermios, "creating editor: %s", err)
	}

	name := flag.Args()[0]
	b, err := newBuffer(name)
	if err != nil {
		fatal(oldTermios, "can't open file: %s", err)
	}
	e.SetBuffer(name, b)
	e.Render()
	for {
		if err := e.ProcessKeyPress(); err != nil {
			if err == errExit {
				return
			}
			log.Println(err)
			return
		}
	}
}

func fatal(t *syscall.Termios, format string, content ...interface{}) {
	disableRawMode(t)
	log.Fatalf(format, content...)
}
