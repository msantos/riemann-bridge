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
	query      string
	src        *url.URL
	dst        *url.URL
	bufferSize int
	number     int
	verbose    int
	stdout     *log.Logger
	stderr     *log.Logger
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
		query:      *query,
		src:        src,
		dst:        dst,
		bufferSize: *bufferSize,
		number:     *number,
		verbose:    *verbose,
		stdout:     log.New(os.Stdout, "", 0),
		stderr:     log.New(os.Stderr, "", 0),
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
			stdout(argv, dch, edch)
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
			stdin(argv, sch, esch)
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

	if err := eventLoop(argv, sch, dch, esch, edch); err != nil {
		argv.stderr.Fatalf("%s\n", err)
	}
}

func eventLoop(argv *argvT, sch <-chan []byte, dch chan<- []byte,
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
						argv.stderr.Printf("dropping event:%s\n", ev)
					}
					continue
				}
			} else {
				dch <- ev
			}
		}
	}
}

func stdin(argv *argvT, evch chan<- []byte, errch chan<- error) {
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
	errch <- errEOF
}

func stdout(argv *argvT, evch <-chan []byte, errch chan<- error) {
	for {
		ev := <-evch
		_, err := fmt.Printf("%s\n", ev)
		if err != nil {
			errch <- err
		}
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
