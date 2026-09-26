package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"cipherdrive/pkg/crypto"
	"cipherdrive/pkg/display"
	"cipherdrive/pkg/storage"
	"cipherdrive/pkg/stream"
	"cipherdrive/pkg/vault"
)

// App is the Wails-bound backend. Every exported method on App is callable from the React
// frontend via window.go.main.App.<Method>. Per the zero-knowledge architecture, plaintext
// file bytes, the master key, and every FEK are handled strictly in Go and are never sent to
// the JS side -- only metadata, progress numbers, and opaque status/URL values cross the
// boundary.
type App struct {
	ctx context.Context

	vaultMgr     *vault.VaultManager
	dispatcher   *storage.Dispatcher
	gdriveDriver *storage.GDriveDriver
	laptopDriver *storage.LaptopDriver
	streamSrv    *stream.LoopbackServer

	stagingDir string
}

// NewApp constructs the (not-yet-initialized) App. Real setup happens in startup, once Wails
// hands us a context.
func NewApp() *App {
	return &App{}
}

// startup is called once by Wails after the frontend is ready.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	vm, err := vault.NewVaultManager()
	if err != nil {
		wailsruntime.LogFatalf(ctx, "failed to initialize vault manager: %v", err)
		return
	}
	a.vaultMgr = vm

	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	a.stagingDir = filepath.Join(configDir, "CipherDrive", "staging")
	if err := os.MkdirAll(a.stagingDir, 0700); err != nil {
		wailsruntime.LogErrorf(ctx, "failed to create staging directory: %v", err)
	}
	// 1. Enforce Win32 DWM Blackout while window is still hidden
	if err := display.EnableBlackoutByTitle("CipherDrive"); err != nil {
		fmt.Printf("[Security Warning] Blackout mode failed: %v\n", err)
	} else {
		fmt.Println("[Security Engine] Win32 capture exclusion active (WDA_EXCLUDEFROMCAPTURE)")
	}

	// 2. Reveal the window now that DWM protection is active
	wailsruntime.WindowShow(ctx)

	streamSrv, err := stream.NewLoopbackServer()
	if err != nil {
		wailsruntime.LogErrorf(ctx, "failed to start loopback streaming server: %v", err)
	} else {
		a.streamSrv = streamSrv
		a.streamSrv.Start()
	}

	// Drivers start unconfigured. UnlockVault -> reloadDriversFromSettings wires them up
	// once saved settings (if any) are available.
	a.dispatcher = storage.NewDispatcher(nil, nil)
}

// shutdown is called by Wails as the app closes.
func (a *App) shutdown(ctx context.Context) {
	if a.streamSrv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = a.streamSrv.Stop(shutdownCtx)
	}
	if a.vaultMgr != nil {
		a.vaultMgr.LockVault()
	}
}

// ---------- Vault lifecycle ----------

// UnlockVault authenticates the user's master passphrase, decrypts the local vault (or
// creates a new one on first run), and wires up any previously-saved storage driver settings.
func (a *App) UnlockVault(passphrase string) error {
	if err := a.vaultMgr.UnlockVault(passphrase); err != nil {
		return err
	}
	a.reloadDriversFromSettings()
	return nil
}

// LockVault clears the master key and all decrypted metadata from memory.
func (a *App) LockVault() {
	a.vaultMgr.LockVault()
	a.dispatcher = storage.NewDispatcher(nil, nil)
	a.gdriveDriver = nil
	a.laptopDriver = nil
}

// IsUnlocked reports whether the vault is currently unlocked.
func (a *App) IsUnlocked() bool {
	return a.vaultMgr.IsUnlocked()
}

// ---------- File listing ----------

// ListVaultFiles returns metadata for every file registered in the vault. FileRecord
// deliberately has no plaintext-bearing fields; EncryptedFEK stays wrapped.
func (a *App) ListVaultFiles() ([]vault.FileRecord, error) {
	return a.vaultMgr.ListFileRecords()
}

// ---------- Upload flow ----------

// SelectFileForUpload opens a native file picker and returns the chosen path, or "" if the
// user cancelled.
func (a *App) SelectFileForUpload() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select a file to encrypt and upload",
	})
}

