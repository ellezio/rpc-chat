package repl

import (
	"bufio"
	"errors"
	"io"
	"iter"
	"strings"
)

const promptPrefix = ">> "

func Start(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)

	for {
		out.Write([]byte(promptPrefix))

		if !scanner.Scan() {
			return scanner.Err()
		}

		prompt := scanner.Text()
		prompt = strings.TrimSpace(prompt)

		if strings.HasPrefix(prompt, "/") {
			exit, err := handleCommand(prompt)
			if err != nil {
				io.WriteString(out, err.Error())
				io.WriteString(out, "\n")
			}

			if exit {
				return nil
			}
		}

		io.WriteString(out, prompt)
		io.WriteString(out, "\n")
	}
}

func handleCommand(command string) (bool, error) {
	cmdIter := strings.SplitSeq(command, " ")
	next, stop := iter.Pull(cmdIter)
	defer stop()

	cmd, ok := next()
	if !ok {
		return false, errors.New("invalid command")
	}

	switch cmd {
	case "/exit":
		return true, nil
	}

	return false, nil
}
