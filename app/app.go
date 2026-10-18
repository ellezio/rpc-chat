package app

import (
	"errors"
	"fmt"
	"iter"
	"rpc-chat/client"
	"strings"
)

type App struct {
	client *client.Client
}

func New() *App {
	return &App{}
}

func (a *App) HandleInput(input string, msgCh chan<- string) bool {
	if strings.HasPrefix(input, "/") {
		exit, err := a.handleCommand(input)
		if err != nil {
			msgCh <- err.Error()
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
	}

	return false, nil
}

func (a *App) handleConnect(addr string) error {
	clt := client.New()
	accepted, err := clt.Connect(addr)
	if err != nil {
		return err
	}

	if !accepted {
		fmt.Println("server not accepted request")
		return nil
	}

	a.client = clt
	return nil
}