// UploadProgressEvent is emitted on the "upload:progress" Wails event channel while a file is
// being encrypted and dispatched to storage targets.
type UploadProgressEvent struct {
	FileID  string  `json:"fileId"`
	Stage   string  `json:"stage"` // "encrypting" | "uploading" | "done" | "error"
	Percent float64 `json:"percent"`
	Target  string  `json:"target,omitempty"`
	Message string  `json:"message,omitempty"`
}

// EncryptAndUpload encrypts srcPath client-side with a fresh random 256-bit FEK, then
// concurrently dispatches the encrypted container to every target set true in targets (keys:
// "gdrive", "laptop"). Progress streams to the frontend via the "upload:progress" event.
func (a *App) EncryptAndUpload(srcPath string, targets map[string]bool) (vault.FileRecord, error) {
	var record vault.FileRecord

	if !a.vaultMgr.IsUnlocked() {
		return record, errors.New("vault is locked")
	}
	info, err := os.Stat(srcPath)
	if err != nil {
		return record, fmt.Errorf("cannot read source file: %w", err)
	}
	if info.IsDir() {
		return record, errors.New("cannot upload a directory")
	}

	fileID, err := newID()
	if err != nil {
		return record, err
	}

	fek := make([]byte, crypto.KeySize)
	if _, err := rand.Read(fek); err != nil {
		return record, fmt.Errorf("failed to generate file encryption key: %w", err)
	}

	encryptedName := fileID + ".cdenc"
	stagingPath := filepath.Join(a.stagingDir, encryptedName)

	a.emitProgress(fileID, "encrypting", 0, "", "")

	chunkCount, _, err := crypto.EncryptStreamWithProgress(srcPath, stagingPath, fek, func(chunksProcessed int, bytesProcessed int64) {
		pct := 0.0
		if info.Size() > 0 {
			pct = float64(bytesProcessed) / float64(info.Size()) * 100
		}
		a.emitProgress(fileID, "encrypting", pct, "", "")
	})
	if err != nil {
		os.Remove(stagingPath)
		a.emitProgress(fileID, "error", 0, "", err.Error())
		return record, fmt.Errorf("encryption failed: %w", err)
	}

	masterKey, err := a.vaultMgr.GetMasterKey()
	if err != nil {
		os.Remove(stagingPath)
		return record, err
	}
	wrappedFEK, err := crypto.EncryptBlock(masterKey, fek)
	zero(masterKey)
	if err != nil {
		os.Remove(stagingPath)
		return record, fmt.Errorf("failed to wrap file key: %w", err)
	}

	a.emitProgress(fileID, "uploading", 0, "", "")

	statuses := a.dispatcher.DispatchUpload(a.ctx, stagingPath, encryptedName, targets, func(target storage.StorageTarget, sent, total int64) {
		pct := 0.0
		if total > 0 {
			pct = float64(sent) / float64(total) * 100
		}
		a.emitProgress(fileID, "uploading", pct, string(target), "")
	})

	confirmedTargets := map[string]bool{}
	var uploadErrs []string
	for target, status := range statuses {
		confirmedTargets[string(target)] = status.Success
		if !status.Success {
			uploadErrs = append(uploadErrs, fmt.Sprintf("%s: %s", target, status.Error))
		}
	}

	if len(confirmedTargets) == 0 {
		os.Remove(stagingPath)
		return record, errors.New("no storage targets were selected or configured")
	}

	anySucceeded := false
	for _, ok := range confirmedTargets {
		if ok {
			anySucceeded = true
		}
	}
	if !anySucceeded {
		os.Remove(stagingPath)
		errMsg := "all uploads failed: " + joinErrs(uploadErrs)
		a.emitProgress(fileID, "error", 0, "", errMsg)
		return record, errors.New(errMsg)
	}

	record = vault.FileRecord{
		ID:            fileID,
		OriginalName:  filepath.Base(srcPath),
		EncryptedName: encryptedName,
		MimeType:      detectMime(srcPath),
		SizeBytes:     info.Size(),
		ChunkCount:    chunkCount,
		UploadTime:    time.Now(),
		Targets:       confirmedTargets,
		EncryptedFEK:  wrappedFEK,
	}

	if err := a.vaultMgr.AddFileRecord(record); err != nil {
		os.Remove(stagingPath)
		return record, fmt.Errorf("upload succeeded but failed to save vault metadata: %w", err)
	}

	// The local staging copy is kept on disk as a cache for later export/streaming; it is
	// already ciphertext, so retaining it does not weaken the zero-knowledge guarantee.
	if len(uploadErrs) > 0 {
		a.emitProgress(fileID, "done", 100, "", "partial: "+joinErrs(uploadErrs))
	} else {
		a.emitProgress(fileID, "done", 100, "", "")
	}

	return record, nil
}

