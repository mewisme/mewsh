package connect

import (
	"os"

	"github.com/mewisme/mewsh/internal/secret"
)

// AskpassEnvRef is read by the mewsh binary when invoked as SSH_ASKPASS.
const AskpassEnvRef = "MEWSH_ASKPASS_REF"

// RunAskpass prints the keyring password to stdout (no trailing newline).
func RunAskpass(ref string) error {
	pass, err := secret.GetPassword(ref)
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(pass)
	return err
}
