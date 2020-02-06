package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

// argvT : command line arguments
type argvT struct {
	query   string
	src     *url.URL
	dst     *url.URL
	number  int
	verbose int
	stdout  *log.Logger
	stderr  *log.Logger
}

const (
	version = "0.4.0"
)

var (
	eof = fmt.Errorf("EOF")
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

	number := flag.Int("number", -1,
		"Forward the first *number* messages and exit")

	verbose := flag.Int("verbose", 0, "Enable debug messages")

	flag.Parse()

	if flag.NArg() > 0 {
		dstStr = flag.Arg(0)
	}

	var src *url.URL
	var dst *url.URL
	var err error

	src, err = urlFromArg(*srcStr,
		"subscribe=true&query="+url.QueryEscape(*query))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	dst, err = urlFromArg(dstStr, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	return &argvT{
		query:   *query,
		src:     src,
		dst:     dst,
		number:  *number,
		verbose: *verbose,
		stdout:  log.New(os.Stdout, "", 0),
		stderr:  log.New(os.Stderr, "", 0),
	}
}

func main() {
	argv := args()

	dch := make(chan []byte)
	sch := make(chan []byte)
	edch := make(chan error)
	esch := make(chan error)

	go func() {
		if argv.dst == nil {
			stdout(argv, edch, dch)
		} else {
			ws(argv, argv.dst.String(), edch, func(s *websocket.Conn) error {
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
			stdin(argv, esch, sch)
		} else {
			ws(argv, argv.src.String(), esch, func(s *websocket.Conn) error {
				_, message, err := s.ReadMessage()
				if err != nil {
					return err
				}
				sch <- message
				return nil
			})
		}
	}()

	n := argv.number

loop:
	for {
		select {
		case err := <-edch:
			argv.stderr.Fatalf("%s: %s\n", argv.dst.String(), err)
		case err := <-esch:
			if err != eof {
				argv.stderr.Fatalf("%s: %s\n", argv.src.String(), err)
			}
			break loop
		case ev := <-sch:
			if argv.number > -1 {
				n--
				if n < 0 {
					os.Exit(0)
				}
			}
			dch <- ev
		}
	}
}

func stdin(argv *argvT, errch chan<- error, evch chan<- []byte) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		in := scanner.Bytes()
		if bytes.TrimSpace(in) == nil {
			continue
		}
		m := make(map[string]interface{})
		if err := json.Unmarshal(in, &m); err != nil {
			if argv.verbose > 0 {
				argv.stderr.Println(err)
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
	errch <- eof
}

func stdout(argv *argvT, errch chan<- error, evch <-chan []byte) {
	for {
		ev := <-evch
		fmt.Printf("%s\n", ev)
	}
}

func ws(argv *argvT, url string, errch chan<- error,
	ev func(*websocket.Conn) error) {
	if argv.verbose > 0 {
		argv.stderr.Printf("ws: connecting to %s", url)
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
