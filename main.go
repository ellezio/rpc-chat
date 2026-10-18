package main

import (
	"os"
	"rpc-chat/app"
	"rpc-chat/editor"
	"rpc-chat/server"
	"strings"
)

func main() {
	e := editor.New(os.Stdin, os.Stdout)
	e.Start()

	go server.Start(":0", e.GetWriter())

	a := app.New()
	for {
		line, err := e.Readline(">> ")
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)

		if a.HandleInput(line, e.GetWriter()) {
			return
		}
	}
}
