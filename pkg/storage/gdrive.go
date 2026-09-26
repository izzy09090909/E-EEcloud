package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// GDriveConfig stores OAuth2 credentials and folder targets for Google Drive
type GDriveConfig struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	TargetFolder string    `json:"target_folder_id,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// GDriveDriver implements StorageDriver for Google Drive using Resumable Upload v3 API
type GDriveDriver struct {
	mu             sync.RWMutex
	config         GDriveConfig
	httpClient     *http.Client
	onTokenRefresh func(access, refresh string, expiresAt time.Time)
}

// SetTokenRefreshHandler registers a callback invoked whenever this driver silently refreshes
// its Google access token, so the caller (app.go) can persist the new token into the vault
// instead of it only living in this in-memory driver.
func (g *GDriveDriver) SetTokenRefreshHandler(fn func(access, refresh string, expiresAt time.Time)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.onTokenRefresh = fn
}

// ensureValidToken refreshes the access token if it is missing or within 2 minutes of expiry.
// Google access tokens are short-lived (~1 hour), so long uploads/streaming sessions need
// this checked before every request rather than once at driver construction.
func (g *GDriveDriver) ensureValidToken(ctx context.Context) error {
	g.mu.RLock()
	needsRefresh := g.config.AccessToken == "" || time.Now().Add(2*time.Minute).After(g.config.ExpiresAt)
	refreshToken := g.config.RefreshToken
	clientID := g.config.ClientID
	clientSecret := g.config.ClientSecret
	g.mu.RUnlock()

	if !needsRefresh {
		return nil
	}
	if refreshToken == "" {
		return errors.New("google drive access token expired and no refresh token is stored; reconnect Google Drive in Settings")
	}

	result, err := RefreshGoogleAccessToken(ctx, clientID, clientSecret, refreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh google access token: %w", err)
	}
	if result.RefreshToken == "" {
		result.RefreshToken = refreshToken // Google omits refresh_token on refresh responses
	}

	g.mu.Lock()
	g.config.AccessToken = result.AccessToken
	g.config.RefreshToken = result.RefreshToken
	g.config.ExpiresAt = result.ExpiresAt
	handler := g.onTokenRefresh
	g.mu.Unlock()

	if handler != nil {
		handler(result.AccessToken, result.RefreshToken, result.ExpiresAt)
	}
	return nil
}

// NewGDriveDriver constructs an authenticated Google Drive driver
func NewGDriveDriver(cfg GDriveConfig) *GDriveDriver {
	return &GDriveDriver{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 0, // Infinite timeout for long chunked stream uploads
		},
	}
}


// CheckHealth verifies connection to Google Drive API endpoint
func (g *GDriveDriver) CheckHealth(ctx context.Context) error {
	if err := g.ensureValidToken(ctx); err != nil {
		return err
	}
	g.mu.RLock()
	token := g.config.AccessToken
	g.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/drive/v3/about?fields=user", nil)
	if err != nil {
		return fmt.Errorf("failed to build health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gdrive health check connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gdrive health check failed with HTTP status %d", resp.StatusCode)
	}

	return nil
}


// UploadChunkStream uploads an encrypted binary stream using Google Drive Resumable Upload
func (g *GDriveDriver) UploadChunkStream(ctx context.Context, remoteName string, r io.Reader, size int64) error {
	if err := g.ensureValidToken(ctx); err != nil {
		return err
	}
	g.mu.RLock()
	token := g.config.AccessToken
	folderID := g.config.TargetFolder
	g.mu.RUnlock()

	// Step 1: Initiate Resumable Upload Session
	meta := map[string]interface{}{
		"name":     remoteName,
		"mimeType": "application/octet-stream",
	}
	if folderID != "" {
		meta["parents"] = []string{folderID}
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	sessionURL := "https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable"
	initReq, err := http.NewRequestWithContext(ctx, "POST", sessionURL, bytes.NewReader(metaBytes))
	if err != nil {
		return fmt.Errorf("failed to create resumable session request: %w", err)
	}

	initReq.Header.Set("Authorization", "Bearer "+token)
	initReq.Header.Set("Content-Type", "application/json; charset=UTF-8")
	initReq.Header.Set("X-Upload-Content-Type", "application/octet-stream")
	initReq.Header.Set("X-Upload-Content-Length", strconv.FormatInt(size, 10))

	initResp, err := g.httpClient.Do(initReq)
	if err != nil {
		return fmt.Errorf("resumable upload session init failed: %w", err)
	}
	defer initResp.Body.Close()

	if initResp.StatusCode != http.StatusOK {
		return fmt.Errorf("gdrive upload init returned HTTP %d", initResp.StatusCode)
	}

	uploadLocation := initResp.Header.Get("Location")
	if uploadLocation == "" {
		return fmt.Errorf("missing upload session location header from gdrive")
	}


	// Step 2: Stream encrypted payload to the upload location
	uploadReq, err := http.NewRequestWithContext(ctx, "PUT", uploadLocation, r)
	if err != nil {
		return fmt.Errorf("failed to create upload stream request: %w", err)
	}

	uploadReq.Header.Set("Content-Length", strconv.FormatInt(size, 10))
	uploadReq.Header.Set("Content-Type", "application/octet-stream")

	uploadResp, err := g.httpClient.Do(uploadReq)
	if err != nil {
		return fmt.Errorf("failed during byte stream upload to gdrive: %w", err)
	}
	defer uploadResp.Body.Close()

	if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("gdrive upload failed with status %d", uploadResp.StatusCode)
	}

	return nil
}


// DownloadChunkStream streams remote file contents back to an io.Writer
func (g *GDriveDriver) DownloadChunkStream(ctx context.Context, remoteName string, w io.Writer) error {
	if err := g.ensureValidToken(ctx); err != nil {
		return err
	}
	g.mu.RLock()
	token := g.config.AccessToken
	g.mu.RUnlock()

	// Resolve file ID from remote name
	queryURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=name='%s'+and+trashed=false", remoteName)
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to query gdrive file: %w", err)
	}
	defer resp.Body.Close()

	var searchRes struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil || len(searchRes.Files) == 0 {
		return fmt.Errorf("file '%s' not found on Google Drive", remoteName)
	}

	fileID := searchRes.Files[0].ID
	downloadURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", fileID)

	dlReq, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err
	}
	dlReq.Header.Set("Authorization", "Bearer "+token)

	dlResp, err := g.httpClient.Do(dlReq)
	if err != nil {
		return fmt.Errorf("gdrive download request failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("gdrive media download failed with status %d", dlResp.StatusCode)
	}

	_, err = io.Copy(w, dlResp.Body)
	return err
}

// DeleteFile removes an encrypted file from Google Drive. Previously a no-op stub that
// silently returned nil without deleting anything -- now resolves the file ID by name (as
// DownloadChunkStream does) and issues a real DELETE request.
func (g *GDriveDriver) DeleteFile(ctx context.Context, remoteName string) error {
	if err := g.ensureValidToken(ctx); err != nil {
		return err
	}
	g.mu.RLock()
	token := g.config.AccessToken
	g.mu.RUnlock()

	queryURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=name='%s'+and+trashed=false", remoteName)
	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to locate file for deletion: %w", err)
	}
	defer resp.Body.Close()

	var searchRes struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil || len(searchRes.Files) == 0 {
		return fmt.Errorf("file '%s' not found on Google Drive (nothing to delete)", remoteName)
	}

	fileID := searchRes.Files[0].ID
	delURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", fileID)
	delReq, err := http.NewRequestWithContext(ctx, "DELETE", delURL, nil)
	if err != nil {
		return err
	}
	delReq.Header.Set("Authorization", "Bearer "+token)

	delResp, err := g.httpClient.Do(delReq)
	if err != nil {
		return fmt.Errorf("gdrive delete request failed: %w", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("gdrive delete failed with HTTP %d", delResp.StatusCode)
	}
	return nil
}