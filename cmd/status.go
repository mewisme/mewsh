package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

var errFormCancelled = errors.New("form cancelled")

func isFormCancelled(err error) bool {
	return errors.Is(err, errFormCancelled) || errors.Is(err, huh.ErrUserAborted)
}

func wrapFormErr(err error) error {
	if err == nil {
		return nil
	}
	if isFormCancelled(err) {
		return errFormCancelled
	}
	return err
}

func printStatus(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
