package repl

import (
	"bufio"
	"io"
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

		io.WriteString(out, prompt)
		io.WriteString(out, "\n")
	}
}
