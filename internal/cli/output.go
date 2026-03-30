package cli

import (
	"fmt"
	"io"
)

func WriteLine(w io.Writer, msg string) error {
	_, err := fmt.Fprintln(w, msg)
	return err
}
