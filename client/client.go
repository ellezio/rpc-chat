package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"rpc-chat/chat"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type AsyncMessageWriter interface {
	AsyncWriteMessage(string)
}

type Client struct {
	msgwr    AsyncMessageWriter
	username string
}

func New(username string, msgwr AsyncMessageWriter) *Client {
	return &Client{username: username, msgwr: msgwr}
}

func (c *Client) Connect(addr string) (chat.SendChatMessageFn, chat.LeaveChatFn, error) {
	ctx := context.Background()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("creating gRPC client: %w", err)
	}

	chatClient := chat.NewChatClient(conn)

	reply, err := chatClient.RequestChat(ctx, &chat.PeerInfo{Username: c.username})
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("requesting chat: %w", err)
	}

	if !reply.Accept {
		conn.Close()
		return nil, nil, nil
	}

	stream, err := chatClient.StartChatChannel(ctx)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("starting chat channel: %w", err)
	}

	go func() {
		peerUsername := reply.PeerInfo.Username
		for {
			msg, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					c.msgwr.AsyncWriteMessage(fmt.Sprintf("%s left the chat", peerUsername))
					break
				}

				if status.Code(err) == codes.Canceled {
					break
				}

				c.msgwr.AsyncWriteMessage("ERROR: unexpected error occurred when reading peer's chat message: " + err.Error())
				break
			}

			c.msgwr.AsyncWriteMessage(chat.FormatMessage(msg.Content, peerUsername))
		}
		_ = conn.Close()
	}()

	return stream.Send, func() {
		_ = stream.CloseSend()
		_ = conn.Close()
	}, nil
}
