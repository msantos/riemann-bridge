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
package pipe

type Pipe struct {
	ch  chan []byte
	err error
	buf []byte
}

type Piper interface {
	In() *Pipe
	Out(*Pipe) error
}

func New(bufsize int) *Pipe {
	ch := make(chan []byte, bufsize)
	return &Pipe{ch: ch}
}

func (p *Pipe) Close() {
	close(p.ch)
}

func (p *Pipe) SetErr(err error) *Pipe {
	p.err = err
	return p
}

func (p *Pipe) Send(b []byte) bool {
	if len(p.ch) == 0 {
		p.ch <- b
		return true
	}

	select {
	case p.ch <- b:
		return true
	default:
		return false
	}
}

func (p *Pipe) Recv() bool {
	if p.err != nil {
		return false
	}

	var ok bool
	p.buf, ok = <-p.ch

	return ok
}

func (p *Pipe) Bytes() []byte {
	return p.buf
}

func (p *Pipe) Err() error {
	return p.err
}
