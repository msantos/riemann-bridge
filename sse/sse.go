// MIT License
//
// Copyright (c) 2021-2022 Michael Santos
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
	URL     string
	Verbose int
}

func (sse *IO) In(p *pipe.Pipe) *pipe.Pipe {
	if sse.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "sse: connecting to %s\n", sse.URL)
	}

	c, err := eventsource.Subscribe(sse.URL, "")
	if err != nil {
		return p.SetErr(err)
	}

	go func() {
		defer c.Close()
		defer p.Close()

		var event eventsource.Event

		go func() {
			errmsg := <-c.Errors
			if sse.Verbose > 0 {
				fmt.Fprintf(os.Stderr, "%s\n", errmsg)
			}
		}()

		for {
			event = <-c.Events

			ok, err := p.Send([]byte(event.Data()))

			if !ok && sse.Verbose > 0 {
				fmt.Fprintf(os.Stderr, "dropping event:%s\n", event)
			}

			if err != nil {
				return
			}
		}
	}()

	return p
}

func (sse *IO) Out(p *pipe.Pipe) error {
	return nil
}
