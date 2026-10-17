package main

import (
	"fmt"
	"io"
	"os"
	"rpc-chat/repl"
)

func main() {
	if err := repl.Start(os.Stdin, os.Stdout); err != nil {
		io.WriteString(os.Stderr, fmt.Sprintf("Repl exited with error: %v", err))
	}
}
