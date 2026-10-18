package server

import (
	"context"
	"fmt"
	"net"
	"rpc-chat/chat"

	"google.golang.org/grpc"
)

type server struct {
	chat.UnimplementedChatServer
}

func (s *server) RequestChat(_ context.Context, req *chat.RequestChatRequest) (*chat.RequestChatReply, error) {
	return &chat.RequestChatReply{Accept: true}, nil
}

func Start(addr string, msgCh chan<- string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServer(s, &server{})

	msgCh <- fmt.Sprintf("server at %s", lis.Addr())
	return s.Serve(lis)
}
