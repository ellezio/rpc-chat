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

	srv := server.New(e.GetMessageChan())
	go srv.Start(":0")

	a := app.New(e, srv)

	e.SetPrompt(">> ")
	for {
		line, err := e.Readline()
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)

		if a.HandleInput(line) {
			return
		}
	}
}
