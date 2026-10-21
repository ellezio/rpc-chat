package app

import (
	"errors"
	"fmt"
	"iter"
	"rpc-chat/chat"
	"rpc-chat/client"
	"rpc-chat/server"
	"strings"
)

type MessageWriter interface {
	client.AsyncMessageWriter
	WriteMessage(string)
}

type App struct {
	client     *client.Client
	server     *server.Server
	msgwr      MessageWriter
	username   string
	sendMsg    chat.SendChatMessageFn
	leaveChat  chat.LeaveChatFn
	chatClosed chan struct{}
}

func New(msgwr MessageWriter, srv *server.Server, username string) *App {
	a := &App{
		msgwr:      msgwr,
		server:     srv,
		username:   username,
		chatClosed: make(chan struct{}),
	}

	srv.ChatClosed = a.chatClosed

	return a
}

func (a *App) HandleInput(input string) bool {
	select {
	case <-a.chatClosed:
		if a.leaveChat != nil {
			a.leaveChat()
		}
		a.sendMsg = nil
		a.leaveChat = nil
	default:
	}

	if strings.HasPrefix(input, "/") {
		exit, err := a.handleCommand(input)
		if err != nil {
			a.msgwr.WriteMessage(err.Error())
		}

		return exit
	}

	if a.sendMsg != nil && a.leaveChat != nil {
		err := a.sendMsg(&chat.ChatMessage{Content: input})
		if err != nil {
			a.leaveChat()
			a.sendMsg = nil
			a.leaveChat = nil
		}
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
		sendMsg, leaveChat, err := a.server.ReplyChatRequest(addr, true)
		if err != nil {
			return false, err
		}

		a.sendMsg = sendMsg
		a.leaveChat = leaveChat

	case "/leave":
		if a.leaveChat != nil {
			a.leaveChat()
			a.sendMsg = nil
			a.leaveChat = nil
			a.msgwr.WriteMessage("you left the chat")
		}
	}

	return false, nil
}

func (a *App) handleConnect(addr string) error {
	clt := client.New(a.username, a.msgwr)
	sendMsg, leaveChat, err := clt.Connect(addr)
	if err != nil {
		return err
	}

	if sendMsg == nil {
		a.msgwr.WriteMessage("server not accepted request")
		return nil
	} else {
		a.msgwr.WriteMessage("chat request accepted")
	}

	a.client = clt
	a.sendMsg = sendMsg
	a.leaveChat = leaveChat
	return nil
}