// ---------- Export / decrypt flow ----------

// DecryptAndExport downloads (if not already staged locally) the encrypted container for
// fileID from whichever storage target has it, decrypts it in full, and writes plaintext to a
// user-chosen destination via a native save dialog. Returns the chosen path.
func (a *App) DecryptAndExport(fileID string) (string, error) {
	if !a.vaultMgr.IsUnlocked() {
		return "", errors.New("vault is locked")
	}

	record, err := a.vaultMgr.GetFileRecord(fileID)
	if err != nil {
		return "", err
	}

	stagingPath := filepath.Join(a.stagingDir, record.EncryptedName)
	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		if err := a.fetchToStaging(record, stagingPath); err != nil {
			return "", err
		}
	}

	destPath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save decrypted file",
		DefaultFilename: record.OriginalName,
	})
	if err != nil {
		return "", err
	}
	if destPath == "" {
		return "", errors.New("export cancelled")
	}

	masterKey, err := a.vaultMgr.GetMasterKey()
	if err != nil {
		return "", err
	}
	fek, err := crypto.DecryptBlock(masterKey, record.EncryptedFEK)
	zero(masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to unwrap file key: %w", err)
	}
	defer zero(fek)

	if err := crypto.DecryptStream(stagingPath, destPath, fek); err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return destPath, nil
}

// fetchToStaging downloads the encrypted container from whichever configured target has it,
// preferring the laptop node (typically faster on LAN) before falling back to Google Drive.
func (a *App) fetchToStaging(record vault.FileRecord, stagingPath string) error {
	var preferred []storage.StorageTarget
	if record.Targets["laptop"] {
		preferred = append(preferred, storage.TargetLaptop)
	}
	if record.Targets["gdrive"] {
		preferred = append(preferred, storage.TargetGDrive)
	}
	if len(preferred) == 0 {
		return errors.New("no storage target has this file")
	}

	_, err := a.dispatcher.DispatchDownload(a.ctx, record.EncryptedName, stagingPath, preferred)
	return err
}

// ---------- Delete ----------

// DeleteVaultFile removes the file's encrypted copies from every configured storage target
// that has it, deletes the local staging cache, and removes the metadata record from the
// vault. Remote-cleanup failures are reported but do not block the local removal.
func (a *App) DeleteVaultFile(fileID string) error {
	if !a.vaultMgr.IsUnlocked() {
		return errors.New("vault is locked")
	}
	record, err := a.vaultMgr.GetFileRecord(fileID)
	if err != nil {
		return err
	}

	var deleteErrs []string
	if record.Targets["gdrive"] && a.gdriveDriver != nil {
		if err := a.gdriveDriver.DeleteFile(a.ctx, record.EncryptedName); err != nil {
			deleteErrs = append(deleteErrs, "gdrive: "+err.Error())
		}
	}
	if record.Targets["laptop"] && a.laptopDriver != nil {
		if err := a.laptopDriver.DeleteFile(a.ctx, record.EncryptedName); err != nil {
			deleteErrs = append(deleteErrs, "laptop: "+err.Error())
		}
	}

	os.Remove(filepath.Join(a.stagingDir, record.EncryptedName))

	if err := a.vaultMgr.RemoveFileRecord(fileID); err != nil {
		return err
	}

	if len(deleteErrs) > 0 {
		return fmt.Errorf("removed from vault, but remote cleanup had errors: %s", joinErrs(deleteErrs))
	}
	return nil
}

// ---------- Streaming ----------

