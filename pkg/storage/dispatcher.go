package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// StorageTarget defines supported cloud & local remote target backends
type StorageTarget string

const (
	TargetGDrive StorageTarget = "gdrive"
	TargetLaptop StorageTarget = "laptop"
)

// TargetStatus tracks operational health and upload progress for a target
type TargetStatus struct {
	Target        StorageTarget `json:"target"`
	Success       bool          `json:"success"`
	BytesSent     int64         `json:"bytes_sent"`
	Error         string        `json:"error,omitempty"`
	DurationMs    int64         `json:"duration_ms"`
}

// StorageDriver defines the contract required for cloud storage clients
type StorageDriver interface {
	UploadChunkStream(ctx context.Context, remoteName string, r io.Reader, size int64) error
	DownloadChunkStream(ctx context.Context, remoteName string, w io.Writer) error
	DeleteFile(ctx context.Context, remoteName string) error
	CheckHealth(ctx context.Context) error
}

// Dispatcher ProgressCallback reports concurrent dispatch status to Wails UI
type ProgressCallback func(target StorageTarget, bytesTransferred int64, totalBytes int64)

// Dispatcher coordinates multi-cloud concurrent transfers with fault tolerance
type Dispatcher struct {
	mu      sync.RWMutex
	drivers map[StorageTarget]StorageDriver
}

// NewDispatcher constructs a storage manager for handling dual-destination uploads
func NewDispatcher(gdriveDriver, laptopDriver StorageDriver) *Dispatcher {
	drivers := make(map[StorageTarget]StorageDriver)
	if gdriveDriver != nil {
		drivers[TargetGDrive] = gdriveDriver
	}
	if laptopDriver != nil {
		drivers[TargetLaptop] = laptopDriver
	}

	return &Dispatcher{
		drivers: drivers,
	}
}

// DispatchUpload streams an encrypted staging file to all requested storage targets concurrently
func (d *Dispatcher) DispatchUpload(ctx context.Context, encryptedFilePath, remoteName string, targets map[string]bool, onProgress ProgressCallback) map[StorageTarget]*TargetStatus {
	results := make(map[StorageTarget]*TargetStatus)
	var wg sync.WaitGroup
	var resultsMu sync.Mutex

	fileInfo, err := os.Stat(encryptedFilePath)
	if err != nil {
		results[TargetGDrive] = &TargetStatus{
			Target: TargetGDrive,
			Success: false,
			Error: fmt.Sprintf("failed to read staging file: %v", err),
		}
		return results
	}

	totalSize := fileInfo.Size()

	for targetKey, enabled := range targets {
		target := StorageTarget(targetKey)
		if !enabled {
			continue
		}

		d.mu.RLock()
		driver, exists := d.drivers[target]
		d.mu.RUnlock()

		if !exists || driver == nil {
			resultsMu.Lock()
			results[target] = &TargetStatus{
				Target:  target,
				Success: false,
				Error:   fmt.Sprintf("driver for target '%s' is not configured", target),
			}
			resultsMu.Unlock()
			continue
		}

		wg.Add(1)
		go func(t StorageTarget, drv StorageDriver) {
			defer wg.Done()
			startTime := time.Now()

			file, err := os.Open(encryptedFilePath)
			if err != nil {
				resultsMu.Lock()
				results[t] = &TargetStatus{
					Target:     t,
					Success:    false,
					Error:      fmt.Sprintf("failed to open source file: %v", err),
					DurationMs: time.Since(startTime).Milliseconds(),
				}
				resultsMu.Unlock()
				return
			}
			defer file.Close()

			// Wrap reader in progress monitor wrapper
			progressReader := &progressTrackingReader{
				reader: file,
				onRead: func(n int64) {
					if onProgress != nil {
						onProgress(t, n, totalSize)
					}
				},
			}

			// Execute upload on backend storage driver
			err = drv.UploadChunkStream(ctx, remoteName, progressReader, totalSize)
			duration := time.Since(startTime).Milliseconds()

			status := &TargetStatus{
				Target:     t,
				Success:    err == nil,
				BytesSent:  progressReader.totalRead,
				DurationMs: duration,
			}
			if err != nil {
				status.Error = err.Error()
			}

			resultsMu.Lock()
			results[t] = status
			resultsMu.Unlock()
		}(target, driver)
	}

	wg.Wait()
	return results
}

// DispatchDownload attempts downloading an encrypted payload with failover across active targets
func (d *Dispatcher) DispatchDownload(ctx context.Context, remoteName, localDstPath string, preferredTargets []StorageTarget) (StorageTarget, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var lastErr error
	for _, target := range preferredTargets {
		driver, exists := d.drivers[target]
		if !exists || driver == nil {
			continue
		}

		outPath := localDstPath + ".tmp"
		outFile, err := os.Create(outPath)
		if err != nil {
			return "", fmt.Errorf("failed to create temporary download target: %w", err)
		}

		err = driver.DownloadChunkStream(ctx, remoteName, outFile)
		outFile.Close()

		if err == nil {
			if err := os.Rename(outPath, localDstPath); err != nil {
				return "", fmt.Errorf("failed to finalize downloaded file: %w", err)
			}
			return target, nil
		}

		os.Remove(outPath)
		lastErr = fmt.Errorf("target '%s' download failed: %w", target, err)
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no valid storage targets available for download")
}

type progressTrackingReader struct {
	reader    io.Reader
	totalRead int64
	onRead    func(total int64)
}

func (p *progressTrackingReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	if n > 0 {
		p.totalRead += int64(n)
		if p.onRead != nil {
			p.onRead(p.totalRead)
		}
	}
	return n, err
}