package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"rpc-chat/chat"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type StartChatMessage struct {
	sendMsg   chat.SendChatMessageFn
	leaveChat chat.LeaveChatFn
}

type Server struct {
	chat.UnimplementedChatServer

	msgCh    chan<- string
	username string

	mu                 sync.Mutex
	pendingChatRequest map[string]chan<- bool
	peerUsername       string

	startChat chan StartChatMessage

	ChatClosed chan<- struct{}
}

func New(msgCh chan<- string, username string) *Server {
	return &Server{
		msgCh:              msgCh,
		pendingChatRequest: make(map[string]chan<- bool),
		username:           username,
		startChat:          make(chan StartChatMessage),
	}
}

func (s *Server) RequestChat(ctx context.Context, pi *chat.PeerInfo) (*chat.RequestChatReply, error) {
	p, _ := peer.FromContext(ctx)
	s.msgCh <- fmt.Sprintf("Incoming chat request from: %s(%s)", pi.Username, p.Addr)

	ch := make(chan bool)
	s.mu.Lock()
	s.pendingChatRequest[p.Addr.String()] = ch
	s.mu.Unlock()

	deadline := time.After(time.Minute)

	accept := false

	select {
	case accept = <-ch:
		s.mu.Lock()
		s.peerUsername = pi.Username
		s.mu.Unlock()
	case <-deadline:
		s.msgCh <- fmt.Sprintf("%s(%s) Chat request has expired", pi.Username, p.Addr)
		s.mu.Lock()
		delete(s.pendingChatRequest, p.Addr.String())
		s.mu.Unlock()
	}

	return &chat.RequestChatReply{Accept: accept, PeerInfo: &chat.PeerInfo{Username: s.username}}, nil
}

func (s *Server) ReplyChatRequest(id string, accept bool) (chat.SendChatMessageFn, chat.LeaveChatFn, error) {
	s.mu.Lock()
	if ch, ok := s.pendingChatRequest[id]; ok {
		delete(s.pendingChatRequest, id)
		ch <- accept
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
	done := make(chan struct{})
	closeChat := func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}

	s.startChat <- StartChatMessage{
		sendMsg:   stream.Send,
		leaveChat: closeChat,
	}

	// When StartChatChannel returns the contex is canceled so the gorouting terminates
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					s.msgCh <- fmt.Sprintf("%s left the chat", s.peerUsername)
					break
				}

				if status.Code(err) == codes.Canceled {
					break
				}

				s.msgCh <- "ERROR: " + err.Error()
				break
			}

			s.msgCh <- chat.FormatMessage(msg.Content, s.peerUsername)
		}

		s.ChatClosed <- struct{}{}
		closeChat()
	}()

	<-done

	return nil
}

func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := grpc.NewServer()
	chat.RegisterChatServer(srv, s)

	s.msgCh <- fmt.Sprintf("server at %s", lis.Addr())
	return srv.Serve(lis)
}
