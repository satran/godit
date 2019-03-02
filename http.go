package main

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	termbox "github.com/nsf/termbox-go"
)

func (g *godit) startHTTPServer() error {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	g.asyncFns <- func() {
		g.httpPort = listener.Addr().(*net.TCPAddr).Port
		g.set_status("HTTP port on %d", g.httpPort)
		g.draw()
		termbox.Flush()
	}
	return http.Serve(listener, g)
}

func (g *godit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/buffers/current":
		g.handleCurrentBuffer(w, r)
	}
}

func (g *godit) handleCurrentBuffer(w http.ResponseWriter, r *http.Request) {
	v := g.active.leaf
	switch r.Method {
	case "GET":
		c := make(chan string)
		g.asyncFns <- func() {
			c <- v.buf.path
		}
		path := <-c
		io.WriteString(w, path)
		return
	case "POST":
		by, _ := ioutil.ReadAll(r.Body)
		line := strings.Split(string(by), "\n")[0]
		chunks := strings.Split(line, ":")
		if _, err := os.Stat(chunks[0]); os.IsNotExist(err) {
			return
		}
		g.asyncFns <- func() {
			g.open_buffers_from_pattern(chunks[0])
			num := 1
			if len(chunks) > 1 {
				num, _ = strconv.Atoi(chunks[1])
			}
			v.on_vcommand(vcommand_move_cursor_to_line, rune(num))
			v.finalize_action_group()
		}
	}
}
