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
	verbose int
}

const (
	version = "0.1.0"
)

func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return def
}

func args() *argvT {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, `%s v%s
Usage: %s [<option>] <query>

`, path.Base(os.Args[0]), version, os.Args[0])
		flag.PrintDefaults()
	}

	srcStr := flag.String(
		"src",
		getenv("RIEMANN_BRIDGE_SRC", "ws://127.0.0.1:5557/index"),
		"Source Riemann server ipaddr:port",
	)
	dstStr := flag.String(
		"dst",
		getenv("RIEMANN_BRIDGE_DST", "ws://127.0.0.1:6557/events"),
		"Destination Riemann server ipaddr:port",
	)

	verbose := flag.Int("verbose", 0, "Enable debug messages")

	flag.Parse()

	query := `not ( service ~= "^riemann" and state = "expired" )`
	if flag.NArg() > 0 {
		query = flag.Arg(0)
	}

	src, err := url.Parse(*srcStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	src.RawQuery = "subscribe=true&query=" + url.QueryEscape(query)

	dst, err := url.Parse(*dstStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid url: %v\n", err)
		os.Exit(1)
	}

	return &argvT{
		query:   query,
		src:     src,
		dst:     dst,
		verbose: *verbose,
	}
}

func main() {
	argv := args()

	stdout := log.New(os.Stdout, "", 0)
	stderr := log.New(os.Stderr, "", 0)

	stderr.Printf("connecting to %s", argv.dst.String())
	dst, _, err := websocket.DefaultDialer.Dial(argv.dst.String(), nil)
	if err != nil {
		stderr.Fatal("dial:", err)
	}
	defer dst.Close()

	stderr.Printf("connecting to %s", argv.src.String())
	src, _, err := websocket.DefaultDialer.Dial(argv.src.String(), nil)
	if err != nil {
		stderr.Fatal("dial:", err)
	}
	defer src.Close()

	for {
		_, message, err := src.ReadMessage()
		if err != nil {
			stderr.Println("read:", err)
			os.Exit(111)
		}
		if argv.verbose == 1 {
			stdout.Printf("recv: %s", message)
		}
		if err := dst.WriteMessage(websocket.TextMessage, message); err != nil {
			stderr.Println("write:", err)
			os.Exit(111)
		}
	}
}
