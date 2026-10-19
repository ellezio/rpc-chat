package server

import (
	"context"
	"fmt"
	"net"
	"rpc-chat/chat"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type Server struct {
	chat.UnimplementedChatServer

	msgCh chan<- string

	mu                 sync.Mutex
	pendingChatRequest map[string]chan<- bool
}

func New(msgCh chan<- string) *Server {
	return &Server{msgCh: msgCh, pendingChatRequest: make(map[string]chan<- bool)}
}

func (s *Server) RequestChat(ctx context.Context, req *chat.RequestChatRequest) (*chat.RequestChatReply, error) {
	p, _ := peer.FromContext(ctx)
	s.msgCh <- fmt.Sprintf("Incoming chat request from: %s(%s)", req.Name, p.Addr)

	ch := make(chan bool)
	s.mu.Lock()
	s.pendingChatRequest[p.Addr.String()] = ch
	s.mu.Unlock()

	deadline := time.After(time.Minute)

	accept := false

	select {
	case accept = <-ch:
	case <-deadline:
		s.msgCh <- fmt.Sprintf("%s(%s) Chat request has expired", req.Name, p.Addr)
		s.mu.Lock()
		delete(s.pendingChatRequest, p.Addr.String())
		s.mu.Unlock()
	}

	return &chat.RequestChatReply{Accept: accept}, nil
}

func (s *Server) ReplyChatRequest(id string, accept bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.pendingChatRequest[id]; ok {
		delete(s.pendingChatRequest, id)
		ch <- accept
	}

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
