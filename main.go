// MIT License
//
// Copyright (c) 2020 Michael Santos
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
	"net/url"
	"os"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

// argvT : command line arguments
type argvT struct {
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

var (
	errEOF = fmt.Errorf("EOF")
)

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}

func urlFromArg(arg, query string) (*url.URL, error) {
	if arg != "-" {
		u, err := url.Parse(arg)
		if err != nil {
			return nil, err
		}
		if query != "" {
			u.RawQuery = query
		}
		return u, nil
	}
	return nil, nil
}

func args() *argvT {
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

	return &argvT{
		query:      *query,
		src:        src,
		dst:        dst,
		bufferSize: *bufferSize,
		number:     *number,
		verbose:    *verbose,
	}
}

func main() {
	argv := args()

	dch := make(chan []byte)
	if argv.bufferSize > 0 {
		dch = make(chan []byte, argv.bufferSize)
	}

	sch := make(chan []byte)
	edch := make(chan error)
	esch := make(chan error)

	go func() {
		if argv.dst == nil {
			argv.stdout(dch, edch)
		} else {
			argv.ws(argv.dst.String(), edch, func(s *websocket.Conn) error {
				ev := <-dch
				if err := s.WriteMessage(websocket.TextMessage, ev); err != nil {
					return err
				}
				return nil
			})
		}
	}()

	go func() {
		if argv.src == nil {
			argv.stdin(sch, esch)
		} else {
			argv.ws(argv.src.String(), esch, func(s *websocket.Conn) error {
				_, message, err := s.ReadMessage()
				if err != nil {
					return err
				}
				sch <- message
				return nil
			})
		}
	}()

	if err := argv.eventLoop(sch, dch, esch, edch); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(111)
	}
}

func (argv *argvT) eventLoop(sch <-chan []byte, dch chan<- []byte,
	esch, edch <-chan error) error {
	n := argv.number

	for {
		select {
		case err := <-edch:
			return err
		case err := <-esch:
			if err == errEOF {
				return nil
			}
			return err
		case ev := <-sch:
			if argv.number > -1 {
				n--
				if n < 0 {
					return nil
				}
			}
			if argv.bufferSize > 0 {
				select {
				case dch <- ev:
				default:
					if argv.verbose > 0 {
						fmt.Fprintf(os.Stderr, "dropping event:%s\n", ev)
					}
					continue
				}
			} else {
				dch <- ev
			}
		}
	}
}

func (argv *argvT) stdin(evch chan<- []byte, errch chan<- error) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		in := scanner.Bytes()
		if bytes.TrimSpace(in) == nil {
			continue
		}
		m := make(map[string]interface{})
		if err := json.Unmarshal(in, &m); err != nil {
			if argv.verbose > 0 {
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
	errch <- errEOF
}

func (argv *argvT) stdout(evch <-chan []byte, errch chan<- error) {
	for {
		ev := <-evch
		_, err := fmt.Printf("%s\n", ev)
		if err != nil {
			errch <- err
		}
	}
}

func (argv *argvT) ws(url string, errch chan<- error,
	ev func(*websocket.Conn) error) {
	if argv.verbose > 0 {
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
