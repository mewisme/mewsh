package secret

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const Service = "mewsh"

func SetPassword(ref, password string) error {
	if ref == "" {
		return fmt.Errorf("password reference is empty")
	}
	return keyring.Set(Service, ref, password)
}

func GetPassword(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("password reference is empty")
	}
	pass, err := keyring.Get(Service, ref)
	if err != nil {
		return "", fmt.Errorf("get password from keyring: %w", err)
	}
	return pass, nil
}

func DeletePassword(ref string) error {
	if ref == "" {
		return nil
	}
	err := keyring.Delete(Service, ref)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
