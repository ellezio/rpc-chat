package client

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
}

func New() *Client {
	return &Client{}
}

func (c *Client) Connect(addr string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("creating gRPC client: %w", err)
	}

	c.conn = conn

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
