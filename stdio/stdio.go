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
package stdio

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/msantos/riemann-bridge/pipe"
)

type IO struct {
	BufferSize int
	Number     int
	Verbose    int
}

func (sio *IO) In() *pipe.Pipe {
	n := sio.Number

	p := pipe.New(sio.BufferSize)

	go func() {
		defer p.Close()

		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			n--

			in := scanner.Bytes()
			if bytes.TrimSpace(in) == nil {
				continue
			}

			m := make(map[string]interface{})
			if err := json.Unmarshal(in, &m); err != nil {
				if sio.Verbose > 0 {
					fmt.Fprintln(os.Stderr, err)
				}
				continue
			}

			if m["time"] == nil {
				m["time"] = time.Now()
			}

			event, err := json.Marshal(m)
			if err != nil {
				return
			}

			if !p.Send(event) && sio.Verbose > 0 {
				fmt.Fprintf(os.Stderr, "dropping event:%s\n", event)
			}

			if n == 0 {
				return
			}
		}
	}()

	return p
}

func (sio *IO) Out(p *pipe.Pipe) error {
	for p.Recv() {
		if _, err := fmt.Printf("%s\n", p.Bytes()); err != nil {
			return err
		}
	}

	if err := p.Err(); err != nil {
		return err
	}

	return nil
}
