package chat

import "fmt"

type SendChatMessageFn func(*ChatMessage) error

type LeaveChatFn func()

func FormatMessage(msg string, username string) string {
	return fmt.Sprintf("[%s]> %s", username, msg)
}
