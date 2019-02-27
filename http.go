package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	termbox "github.com/nsf/termbox-go"
)

func (g *godit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/current/buffer":
		g.handleCurrentBuffer(w, r)
	}
}

func (g *godit) handleCurrentBuffer(w http.ResponseWriter, r *http.Request) {
	v := g.active.leaf
	switch r.Method {
	case "GET":
		io.WriteString(w, v.buf.path)
		return
	case "POST":
		by, _ := ioutil.ReadAll(r.Body)
		line := strings.Split(string(by), "\n")[0]
		chunks := strings.Split(line, ":")
		if _, err := os.Stat(chunks[0]); os.IsNotExist(err) {
			return
		}
		g.open_buffers_from_pattern(chunks[0])
		num := 1
		if len(chunks) > 1 {
			num, _ = strconv.Atoi(chunks[1])
		}
		v.on_vcommand(vcommand_move_cursor_to_line, rune(num))
		v.finalize_action_group()
	}
}
