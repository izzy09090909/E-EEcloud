//go:build !windows
// +build !windows

package vault

import (
	"errors"
	"os/exec"
	"runtime"
)

// This file provides the non-Windows counterpart to dpapi_windows.go. On macOS it shells out
// to the `security` CLI to store a secret in the user's login Keychain; on Linux (and any
// other platform) it is a safe no-op passthrough, since the vault.dat file itself is already
// protected by the user's passphrase and OS-level wrapping is treated as optional
// defense-in-depth everywhere, never a requirement.

const (
	keychainService = "CipherDrive"
	keychainAccount = "vault-master-salt"
)

// ProtectSecret wraps a secret using the macOS Keychain when available; elsewhere it returns
// the input unmodified.
func ProtectSecret(plaintext []byte) ([]byte, error) {
	if runtime.GOOS == "darwin" {
		return protectWithKeychain(plaintext)
	}
	return plaintext, nil
}

// UnprotectSecret reverses ProtectSecret.
func UnprotectSecret(ciphertext []byte) ([]byte, error) {
	if runtime.GOOS == "darwin" {
		return unprotectWithKeychain(ciphertext)
	}
	return ciphertext, nil
}

// IsPlatformKeyProtectionAvailable reports whether OS-native secret wrapping is available.
func IsPlatformKeyProtectionAvailable() bool {
	return runtime.GOOS == "darwin"
}

// protectWithKeychain stores the secret in the macOS login Keychain via the `security` CLI
// and returns a small reference marker; the real bytes never touch vault.dat directly.
func protectWithKeychain(plaintext []byte) ([]byte, error) {
	// Best-effort cleanup of any prior entry so add-generic-password doesn't fail with
	// "already exists" on re-hardening.
	_ = exec.Command("security", "delete-generic-password", "-a", keychainAccount, "-s", keychainService).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-a", keychainAccount,
		"-s", keychainService,
		"-w", string(plaintext),
		"-U",
	)
	if err := cmd.Run(); err != nil {
		return nil, errors.New("keychain store failed: " + err.Error())
	}

	return []byte("KEYCHAIN_REF:" + keychainAccount), nil
}

func unprotectWithKeychain(_ []byte) ([]byte, error) {
	out, err := exec.Command("security", "find-generic-password",
		"-a", keychainAccount,
		"-s", keychainService,
		"-w",
	).Output()
	if err != nil {
		return nil, errors.New("keychain retrieve failed: " + err.Error())
	}
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return out, nil
}
