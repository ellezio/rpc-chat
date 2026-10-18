package editor

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

var (
	linefeed          = [4]byte{0x0d}
	endOfText         = [4]byte{0x03}
	endOfTransmission = [4]byte{0x04}
	backspace         = [4]byte{0x7f}
	delete_           = [4]byte{0x08}
)

type input struct {
	b [4]byte
	n int
}

type Editor struct {
	linebuf [1024]byte
	lineend int

	inputCh   chan input
	messageCh chan string

	in, out *os.File

	originalState *term.State
}

func New(in *os.File, out *os.File) *Editor {
	return &Editor{
		inputCh:   make(chan input, 1),
		messageCh: make(chan string, 1),
		in:        in,
		out:       out,
	}
}

func (e *Editor) Start() {
	go func() {
		var buf [4]byte
		for {
			clear(buf[:])
			n, err := e.in.Read(buf[:])
			if err == io.EOF {
				e.inputCh <- input{b: endOfTransmission, n: 1}
				return
			}

			e.inputCh <- input{b: buf, n: n}
			e.messageCh <- fmt.Sprintf("%v", buf[:n])
		}
	}()
}

func (e *Editor) Readline(prompt string) (string, error) {
	e.originalState, _ = term.MakeRaw(int(e.in.Fd()))
	defer term.Restore(int(e.in.Fd()), e.originalState)

	io.WriteString(e.out, prompt)

	for {
		select {
		case inp := <-e.inputCh:
			switch inp.b {
			case endOfText, endOfTransmission:
				return "", io.EOF
			case linefeed:
				end := e.lineend
				e.lineend = 0
				io.WriteString(e.out, "\r\n")
				return string(e.linebuf[:end]), nil
			case backspace, delete_:
				e.lineend = max(0, e.lineend-1)
				io.WriteString(e.out, "\x1b[1D\x1b[0K")
				continue
			}

			// is escape code
			if inp.b[0] == 27 {
				continue
			}

			copy(e.linebuf[e.lineend:], inp.b[:inp.n])
			e.lineend += inp.n

			e.out.Write(inp.b[:inp.n])

		case msg := <-e.messageCh:
			io.WriteString(e.out, "\r\x1b[2K")
			io.WriteString(e.out, msg)
			io.WriteString(e.out, "\r\n")

			io.WriteString(e.out, prompt)
			e.out.Write(e.linebuf[:e.lineend])
		}
	}
}

func (e *Editor) GetWriter() chan<- string {
	return e.messageCh
}
