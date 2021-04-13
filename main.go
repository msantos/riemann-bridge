// MIT License
//
// Copyright (c) 2020-2021 Michael Santos
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

type stateT struct {
	query      string
	src        *url.URL
	dst        *url.URL
	bufferSize int
	number     int
	verbose    int
}

const (
	version = "0.4.0"
)

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}

func urlFromArg(arg, query string) (*url.URL, error) {
	if arg == "-" {
		return nil, nil
	}
	u, err := url.Parse(arg)
	if err != nil {
		return nil, err
	}
	u.RawQuery = query
	return u, nil
}

func args() *stateT {
	dstStr := getenv("RIEMANN_BRIDGE_DST", "ws://127.0.0.1:6556/events")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, `%s v%s
Usage: %s [<option>] <destination (default %s)>

`, path.Base(os.Args[0]), version, os.Args[0], dstStr)
		flag.PrintDefaults()
	}

	srcStr := flag.String(
		"src",
		getenv("RIEMANN_BRIDGE_SRC", "ws://127.0.0.1:5556/index"),
		"Source Riemann server ipaddr:port",
	)
	query := flag.String(
		"query",
		getenv("RIEMANN_BRIDGE_QUERY",
			`not (service ~= "^riemann" or state = "expired")`),
		"Riemann query",
	)

	bufferSize := flag.Int("buffer-size", 0,
		"Buffer size (0 to disable)")

	number := flag.Int("number", -1,
		"Forward the first *number* messages and exit")

	verbose := flag.Int("verbose", 0, "Enable debug messages")

	flag.Parse()

	if flag.NArg() > 0 {
		dstStr = flag.Arg(0)
	}

	src, err := urlFromArg(*srcStr,
		"subscribe=true&query="+url.QueryEscape(*query))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	dst, err := urlFromArg(dstStr, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	return &stateT{
		query:      *query,
		src:        src,
		dst:        dst,
		bufferSize: *bufferSize,
		number:     *number,
		verbose:    *verbose,
	}
}

func main() {
	state := args()

	wch := make(chan []byte)
	if state.bufferSize > 0 {
		wch = make(chan []byte, state.bufferSize)
	}

	rch := make(chan []byte)
	errch := make(chan error)

	go func() {
		if state.dst == nil {
			state.stdout(wch, errch)
		} else {
			state.ws(state.dst.String(), errch, func(s *websocket.Conn) error {
				ev := <-wch
				return s.WriteMessage(websocket.TextMessage, ev)
			})
		}
	}()

	go func() {
		if state.src == nil {
			state.stdin(rch, errch)
		} else {
			state.ws(state.src.String(), errch, func(s *websocket.Conn) error {
				_, message, err := s.ReadMessage()
				if err != nil {
					return err
				}
				rch <- message
				return nil
			})
		}
	}()

	if err := state.eventLoop(rch, wch, errch); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(111)
	}
}

func (state *stateT) eventLoop(rch <-chan []byte, wch chan<- []byte,
	errch <-chan error) error {
	n := state.number

	for {
		select {
		case err := <-errch:
			if err == io.EOF {
				return nil
			}
			return err
		case ev := <-rch:
			if state.number > -1 {
				n--
				if n < 0 {
					return nil
				}
			}
			if state.bufferSize > 0 {
				select {
				case wch <- ev:
				default:
					if state.verbose > 0 {
						fmt.Fprintf(os.Stderr, "dropping event:%s\n", ev)
					}
					continue
				}
			} else {
				wch <- ev
			}
		}
	}
}

func (state *stateT) stdin(evch chan<- []byte, errch chan<- error) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		in := scanner.Bytes()
		if bytes.TrimSpace(in) == nil {
			continue
		}
		m := make(map[string]interface{})
		if err := json.Unmarshal(in, &m); err != nil {
			if state.verbose > 0 {
				fmt.Fprintln(os.Stderr, err)
			}
			continue
		}
		if m["time"] == nil {
			m["time"] = time.Now()
		}
		out, err := json.Marshal(m)
		if err != nil {
			errch <- err
			return
		}
		evch <- out
	}
	errch <- io.EOF
}

func (state *stateT) stdout(evch <-chan []byte, errch chan<- error) {
	for {
		ev := <-evch
		_, err := fmt.Printf("%s\n", ev)
		if err != nil {
			errch <- err
			return
		}
	}
}

func (state *stateT) ws(url string, errch chan<- error,
	ev func(*websocket.Conn) error) {
	if state.verbose > 0 {
		fmt.Fprintf(os.Stderr, "ws: connecting to %s", url)
	}
	s, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		errch <- err
		return
	}
	defer s.Close()

	for {
		if err := ev(s); err != nil {
			errch <- err
			return
		}
	}
}
