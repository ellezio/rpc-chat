package main

import (
	"flag"
	"io"
	"os"
	"rpc-chat/app"
	"rpc-chat/editor"
	"rpc-chat/server"
	"strings"
)

var username = flag.String("username", "", "Name of user shown in chat")

func main() {
	flag.Parse()
	if *username == "" {
		io.WriteString(os.Stderr, "-username flag is required")
		os.Exit(1)
	}

	e := editor.New(os.Stdin, os.Stdout)
	e.Start()

	srv := server.New(e.GetMessageChan(), *username)
	go srv.Start(":0")

	a := app.New(e, srv, *username)

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
