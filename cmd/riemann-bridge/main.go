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
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/msantos/riemann-bridge/pipe"
	"github.com/msantos/riemann-bridge/stdio"
	"github.com/msantos/riemann-bridge/websocket"
)

type stateT struct {
	query      string
	src        string
	dst        string
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

func queryURL(arg, query string) (string, error) {
	u, err := url.Parse(arg)
	if err != nil {
		return "", err
	}
	u.RawQuery = query
	return u.String(), nil
}

func args() *stateT {
	dst := getenv("RIEMANN_BRIDGE_DST", "ws://127.0.0.1:6556/events")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, `%s v%s
Usage: %s [<option>] <destination (default %s)>

`, path.Base(os.Args[0]), version, os.Args[0], dst)
		flag.PrintDefaults()
	}

	src := flag.String(
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

	bufferSize := flag.Uint("buffer-size", 0,
		"Drop any events exceeding the buffer size (0 (unbuffered))")

	number := flag.Int("number", -1,
		"Forward the first *number* messages and exit")

	verbose := flag.Int("verbose", 0, "Enable debug messages")

	flag.Parse()

	if flag.NArg() > 0 {
		dst = flag.Arg(0)
	}

	return &stateT{
		query:      *query,
		src:        *src,
		dst:        dst,
		bufferSize: int(*bufferSize),
		number:     *number,
		verbose:    *verbose,
	}
}

func main() {
	state := args()

	stdin, err := state.In()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", state.src, err)
		os.Exit(1)
	}

	stdout, err := state.Out()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", state.dst, err)
		os.Exit(1)
	}

	in, err := stdin.In()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", state.src, err)
		os.Exit(1)
	}

	if err := stdout.Out(in); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(111)
	}
}

func (state *stateT) In() (pipe.Piper, error) {
	if state.src == "-" {
		return &stdio.IO{
			BufferSize: state.bufferSize,
			Number:     state.number,
			Verbose:    state.verbose,
		}, nil
	}

	u, err := queryURL(
		state.src,
		"subscribe=true&query="+url.QueryEscape(state.query),
	)
	if err != nil {
		return nil, err
	}

	return &websocket.IO{
		URL:        u,
		BufferSize: state.bufferSize,
		Number:     state.number,
		Verbose:    state.verbose,
	}, nil
}

func (state *stateT) Out() (pipe.Piper, error) {
	if state.dst == "-" {
		return &stdio.IO{
			Verbose: state.verbose,
		}, nil
	}

	u, err := queryURL(state.dst, "")
	if err != nil {
		return nil, err
	}

	return &websocket.IO{
		URL:     u,
		Verbose: state.verbose,
	}, nil
}