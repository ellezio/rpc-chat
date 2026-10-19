package server

import (
	"context"
	"fmt"
	"net"
	"rpc-chat/chat"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type server struct {
	msgCh chan<- string
	chat.UnimplementedChatServer
}

func (s *server) RequestChat(ctx context.Context, req *chat.RequestChatRequest) (*chat.RequestChatReply, error) {
	p, _ := peer.FromContext(ctx)
	s.msgCh <- fmt.Sprintf("Incoming chat request from: %s(%s)", req.Name, p.Addr)
	return &chat.RequestChatReply{Accept: true}, nil
}

func Start(addr string, msgCh chan<- string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	chat.RegisterChatServer(s, &server{msgCh: msgCh})

	msgCh <- fmt.Sprintf("server at %s", lis.Addr())
	return s.Serve(lis)
}
