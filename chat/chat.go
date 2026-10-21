package chat

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MessageWriter interface {
	WriteMessage(string)
}

type SendChatMessageFn func(*ChatMessage) error

type LeaveChatFn func()

func FormatMessage(msg string, username string) string {
	return fmt.Sprintf("[%s]> %s", username, msg)
}

type MessageStreamer = func() (*ChatMessage, error)

func ReciveStreamMessages(recv MessageStreamer, peerUsername string, msgwr MessageWriter) error {
	for {
		msg, err := recv()
		if err != nil {
			code := status.Code(err)
			if errors.Is(err, io.EOF) || code == codes.Canceled || code == codes.Unavailable {
				return nil
			}

			return err
		}

		msgwr.WriteMessage(FormatMessage(msg.Content, peerUsername))
	}
}
