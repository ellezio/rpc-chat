package editor

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

var (
	linefeed          byte = 0x0d
	endOfText         byte = 0x03
	endOfTransmission byte = 0x04
	backspace         byte = 0x7f
	delete_           byte = 0x08
)

type input struct {
	b [4]byte
	n int
}

type Editor struct {
	mu         sync.Mutex
	linebuf    [1024]byte
	lineend    int
	prompt     string
	renderLine bool

	messageCh chan string

	in, out *os.File
}

func New(in *os.File, out *os.File) *Editor {
	return &Editor{
		messageCh: make(chan string, 8),
		in:        in,
		out:       out,
	}
}

func (e *Editor) Start() {
	go func() {
		for msg := range e.messageCh {
			e.mu.Lock()
			io.WriteString(e.out, "\r\x1b[2K")
			io.WriteString(e.out, msg)
			io.WriteString(e.out, "\r\n")
			if e.renderLine {
				io.WriteString(e.out, e.prompt)
				e.out.Write(e.linebuf[:e.lineend])
			}
			e.mu.Unlock()
		}
	}()
}

func (e *Editor) SetPrompt(prompt string) {
	e.prompt = prompt
}

func (e *Editor) Readline() (string, error) {
	originalState, _ := term.MakeRaw(int(e.in.Fd()))
	defer term.Restore(int(e.in.Fd()), originalState)

	e.mu.Lock()
	e.renderLine = true
	io.WriteString(e.out, e.prompt)
	e.mu.Unlock()

	var buf [1]byte
	for {
		clear(buf[:])
		n, err := e.in.Read(buf[:])
		if err == io.EOF || n == 0 {
			e.mu.Lock()
			e.renderLine = false
			e.mu.Unlock()
			return "", io.EOF
		}

		e.mu.Lock()
		switch buf[0] {
		case endOfText, endOfTransmission:
			e.renderLine = false
			e.mu.Unlock()
			return "", io.EOF
		case linefeed:
			line := string(e.linebuf[:e.lineend])
			e.lineend = 0
			e.renderLine = false
			io.WriteString(e.out, "\r\n")
			e.mu.Unlock()
			return line, nil
		case backspace, delete_:
			e.lineend = max(0, e.lineend-1)
			io.WriteString(e.out, "\x1b[1D\x1b[0K")
			e.mu.Unlock()
			continue
		}

		// is escape code
		if buf[0] == 27 {
			e.mu.Unlock()
			continue
		}

		if e.lineend == len(e.linebuf) {
			e.WriteMessage("cannot read input - max line size reached")
			continue
		}

		e.linebuf[e.lineend] = buf[0]
		e.lineend += 1

		e.out.Write(buf[:])
		e.mu.Unlock()
	}
}

func (e *Editor) WriteMessage(msg string) {
	e.messageCh <- msg
}
