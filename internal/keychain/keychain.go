// Package keychain provides secure storage for the Cliniko API key.
package keychain

import (
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "elixir-medics"
	accountName = "cliniko-api-key"
)

// GetAPIKey retrieves the API key from the system keychain.
// Returns empty string if not found.
func GetAPIKey() (string, error) {
	key, err := keyring.Get(serviceName, accountName)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return key, nil
}

// SetAPIKey stores the API key in the system keychain.
func SetAPIKey(apiKey string) error {
	return keyring.Set(serviceName, accountName, apiKey)
}

// DeleteAPIKey removes the API key from the system keychain.
func DeleteAPIKey() error {
	err := keyring.Delete(serviceName, accountName)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}
