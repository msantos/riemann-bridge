// MIT License
//
// # Copyright (c) 2020-2023 Michael Santos
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
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/msantos/riemann-bridge/pipe"
	"github.com/msantos/riemann-bridge/sse"
	"github.com/msantos/riemann-bridge/stdio"
	"github.com/msantos/riemann-bridge/websocket"
)

type stateT struct {
	query      string
	src        string
	dst        string
	bufferSize int
	number     uint64
	verbose    int
}

const (
	version = "2.0.0"
)

var errUnsupportedProtocol = errors.New("unsupported protocol")

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
	src := getenv("RIEMANN_BRIDGE_SRC", "-")
	dst := getenv("RIEMANN_BRIDGE_DST", "-")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, `%s v%s
Usage: %s [<option>] <source (default '%s')> <destination (default '%s')

`, path.Base(os.Args[0]), version, os.Args[0], src, dst)
		flag.PrintDefaults()
	}

	query := flag.String(
		"query",
		getenv("RIEMANN_BRIDGE_QUERY", `not (service ~= "^riemann" or state = "expired")`),
		"Riemann query",
	)

	bufferSize := flag.Uint("buffer-size", 0,
		"Drop any events exceeding the buffer size (0 (unbuffered))")

	number := flag.Uint("number", 0,
		"Forward the first *number* messages and exit")

	verbose := flag.Int("verbose", 0, "Enable debug messages")

	flag.Parse()

	switch flag.NArg() {
	case 2:
		dst = flag.Arg(1)
		fallthrough
	case 1:
		src = flag.Arg(0)
	default:
	}

	return &stateT{
		query:      *query,
		src:        src,
		dst:        dst,
		bufferSize: int(*bufferSize),
		number:     uint64(*number),
		verbose:    *verbose,
	}
}

func main() {
	state := args()

	stdin, err := state.In()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	stdout, err := state.Out()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	p := pipe.New(state.bufferSize, state.number)

	if err := stdout.Out(stdin.In(p)); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(111)
	}
}

func (state *stateT) In() (pipe.Piper, error) {
	switch {
	case state.src == "-":
		return &stdio.IO{
			Verbose: state.verbose,
		}, nil
	case strings.HasPrefix(state.src, "ws"):
		query, err := queryURL(
			state.src,
			"subscribe=true&query="+url.QueryEscape(state.query),
		)
		if err != nil {
			return nil, err
		}

		return &websocket.IO{
			URL:     query,
			Verbose: state.verbose,
		}, nil
	case strings.HasPrefix(state.src, "http"):
		query, err := queryURL(
			state.src,
			"query="+url.QueryEscape(state.query),
		)
		if err != nil {
			return nil, err
		}

		return &sse.IO{
			URL:     query,
			Verbose: state.verbose,
		}, nil
	default:
		return nil, fmt.Errorf("in: %s: %w", state.src, errUnsupportedProtocol)
	}
}

func (state *stateT) Out() (pipe.Piper, error) {
	switch {
	case state.dst == "-":
		return &stdio.IO{
			Verbose: state.verbose,
		}, nil

	case strings.HasPrefix(state.dst, "ws"):
		query, err := queryURL(state.dst, "")
		if err != nil {
			return nil, err
		}

		return &websocket.IO{
			URL:     query,
			Verbose: state.verbose,
		}, nil
	default:
		return nil, fmt.Errorf("%s: %w", state.dst, errUnsupportedProtocol)
	}
}
