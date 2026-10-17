package main

import (
	"fmt"
	"io"
	"os"
	"rpc-chat/app"
)

func main() {
	a := app.New()
	if err := a.Start(os.Stdin, os.Stdout); err != nil {
		io.WriteString(os.Stderr, fmt.Sprintf("Repl exited with error: %v", err))
	}
}
