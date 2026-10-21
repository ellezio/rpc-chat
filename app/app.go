package app

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"
	"sync/atomic"
	"time"

	"rpc-chat/chat"
	"rpc-chat/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MessageWriter interface {
	WriteMessage(string)
}

type App struct {
	server    *server.Server
	msgwr     MessageWriter
	username  string
	sendMsg   chat.SendChatMessageFn
	leaveChat chat.LeaveChatFn
	inChat    atomic.Bool
}

func New(msgwr MessageWriter, srv *server.Server, username string) *App {
	a := &App{
		msgwr:    msgwr,
		server:   srv,
		username: username,
	}

	a.server.OnChatClose = a.handleCloseChat

	return a
}

func (a *App) HandleInput(input string) bool {
	inChat := a.inChat.Load()
	if (a.sendMsg != nil || a.leaveChat != nil) && !inChat {
		a.sendMsg = nil
		a.leaveChat = nil
	}

	if strings.HasPrefix(input, "/") {
		exit, err := a.handleCommand(input)
		if err != nil {
			a.msgwr.WriteMessage(err.Error())
		}

		return exit
	}

	if inChat {
		err := a.sendMsg(&chat.ChatMessage{Content: input})
		if err != nil {
			a.leaveChat()
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
		if a.inChat.Load() {
			return false, errors.New("cannot connect, already in chat")
		}

		addr, ok := next()
		if !ok {
			return false, errors.New("expected address")
		}
		if err := a.handleConnect(addr); err != nil {
			return false, fmt.Errorf("failed to connect: %w", err)
		}

	case "/accept":
		if a.inChat.Load() {
			return false, errors.New("cannot accept in chat")
		}

		addr, ok := next()
		if !ok {
			return false, errors.New("expected address")
		}
		sendMsg, leaveChat, err := a.server.ReplyChatRequest(addr, true)
		if err != nil {
			return false, err
		}

		a.handleStartChat(sendMsg, leaveChat)

	case "/leave":
		if a.inChat.Load() {
			a.leaveChat()
		} else {
			return false, errors.New("not in chat")
		}
	}

	return false, nil
}

func (a *App) handleConnect(addr string) error {
	a.server.RejectIncomingConnection("connecting to chat")

	if err := a.connect(addr); err != nil {
		a.server.AllowIncomingConnection()
		return err
	}

	return nil
}

func (a *App) connect(addr string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("creating gRPC client: %w", err)
	}

	chatClient := chat.NewChatClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	reply, err := chatClient.RequestChat(ctx, &chat.PeerInfo{Username: a.username})
	cancel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("requesting chat: %w", err)
	}

	if !reply.Accept {
		conn.Close()
		return fmt.Errorf("chat rejected, peer's reason: %s", reply.RejectionReason)
	}

	stream, err := chatClient.StartChatChannel(context.Background())
	if err != nil {
		conn.Close()
		return fmt.Errorf("starting chat channel: %w", err)
	}

	leaveChat := func() {
		_ = stream.CloseSend()
		_ = conn.Close()
		a.handleCloseChat()
	}

	go func() {
		peerUsername := reply.PeerInfo.Username
		err := chat.ReciveStreamMessages(stream.Recv, peerUsername, a.msgwr)
		if err != nil {
			a.msgwr.WriteMessage("ERROR: " + err.Error())
		}

		if a.inChat.Load() {
			a.msgwr.WriteMessage(fmt.Sprintf("%s closed the chat", peerUsername))
		} else {
			a.msgwr.WriteMessage("you closed the chat")
		}

		leaveChat()
	}()

	a.handleStartChat(stream.Send, leaveChat)

	return nil
}

func (a *App) handleStartChat(s chat.SendChatMessageFn, l chat.LeaveChatFn) {
	a.sendMsg = s
	a.leaveChat = l
	a.inChat.Store(true)
	a.server.RejectIncomingConnection("in chat")
}

// the cleanup of sendMsg and leaveChat is processed on next input so this method is safe to use concurrently
func (a *App) handleCloseChat() {
	a.inChat.Store(false)
	a.server.AllowIncomingConnection()
}
