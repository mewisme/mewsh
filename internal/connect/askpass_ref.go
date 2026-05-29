package connect

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mewsh/internal/config"
)

// AskpassModeArg is passed to mewsh.exe by the SSH_ASKPASS launcher script.
const AskpassModeArg = "__askpass__"

const askpassRefFile = ".askpass-ref"

// WriteAskpassRef stores which keyring entry to use for the next SSH_ASKPASS invocation.
func WriteAskpassRef(ref string) error {
	if ref == "" {
		return nil
	}
	if err := config.EnsureDir(); err != nil {
		return err
	}
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, askpassRefFile)
	return os.WriteFile(path, []byte(ref), 0600)
}

// ReadAskpassRefFile returns the keyring ref written by WriteAskpassRef.
func ReadAskpassRefFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, askpassRefFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ref := strings.TrimSpace(string(b))
	if ref == "" {
		return "", os.ErrNotExist
	}
	return ref, nil
}
