// MIT License
//
// Copyright (c) 2021 Michael Santos
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
package sse

import (
	"fmt"
	"os"

	"github.com/donovanhide/eventsource"
	"github.com/msantos/riemann-bridge/pipe"
)

type IO struct {
	BufferSize int
	URL        string
	Number     int
	Verbose    int
}

func (sse *IO) In() *pipe.Pipe {
	if sse.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "sse: connecting to %s\n", sse.URL)
	}

	p := pipe.New(sse.BufferSize)

	c, err := eventsource.Subscribe(sse.URL, "")
	if err != nil {
		return p.SetErr(err)
	}

	n := sse.Number

	go func() {
		defer c.Close()
		defer p.Close()

		for {
			if n == 0 {
				return
			}

			event := <-c.Events
			n--
			if !p.Send([]byte(event.Data())) && sse.Verbose > 0 {
				fmt.Fprintf(os.Stderr, "dropping event:%s\n", event)
			}
		}
	}()

	return p
}

func (sse *IO) Out(p *pipe.Pipe) error {
	return nil
}
