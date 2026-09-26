package vault

import (
	"encoding/json"
	"errors"
	"fmt"

	"cipherdrive/pkg/crypto"
	"cipherdrive/pkg/storage"
)

// AppSettings holds user-configured cloud target credentials and app-level preferences.
// This whole struct is stored inside the vault, AES-256-GCM wrapped with the master key
// (see SaveSettings/GetSettings below), so OAuth tokens and laptop-node auth tokens never
// touch disk in plaintext -- they only exist in the clear in RAM while the vault is unlocked.
type AppSettings struct {
	GDrive         storage.GDriveConfig     `json:"gdrive"`
	GDriveEnabled  bool                     `json:"gdrive_enabled"`
	GDriveTokenExp int64                    `json:"gdrive_token_exp"` // unix seconds
	Laptop         storage.LaptopNodeConfig `json:"laptop"`
	LaptopEnabled  bool                     `json:"laptop_enabled"`
}

// SaveSettings encrypts and persists app-level settings (including cloud credentials)
// inside the vault using the master key. Vault must be unlocked.
func (v *VaultManager) SaveSettings(settings AppSettings) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.isUnlocked || v.masterKey == nil {
		return errors.New("vault is locked")
	}

	plain, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal app settings: %w", err)
	}

	encrypted, err := crypto.EncryptBlock(v.masterKey, plain)
	if err != nil {
		return fmt.Errorf("failed to encrypt app settings: %w", err)
	}

	v.vaultData.EncryptedSettings = encrypted
	return v.saveVaultLocked()
}

// GetSettings decrypts and returns the app-level settings. Returns a zero-value AppSettings
// (not an error) if nothing has been saved yet, so callers can treat "unconfigured" and
// "empty" the same way.
func (v *VaultManager) GetSettings() (AppSettings, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.isUnlocked || v.masterKey == nil {
		return AppSettings{}, errors.New("vault is locked")
	}

	if len(v.vaultData.EncryptedSettings) == 0 {
		return AppSettings{}, nil
	}

	plain, err := crypto.DecryptBlock(v.masterKey, v.vaultData.EncryptedSettings)
	if err != nil {
		return AppSettings{}, fmt.Errorf("failed to decrypt app settings: %w", err)
	}

	var settings AppSettings
	if err := json.Unmarshal(plain, &settings); err != nil {
		return AppSettings{}, fmt.Errorf("corrupted app settings payload: %w", err)
	}

	return settings, nil
}
