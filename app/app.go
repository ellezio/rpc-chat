package app

import (
	"errors"
	"fmt"
	"iter"
	"rpc-chat/client"
	"rpc-chat/server"
	"strings"
)

type MessageWriter interface {
	WriteMessage(string)
}

type App struct {
	client   *client.Client
	server   *server.Server
	msgwr    MessageWriter
	username string
}

func New(msgwr MessageWriter, srv *server.Server, username string) *App {
	return &App{msgwr: msgwr, server: srv, username: username}
}

func (a *App) HandleInput(input string) bool {
	if strings.HasPrefix(input, "/") {
		exit, err := a.handleCommand(input)
		if err != nil {
			a.msgwr.WriteMessage(err.Error())
		}

		return exit
	}

	return false
}

func (a *App) handleCommand(command string) (bool, error) {
	cmdIter := strings.SplitSeq(command, " ")
	next, stop := iter.Pull(cmdIter)
	defer stop()

	cmd, ok := next()
	if !ok {
		return false, errors.New("invalid command")
	}

	switch cmd {
	case "/exit":
		return true, nil
	case "/connect":
		addr, ok := next()
		if !ok {
			return false, errors.New("expected address")
		}
		if err := a.handleConnect(addr); err != nil {
			return false, fmt.Errorf("failed to connect: %w", err)
		}
	case "/accept":
		addr, ok := next()
		if !ok {
			return false, errors.New("expected address")
		}
		a.server.ReplyChatRequest(addr, true)
	}

	return false, nil
}

func (a *App) handleConnect(addr string) error {
	clt := client.New(a.username)
	accepted, err := clt.Connect(addr)
	if err != nil {
		return err
	}

	if !accepted {
		a.msgwr.WriteMessage("server not accepted request")
		return nil
	} else {
		a.msgwr.WriteMessage("chat request accepted")
	}

	a.client = clt
	return nil
}
