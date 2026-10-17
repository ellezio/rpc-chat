package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

func Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	fmt.Printf("server at %s\n", lis.Addr())
	return s.Serve(lis)
}
