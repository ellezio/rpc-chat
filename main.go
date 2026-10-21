package main

import (
	"flag"
	"fmt"
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
		io.WriteString(os.Stderr, "-username [username] flag is required")
		os.Exit(1)
	}

	e := editor.New(os.Stdin, os.Stdout)
	e.Start()

	srv := server.New(e, *username)
	a := app.New(e, srv, *username)
	go srv.Start(":0")

	e.SetPrompt(fmt.Sprintf("[%s]> ", *username))
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