// StreamHandle is returned to the frontend so it can both set a <video>/<audio> src and later
// call StopStream to revoke the session.
type StreamHandle struct {
	URL       string `json:"url"`
	SessionID string `json:"sessionId"`
}

// StartStream ensures the encrypted container is available locally, registers a streaming
// session with the loopback server, and returns a URL suitable for an HTML5 media element.
func (a *App) StartStream(fileID string) (StreamHandle, error) {
	var handle StreamHandle

	if !a.vaultMgr.IsUnlocked() {
		return handle, errors.New("vault is locked")
	}
	if a.streamSrv == nil {
		return handle, errors.New("streaming server is not running")
	}

	record, err := a.vaultMgr.GetFileRecord(fileID)
	if err != nil {
		return handle, err
	}

	stagingPath := filepath.Join(a.stagingDir, record.EncryptedName)
	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		if err := a.fetchToStaging(record, stagingPath); err != nil {
			return handle, err
		}
	}

	masterKey, err := a.vaultMgr.GetMasterKey()
	if err != nil {
		return handle, err
	}
	fek, err := crypto.DecryptBlock(masterKey, record.EncryptedFEK)
	zero(masterKey)
	if err != nil {
		return handle, fmt.Errorf("failed to unwrap file key: %w", err)
	}

	sessionID, err := newID()
	if err != nil {
		return handle, err
	}

	url := a.streamSrv.RegisterSession(sessionID, stagingPath, fek, record.OriginalName, record.SizeBytes)
	return StreamHandle{URL: url, SessionID: sessionID}, nil
}

// StopStream revokes an active streaming session and clears its key material from memory.
func (a *App) StopStream(sessionID string) {
	if a.streamSrv != nil {
		a.streamSrv.UnregisterSession(sessionID)
	}
}

// ---------- Settings ----------

// GetSettings returns the current (decrypted) storage target configuration. This call only
// ever crosses the Go<->JS boundary within the same local desktop process, so returning
// tokens here does not send them over any network.
func (a *App) GetSettings() (vault.AppSettings, error) {
	if !a.vaultMgr.IsUnlocked() {
		return vault.AppSettings{}, errors.New("vault is locked")
	}
	return a.vaultMgr.GetSettings()
}

// SaveLaptopSettings persists the private laptop node's connection details and immediately
// reconfigures the live driver.
func (a *App) SaveLaptopSettings(cfg storage.LaptopNodeConfig, enabled bool) error {
	if !a.vaultMgr.IsUnlocked() {
		return errors.New("vault is locked")
	}
	settings, err := a.vaultMgr.GetSettings()
	if err != nil {
		return err
	}
	settings.Laptop = cfg
	settings.LaptopEnabled = enabled
	if err := a.vaultMgr.SaveSettings(settings); err != nil {
		return err
	}
	a.reloadDriversFromSettings()
	return nil
}

// CheckLaptopHealth pings the configured laptop node.
func (a *App) CheckLaptopHealth() error {
	if a.laptopDriver == nil {
		return errors.New("laptop node is not configured")
	}
	return a.laptopDriver.CheckHealth(a.ctx)
}

// StartGoogleOAuth opens the system browser for Google sign-in and blocks until the flow
// completes, then saves the resulting tokens into the vault and reconfigures the driver.
// clientID/clientSecret come from the user's own Google Cloud OAuth client (Desktop app
// type), entered once in Settings.
func (a *App) StartGoogleOAuth(clientID, clientSecret string) error {
	if !a.vaultMgr.IsUnlocked() {
		return errors.New("vault is locked")
	}

	result, err := storage.RunGoogleOAuthLoopback(a.ctx, clientID, clientSecret, func(authURL string) {
		wailsruntime.BrowserOpenURL(a.ctx, authURL)
	})
	if err != nil {
		return err
	}

	settings, err := a.vaultMgr.GetSettings()
	if err != nil {
		return err
	}
	settings.GDrive.ClientID = clientID
	settings.GDrive.ClientSecret = clientSecret
	settings.GDrive.AccessToken = result.AccessToken
	settings.GDrive.RefreshToken = result.RefreshToken
	settings.GDriveTokenExp = result.ExpiresAt.Unix()
	settings.GDriveEnabled = true

	if err := a.vaultMgr.SaveSettings(settings); err != nil {
		return err
	}
	a.reloadDriversFromSettings()
	return nil
}

