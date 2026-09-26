//go:build windows
// +build windows

package vault

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// This file wraps the Windows Data Protection API (crypt32.dll) to add an optional,
// defense-in-depth layer on top of the passphrase-derived vault: a secret protected here can
// only be unprotected by the same Windows user account on the same machine. It is invoked
// only via VaultManager.HardenWithOSKeyProtection() (pkg/vault/vault.go) and is never
// required to unlock the vault -- losing access to DPAPI (e.g. a rebuilt user profile) does
// not brick the vault, since the passphrase-derived Salt/TestCheck path always still works.

const cryptProtectUIForbidden = 0x1

var (
	modcrypt32             = syscall.NewLazyDLL("crypt32.dll")
	modkernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtectData   = modcrypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = modcrypt32.NewProc("CryptUnprotectData")
	procLocalFree           = modkernel32.NewProc("LocalFree")
)

// dpapiBlob mirrors the Win32 DATA_BLOB struct expected by CryptProtectData/CryptUnprotectData.
type dpapiBlob struct {
	cbData uint32
	pbData *byte
}

func newDPAPIBlob(d []byte) *dpapiBlob {
	if len(d) == 0 {
		return &dpapiBlob{}
	}
	return &dpapiBlob{cbData: uint32(len(d)), pbData: &d[0]}
}

func (b *dpapiBlob) bytes() []byte {
	if b.cbData == 0 || b.pbData == nil {
		return nil
	}
	return unsafe.Slice(b.pbData, int(b.cbData))
}

// ProtectSecret wraps sensitive bytes (e.g. the vault salt) using the current Windows user's
// DPAPI master key, binding decryption to this OS user account on this machine.
func ProtectSecret(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("cannot protect empty secret")
	}

	in := newDPAPIBlob(plaintext)
	entropy := newDPAPIBlob([]byte("CipherDrive-DPAPI-Entropy-v1"))
	var out dpapiBlob

	ret, _, callErr := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(in)),
		0, // description
		uintptr(unsafe.Pointer(entropy)),
		0, // reserved
		0, // prompt struct
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, dpapiError("CryptProtectData", callErr)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	protected := make([]byte, out.cbData)
	copy(protected, out.bytes())
	return protected, nil
}

// UnprotectSecret reverses ProtectSecret, decrypting bytes previously wrapped with DPAPI
// under this Windows user account.
func UnprotectSecret(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("cannot unprotect empty secret")
	}

	in := newDPAPIBlob(ciphertext)
	entropy := newDPAPIBlob([]byte("CipherDrive-DPAPI-Entropy-v1"))
	var out dpapiBlob

	ret, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(in)),
		0,
		uintptr(unsafe.Pointer(entropy)),
		0,
		0,
		cryptProtectUIForbidden,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, dpapiError("CryptUnprotectData", callErr)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	plain := make([]byte, out.cbData)
	copy(plain, out.bytes())
	return plain, nil
}

func dpapiError(op string, sysErr error) error {
	if sysErr != nil {
		if errno, ok := sysErr.(syscall.Errno); ok && errno == 0 {
			return errors.New(op + " failed (unknown error)")
		}
		return fmt.Errorf("%s failed: %w", op, sysErr)
	}
	return errors.New(op + " failed")
}

// IsPlatformKeyProtectionAvailable reports whether OS-native secret wrapping is available.
func IsPlatformKeyProtectionAvailable() bool {
	return true
}
