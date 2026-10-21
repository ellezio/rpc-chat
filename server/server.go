package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"rpc-chat/chat"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type MessageWriter interface {
	WriteMessage(string)
}

type StartChatMessage struct {
	sendMsg   chat.SendChatMessageFn
	leaveChat chat.LeaveChatFn
}

type Server struct {
	chat.UnimplementedChatServer

	msgwr    MessageWriter
	username string

	mu                     sync.Mutex
	pendingChatRequest     map[string]chan<- bool
	peerUsername           string
	rejectConnection       bool
	rejectConnectionReason string

	startChat chan StartChatMessage

	OnChatClose func()
}

func New(msgwr MessageWriter, username string) *Server {
	return &Server{
		msgwr:              msgwr,
		pendingChatRequest: make(map[string]chan<- bool),
		username:           username,
		startChat:          make(chan StartChatMessage),
	}
}

func (s *Server) RequestChat(ctx context.Context, pi *chat.PeerInfo) (*chat.RequestChatReply, error) {
	s.mu.Lock()
	if s.rejectConnection {
		reply := &chat.RequestChatReply{
			Accept:          false,
			PeerInfo:        &chat.PeerInfo{Username: s.username},
			RejectionReason: s.rejectConnectionReason,
		}
		s.mu.Unlock()
		return reply, nil
	}
	s.mu.Unlock()

	p, _ := peer.FromContext(ctx)
	s.msgwr.WriteMessage(fmt.Sprintf("Incoming chat request from: %s(%s)", pi.Username, p.Addr))

	ch := make(chan bool)
	s.mu.Lock()
	s.pendingChatRequest[p.Addr.String()] = ch
	s.mu.Unlock()

	deadline := time.After(time.Minute)

	select {
	case accept := <-ch:
		s.mu.Lock()
		s.peerUsername = pi.Username
		s.mu.Unlock()
		return &chat.RequestChatReply{Accept: accept, PeerInfo: &chat.PeerInfo{Username: s.username}}, nil

	case <-deadline:
		s.msgwr.WriteMessage(fmt.Sprintf("%s(%s) Chat request has expired", pi.Username, p.Addr))
		s.mu.Lock()
		delete(s.pendingChatRequest, p.Addr.String())
		s.mu.Unlock()
		return &chat.RequestChatReply{Accept: false, PeerInfo: &chat.PeerInfo{Username: s.username}, RejectionReason: "chat request has expired"}, nil

	case <-ctx.Done():
		return nil, nil
	}
}

func (s *Server) ReplyChatRequest(id string, accept bool) (chat.SendChatMessageFn, chat.LeaveChatFn, error) {
	s.mu.Lock()
	if ch, ok := s.pendingChatRequest[id]; ok {
		delete(s.pendingChatRequest, id)
		ch <- accept
	} else {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("no incoming connection from %s", id)
	}
	s.mu.Unlock()

	deadline := time.After(10 * time.Second)

	select {
	case sc := <-s.startChat:
		return sc.sendMsg, sc.leaveChat, nil
	case <-deadline:
		return nil, nil, errors.New("%s(%s) Chat accept timeout")
	}
}

func (s *Server) StartChatChannel(stream grpc.BidiStreamingServer[chat.ChatMessage, chat.ChatMessage]) error {
	serverClosedChat := atomic.Bool{}
	done := make(chan struct{})
	closeChat := func() {
		serverClosedChat.Store(true)
		select {
		case done <- struct{}{}:
		default:
		}
	}

	s.startChat <- StartChatMessage{
		sendMsg:   stream.Send,
		leaveChat: closeChat,
	}

	// When StartChatChannel returns, the context is canceled so the gorouting terminates
	go func() {
		err := chat.ReciveStreamMessages(stream.Recv, s.peerUsername, s.msgwr)
		if err != nil {
			s.msgwr.WriteMessage("ERROR: " + err.Error())
		}

		if serverClosedChat.Load() {
			s.msgwr.WriteMessage("you closed the chat")
		} else {
			s.msgwr.WriteMessage(fmt.Sprintf("%s closed the chat", s.peerUsername))
		}

		closeChat()
		s.OnChatClose()
	}()

	<-done

	return nil
}

func (s *Server) AllowIncomingConnection() {
	s.mu.Lock()
	s.rejectConnection = false
	s.rejectConnectionReason = ""
	s.mu.Unlock()
}

func (s *Server) RejectIncomingConnection(reason string) {
	s.mu.Lock()
	s.rejectConnection = true
	s.rejectConnectionReason = reason
	s.mu.Unlock()
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := grpc.NewServer()
	chat.RegisterChatServer(srv, s)

	s.msgwr.WriteMessage(fmt.Sprintf("server at %s", lis.Addr()))
	return srv.Serve(lis)
}