// SaveGDriveTargetFolder updates just the destination Drive folder ID without repeating OAuth.
func (a *App) SaveGDriveTargetFolder(folderID string) error {
	settings, err := a.vaultMgr.GetSettings()
	if err != nil {
		return err
	}
	settings.GDrive.TargetFolder = folderID
	if err := a.vaultMgr.SaveSettings(settings); err != nil {
		return err
	}
	a.reloadDriversFromSettings()
	return nil
}

// CheckGDriveHealth verifies the current Google Drive connection.
func (a *App) CheckGDriveHealth() error {
	if a.gdriveDriver == nil {
		return errors.New("google drive is not configured")
	}
	return a.gdriveDriver.CheckHealth(a.ctx)
}

// HardenVaultKeyStorage stores an OS-protected (Windows DPAPI / macOS Keychain) backup copy
// of the vault salt as defense-in-depth. Safe to call repeatedly; the vault remains fully
// unlockable with just the passphrase even without this.
func (a *App) HardenVaultKeyStorage() error {
	return a.vaultMgr.HardenWithOSKeyProtection()
}

// IsOSKeyProtectionAvailable reports whether HardenVaultKeyStorage can do anything useful on
// this platform (true on Windows and macOS, false on Linux).
func (a *App) IsOSKeyProtectionAvailable() bool {
	return vault.IsPlatformKeyProtectionAvailable()
}

// reloadDriversFromSettings rebuilds gdriveDriver/laptopDriver/dispatcher from whatever is
// currently saved in the vault. Called after unlock and after any settings change.
//
// NOTE on a subtle Go gotcha this deliberately avoids: storage.NewDispatcher takes two
// StorageDriver *interface* values. Passing a typed-nil *GDriveDriver/*LaptopDriver pointer
// as that interface produces a NON-nil interface wrapping a nil pointer, so a naive
// `if driver != nil` check inside the dispatcher would wrongly treat an unconfigured driver
// as configured and panic on first use. We build gdriveIface/laptopIface explicitly so an
// unconfigured target stays a true nil interface.
func (a *App) reloadDriversFromSettings() {
	settings, err := a.vaultMgr.GetSettings()
	if err != nil {
		return
	}

	var gdriveDriver *storage.GDriveDriver
	var laptopDriver *storage.LaptopDriver
	var gdriveIface storage.StorageDriver
	var laptopIface storage.StorageDriver

	if settings.GDriveEnabled && settings.GDrive.AccessToken != "" {
		cfg := settings.GDrive
		cfg.ExpiresAt = time.Unix(settings.GDriveTokenExp, 0)
		gdriveDriver = storage.NewGDriveDriver(cfg)
		gdriveDriver.SetTokenRefreshHandler(func(access, refresh string, expiresAt time.Time) {
			s, err := a.vaultMgr.GetSettings()
			if err != nil {
				return
			}
			s.GDrive.AccessToken = access
			s.GDrive.RefreshToken = refresh
			s.GDriveTokenExp = expiresAt.Unix()
			_ = a.vaultMgr.SaveSettings(s)
		})
		gdriveIface = gdriveDriver
	}

	if settings.LaptopEnabled && settings.Laptop.BaseURL != "" {
		laptopDriver = storage.NewLaptopDriver(settings.Laptop)
		laptopIface = laptopDriver
	}

	a.gdriveDriver = gdriveDriver
	a.laptopDriver = laptopDriver
	a.dispatcher = storage.NewDispatcher(gdriveIface, laptopIface)
}

// ---------- small helpers ----------

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func joinErrs(errs []string) string {
	out := ""
	for i, e := range errs {
		if i > 0 {
			out += "; "
		}
		out += e
	}
	return out
}

func detectMime(path string) string {
	switch filepath.Ext(path) {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

func (a *App) emitProgress(fileID, stage string, percent float64, target, message string) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "upload:progress", UploadProgressEvent{
		FileID:  fileID,
		Stage:   stage,
		Percent: percent,
		Target:  target,
		Message: message,
	})
}
