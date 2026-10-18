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

	go server.Start(":0", e.GetMessageChan())

	a := app.New(e)
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
