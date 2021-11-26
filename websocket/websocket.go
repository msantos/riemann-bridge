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
package websocket

import (
	"fmt"
	"os"

	"github.com/gorilla/websocket"
	"github.com/msantos/riemann-bridge/pipe"
)

type IO struct {
	BufferSize int
	URL        string
	Number     int
	Verbose    int
}

func (ws *IO) In() *pipe.Pipe {
	if ws.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "ws: connecting to %s\n", ws.URL)
	}

	ch := make(chan []byte, ws.BufferSize)

	c, _, err := websocket.DefaultDialer.Dial(ws.URL, nil)
	if err != nil {
		return &pipe.Pipe{Err: err}
	}

	n := ws.Number

	go func() {
		defer c.Close()
		defer close(ch)

		for {
			if n == 0 {
				return
			}

			_, event, err := c.ReadMessage()
			if err != nil {
				return
			}

			n--
			if !pipe.Send(ch, event) && ws.Verbose > 0 {
				fmt.Fprintf(os.Stderr, "dropping event:%s\n", event)
			}
		}
	}()

	return &pipe.Pipe{In: ch}
}

func (ws *IO) Out(p *pipe.Pipe) error {
	if p.Err != nil {
		return p.Err
	}

	if ws.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "ws: connecting to %s\n", ws.URL)
	}

	c, _, err := websocket.DefaultDialer.Dial(ws.URL, nil)
	if err != nil {
		return err
	}

	defer c.Close()

	for event := range p.In {
		if err := c.WriteMessage(websocket.TextMessage, event); err != nil {
			return err
		}
	}

	return nil
}
