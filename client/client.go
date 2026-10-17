package client

import (
	"context"
	"fmt"
	"rpc-chat/chat"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn       *grpc.ClientConn
	chatClient chat.ChatClient
}

func New() *Client {
	return &Client{}
}

func (c *Client) Connect(addr string) (bool, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false, fmt.Errorf("creating gRPC client: %w", err)
	}

	c.conn = conn
	c.chatClient = chat.NewChatClient(conn)

	accepted, err := c.RequestChat(context.Background())
	if err != nil {
		conn.Close()
		return false, fmt.Errorf("requesting chat: %w")
	}

	if !accepted {
		conn.Close()
	}

	return accepted, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) RequestChat(ctx context.Context) (bool, error) {
	reply, err := c.chatClient.RequestChat(ctx, &chat.RequestChatRequest{Name: "edek"})
	if err != nil {
		return false, err
	}

	return reply.Accept, nil
}
