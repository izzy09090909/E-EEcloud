package vault

import (
	"encoding/json"
	"errors"
	"fmt"

	"os"
	"path/filepath"
	"sync"
	"time"

	"cipherdrive/pkg/crypto"
)

const (
	// VaultFileName defines the local encrypted metadata storage file name
	VaultFileName = "vault.dat"
	// KeyCheckToken is used to verify passphrase correctness during unlock
	KeyCheckToken = "CIPHERDRIVE_VALID_PASSPHRASE_TOKEN_v1"
)

// FileRecord represents metadata stored securely inside the backend vault
type FileRecord struct {
	ID            string          `json:"id"`
	OriginalName  string          `json:"original_name"`
	EncryptedName string          `json:"encrypted_name"`
	MimeType      string          `json:"mime_type"`
	SizeBytes     int64           `json:"size_bytes"`
	ChunkCount    int             `json:"chunk_count"`
	UploadTime    time.Time       `json:"upload_time"`
	Targets       map[string]bool `json:"targets"`       // e.g. {"gdrive": true, "laptop": true}
	EncryptedFEK  []byte          `json:"encrypted_fek"` // File Key encrypted with Master Vault Key
	IV            []byte          `json:"iv"`
}

// VaultData holds the persistent state saved in the local system directory
type VaultData struct {
	Salt                []byte                `json:"salt"`
	TestCheck           []byte                `json:"test_check"` // Verification block for password validation
	Files               map[string]FileRecord `json:"files"`
	EncryptedSettings   []byte                `json:"encrypted_settings,omitempty"`   // AppSettings, AES-256-GCM wrapped with masterKey
	ProtectedSaltBackup []byte                `json:"protected_salt_backup,omitempty"` // Optional OS-wrapped (DPAPI/Keychain) copy of Salt; defense-in-depth only, never required to unlock
}

// VaultManager handles zero-knowledge key abstraction and secure metadata storage
type VaultManager struct {
	mu          sync.RWMutex
	vaultPath   string
	masterKey   []byte
	isUnlocked  bool
	vaultData   VaultData
}

// NewVaultManager constructs a vault manager targeting the user's local system directory
func NewVaultManager() (*VaultManager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}

	appDir := filepath.Join(configDir, "CipherDrive")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create system vault directory: %w", err)
	}

	vaultPath := filepath.Join(appDir, VaultFileName)
	return &VaultManager{
		vaultPath: vaultPath,
		vaultData: VaultData{
			Files: make(map[string]FileRecord),
		},
	}, nil
}

// UnlockVault authenticates the user and decrypts/initializes the local key vault
func (v *VaultManager) UnlockVault(passphrase string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}

	// Check if vault already exists on disk
	if _, err := os.Stat(v.vaultPath); os.IsNotExist(err) {
		salt, err := crypto.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate key salt: %w", err)
		}

		// Derive key using Argon2id
		masterKey := crypto.DeriveKeyArgon2id(passphrase, salt)

		// Create verification token to validate password in future unlocks
		testCheck, err := crypto.EncryptBlock(masterKey, []byte(KeyCheckToken))
		if err != nil {
			return fmt.Errorf("failed to build key validation block: %w", err)
		}

		v.masterKey = masterKey
		v.vaultData = VaultData{
			Salt:      salt,
			TestCheck: testCheck,
			Files:     make(map[string]FileRecord),
		}
		v.isUnlocked = true

		return v.saveVaultLocked()
	}

	data, err := os.ReadFile(v.vaultPath)
	if err != nil {
		return fmt.Errorf("failed to read vault file: %w", err)
	}

	var rawVault VaultData
	if err := json.Unmarshal(data, &rawVault); err != nil {
		return fmt.Errorf("corrupted vault metadata format: %w", err)
	}

	// Derive key with stored salt using Argon2id
	derivedKey := crypto.DeriveKeyArgon2id(passphrase, rawVault.Salt)

	decryptedToken, err := crypto.DecryptBlock(derivedKey, rawVault.TestCheck)
	if err != nil || string(decryptedToken) != KeyCheckToken {
		return errors.New("invalid master vault passphrase")
	}

	v.masterKey = derivedKey
	v.vaultData = rawVault
	v.isUnlocked = true
	return nil
}

// LockVault clears the master key from RAM memory and locks the vault
func (v *VaultManager) LockVault() {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Zero out master key bytes in memory
	if v.masterKey != nil {
		for i := range v.masterKey {
			v.masterKey[i] = 0
		}
		v.masterKey = nil
	}

	v.isUnlocked = false
	v.vaultData.Files = make(map[string]FileRecord)
}

// IsUnlocked checks if the vault is currently unlocked
func (v *VaultManager) IsUnlocked() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.isUnlocked
}

// GetMasterKey returns a slice of the master key if unlocked (internal use)
func (v *VaultManager) GetMasterKey() ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.isUnlocked || v.masterKey == nil {
		return nil, errors.New("vault is locked")
	}

	keyCopy := make([]byte, len(v.masterKey))
	copy(keyCopy, v.masterKey)
	return keyCopy, nil
}

// AddFileRecord registers a new encrypted file entry into the vault
func (v *VaultManager) AddFileRecord(record FileRecord) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.isUnlocked {
		return errors.New("vault is locked")
	}

	v.vaultData.Files[record.ID] = record
	return v.saveVaultLocked()
}

// GetFileRecord retrieves metadata for a specific file ID
func (v *VaultManager) GetFileRecord(fileID string) (FileRecord, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.isUnlocked {
		return FileRecord{}, errors.New("vault is locked")
	}

	record, exists := v.vaultData.Files[fileID]
	if !exists {
		return FileRecord{}, fmt.Errorf("file ID '%s' not found in vault", fileID)
	}

	return record, nil
}

// ListFileRecords returns all registered file metadata records
func (v *VaultManager) ListFileRecords() ([]FileRecord, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.isUnlocked {
		return nil, errors.New("vault is locked")
	}

	list := make([]FileRecord, 0, len(v.vaultData.Files))
	for _, rec := range v.vaultData.Files {
		list = append(list, rec)
	}
	return list, nil
}

// RemoveFileRecord removes a file entry from the vault
func (v *VaultManager) RemoveFileRecord(fileID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.isUnlocked {
		return errors.New("vault is locked")
	}

	if _, exists := v.vaultData.Files[fileID]; !exists {
		return fmt.Errorf("file ID '%s' not found", fileID)
	}

	delete(v.vaultData.Files, fileID)
	return v.saveVaultLocked()
}

// HardenWithOSKeyProtection stores an OS-protected backup copy of the vault salt using
// Windows DPAPI or the macOS Keychain (see dpapi_windows.go / dpapi_other.go). This is
// defense-in-depth only: the vault remains fully unlockable with just the passphrase even
// if this backup is unavailable, unused, or the app runs on Linux (a no-op there).
func (v *VaultManager) HardenWithOSKeyProtection() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.isUnlocked {
		return errors.New("vault is locked")
	}
	if !IsPlatformKeyProtectionAvailable() {
		return errors.New("OS-native key protection is not available on this platform")
	}

	protected, err := ProtectSecret(v.vaultData.Salt)
	if err != nil {
		return fmt.Errorf("failed to wrap vault salt with OS key store: %w", err)
	}

	v.vaultData.ProtectedSaltBackup = protected
	return v.saveVaultLocked()
}

// saveVaultLocked writes the JSON metadata structure to disk (mutex assumed held)
func (v *VaultManager) saveVaultLocked() error {
	data, err := json.MarshalIndent(v.vaultData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault state: %w", err)
	}

	return os.WriteFile(v.vaultPath, data, 0600)
}