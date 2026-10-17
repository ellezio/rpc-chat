package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
	"rpc-chat/client"
	"rpc-chat/server"
	"strings"
)

const promptPrefix = ">> "

type App struct {
	client *client.Client
}

func New() *App {
	return &App{}
}

func (a *App) Start(in io.Reader, out io.Writer) error {
	a.startServer()

	scanner := bufio.NewScanner(in)

	for {
		out.Write([]byte(promptPrefix))

		if !scanner.Scan() {
			return scanner.Err()
		}

		prompt := scanner.Text()
		prompt = strings.TrimSpace(prompt)

		if strings.HasPrefix(prompt, "/") {
			exit, err := a.handleCommand(prompt)
			if err != nil {
				io.WriteString(out, err.Error())
				io.WriteString(out, "\n")
			}

			if exit {
				return nil
			}
		}

		io.WriteString(out, prompt)
		io.WriteString(out, "\n")
	}
}

func (a *App) startServer() {
	go server.Start(":0")
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
	if err := clt.Connect(addr); err != nil {
		return err
	}

	a.client = clt
	return nil
}
