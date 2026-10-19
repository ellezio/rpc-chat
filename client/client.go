package client

import (
	"context"
	"fmt"
	"rpc-chat/chat"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn         *grpc.ClientConn
	chatClient   chat.ChatClient
	username     string
	peerUsername string
}

func New(username string) *Client {
	return &Client{username: username}
}

func (c *Client) Connect(addr string) (bool, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false, fmt.Errorf("creating gRPC client: %w", err)
	}

	c.conn = conn
	c.chatClient = chat.NewChatClient(conn)

	reply, err := c.chatClient.RequestChat(context.Background(), &chat.PeerInfo{Username: c.username})
	if err != nil {
		conn.Close()
		return false, fmt.Errorf("requesting chat: %w", err)
	}

	if !reply.Accept {
		conn.Close()
		return reply.Accept, nil
	}

	c.peerUsername = reply.PeerInfo.Username

	return reply.Accept, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
