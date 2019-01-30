package main

import (
	"bufio"
	"errors"
	"io"
	"os"
)

type Buffer interface {
	Name() string
	Path() string
	Get(c Cursor) ([]byte, error)
	Line(l int) ([]byte, error)
	Insert(c Cursor, content []byte) error
	Delete(c Cursor) error
}

type Cursor struct {
	StartLine, StartColumn int
	EndLine, EndColumn     int
}

func newBuffer(name string) (Buffer, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	var lines [][]byte
	var n int
	for {
		// TODO: I'm ignoring long lines, this could be a problem later on. For now continue.
		line, _, err := r.ReadLine()
		if err != nil && err != io.EOF {
			return nil, err
		}
		n++
		lines = append(lines, line)
		if err == io.EOF {
			break
		}
	}
	return &fileBuffer{
		path:   name,
		lines:  lines,
		nlines: n,
	}, nil
}

type fileBuffer struct {
	path   string
	lines  [][]byte
	nlines int
}

func (f *fileBuffer) Name() string {
	return f.path
}

func (f *fileBuffer) Path() string {
	return f.path
}

func (f *fileBuffer) Get(c Cursor) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (f *fileBuffer) Line(l int) ([]byte, error) {
	l++ // lines are numbered from 1 but our array from 0
	if l > f.nlines {
		return nil, errors.New("line not found")
	}
	return f.lines[l], nil
}

func (f *fileBuffer) Insert(c Cursor, content []byte) error {
	return errors.New("not implemented")
}

func (f *fileBuffer) Delete(c Cursor) error {
	return errors.New("not implemented")
}
