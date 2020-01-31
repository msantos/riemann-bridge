package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"

	"github.com/gorilla/websocket"
)

// argvT : command line arguments
type argvT struct {
	query   string
	src     *url.URL
	dst     *url.URL
	number  int
	verbose int
}

const (
	version = "0.3.0"
)

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
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
		`not (service ~= "^riemann" or state = "expired")`,
		"Riemann query",
	)

	number := flag.Int("number", -1,
		"Forward the first *number* messages and exit")

	verbose := flag.Int("verbose", 0, "Enable debug messages")

	flag.Parse()

	if flag.NArg() > 0 {
		dstStr = flag.Arg(0)
	}

	src, err := url.Parse(*srcStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	src.RawQuery = "subscribe=true&query=" + url.QueryEscape(*query)

	dst, err := url.Parse(dstStr)
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
	}
}

func main() {
	argv := args()

	stdout := log.New(os.Stdout, "", 0)
	stderr := log.New(os.Stderr, "", 0)

	if argv.verbose > 0 {
		stderr.Printf("dst: connecting to %s", argv.dst.String())
	}
	dst, _, err := websocket.DefaultDialer.Dial(argv.dst.String(), nil)
	if err != nil {
		stderr.Fatalf("dst: %s: %s", argv.dst.String(), err)
	}
	defer dst.Close()

	if argv.verbose > 0 {
		stderr.Printf("src: connecting to %s", argv.src.String())
	}
	src, _, err := websocket.DefaultDialer.Dial(argv.src.String(), nil)
	if err != nil {
		stderr.Fatalf("src: %s: %s", argv.dst.String(), err)
	}
	defer src.Close()

	n := argv.number
	for {
		_, message, err := src.ReadMessage()
		if err != nil {
			stderr.Println("read:", err)
			os.Exit(111)
		}
		if argv.verbose > 1 {
			stdout.Printf("recv: %s", message)
		}
		if argv.number > -1 {
			n--
			if n <= 0 {
				os.Exit(0)
			}
		}
		if err := dst.WriteMessage(websocket.TextMessage, message); err != nil {
			stderr.Println("write:", err)
			os.Exit(111)
		}
	}
}
