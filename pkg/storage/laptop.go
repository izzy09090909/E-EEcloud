package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// LaptopNodeConfig stores mTLS & authorization settings for secondary target server
type LaptopNodeConfig struct {
	BaseURL    string `json:"base_url"`
	AuthToken  string `json:"auth_token"`
	SkipTLSVerify bool `json:"skip_tls_verify"`
}

// LaptopDriver implements StorageDriver for self-hosted HTTPS / mTLS remote laptop nodes
type LaptopDriver struct {
	config     LaptopNodeConfig
	httpClient *http.Client
}

// NewLaptopDriver constructs a client connected to secondary private server node
func NewLaptopDriver(cfg LaptopNodeConfig) *LaptopDriver {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.SkipTLSVerify,
		},
		MaxIdleConnsPerHost: 10,
	}

	return &LaptopDriver{
		config: cfg,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   0,
		},
	}
}


// CheckHealth verifies reachability of secondary laptop node
func (l *LaptopDriver) CheckHealth(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/api/v1/health", l.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+l.config.AuthToken)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("laptop node unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("laptop node health check returned HTTP %d", resp.StatusCode)
	}

	return nil
}


// UploadChunkStream streams binary file container directly to secondary laptop storage server
func (l *LaptopDriver) UploadChunkStream(ctx context.Context, remoteName string, r io.Reader, size int64) error {
	uploadURL := fmt.Sprintf("%s/api/v1/storage/upload?name=%s", l.config.BaseURL, url.QueryEscape(remoteName))
	
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, r)
	if err != nil {
		return fmt.Errorf("failed to build upload request for laptop node: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+l.config.AuthToken)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", strconv.FormatInt(size, 10))

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stream encrypted data to laptop node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("laptop node upload failed with HTTP %d", resp.StatusCode)
	}

	return nil
}

// DownloadChunkStream pulls encrypted stream from laptop node
func (l *LaptopDriver) DownloadChunkStream(ctx context.Context, remoteName string, w io.Writer) error {
	downloadURL := fmt.Sprintf("%s/api/v1/storage/download?name=%s", l.config.BaseURL, url.QueryEscape(remoteName))
	
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+l.config.AuthToken)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("laptop node download request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("laptop download failed with HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

// DeleteFile deletes remote encrypted payload on laptop node
func (l *LaptopDriver) DeleteFile(ctx context.Context, remoteName string) error {
	deleteURL := fmt.Sprintf("%s/api/v1/storage/delete?name=%s", l.config.BaseURL, url.QueryEscape(remoteName))
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+l.config.AuthToken)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("laptop delete error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("laptop delete returned HTTP %d", resp.StatusCode)
	}

	return nil
}