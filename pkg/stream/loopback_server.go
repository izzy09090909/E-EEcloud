package stream

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cipherdrive/pkg/crypto"
)

// StreamSession holds active session metadata for serving encrypted media on demand
type StreamSession struct {
	SessionID         string
	EncryptedFilePath string
	FEK               []byte
	TotalSize         int64
	MimeType          string
	CreatedAt         time.Time
}

// LoopbackServer manages the local HTTP server providing on-the-fly decryption streaming
type LoopbackServer struct {
	mu       sync.RWMutex
	listener net.Listener
	server   *http.Server
	sessions map[string]*StreamSession
	port     int
}


// NewLoopbackServer creates a new local-only HTTP streaming server bound to 127.0.0.1
func NewLoopbackServer() (*LoopbackServer, error) {
	// Bind to 127.0.0.1 on a random available port (:0) for security
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind loopback streaming socket: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	s := &LoopbackServer{
		listener: listener,
		sessions: make(map[string]*StreamSession),
		port:     port,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStreamRequest)
	mux.HandleFunc("/health", s.handleHealthCheck)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return s, nil
}

// Start launches the background HTTP server goroutine
func (s *LoopbackServer) Start() {
	go func() {
		if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("[StreamServer] Error serving loopback traffic: %v\n", err)
		}
	}()
}

// Stop gracefully shuts down the loopback streaming server
func (s *LoopbackServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// GetPort returns the dynamically assigned port number
func (s *LoopbackServer) GetPort() int {
	return s.port
}


// RegisterSession registers an encrypted file with its File Encryption Key (FEK) for playback
func (s *LoopbackServer) RegisterSession(sessionID, encryptedFilePath string, fek []byte, originalFilename string, totalSize int64) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Detect MIME type based on original extension
	ext := filepath.Ext(originalFilename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		switch strings.ToLower(ext) {
		case ".mp4":
			mimeType = "video/mp4"
		case ".webm":
			mimeType = "video/webm"
		case ".mp3":
			mimeType = "audio/mpeg"
		case ".pdf":
			mimeType = "application/pdf"
		default:
			mimeType = "application/octet-stream"
		}
	}

	session := &StreamSession{
		SessionID:         sessionID,
		EncryptedFilePath: encryptedFilePath,
		FEK:               fek,
		TotalSize:         totalSize,
		MimeType:          mimeType,
		CreatedAt:         time.Now(),
	}

	s.sessions[sessionID] = session

	// Returns the loopback streaming URL to be passed directly to HTML5 video/audio elements
	return fmt.Sprintf("http://127.0.0.1:%d/stream?session_id=%s", s.port, sessionID)
}

// UnregisterSession revokes access and removes session metadata from RAM
func (s *LoopbackServer) UnregisterSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *LoopbackServer) getSession(sessionID string) (*StreamSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, exists := s.sessions[sessionID]
	return sess, exists
}


// handleStreamRequest processes incoming HTTP Byte-Range requests from video players
func (s *LoopbackServer) handleStreamRequest(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id parameter", http.StatusBadRequest)
		return
	}

	session, exists := s.getSession(sessionID)
	if !exists {
		http.Error(w, "Invalid or expired streaming session", http.StatusNotFound)
		return
	}

	// Support CORS for local webview calls
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", session.MimeType)

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// Serve full stream (or initial byte headers) if range header not sent
		w.Header().Set("Content-Length", strconv.FormatInt(session.TotalSize, 10))
		w.WriteHeader(http.StatusOK)

		s.serveByteRange(w, session, 0, session.TotalSize-1)
		return
	}

	// Parse HTTP Range Header (Format: "bytes=start-end")
	start, end, err := parseRangeHeader(rangeHeader, session.TotalSize)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", session.TotalSize))
		http.Error(w, "Requested range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	contentLength := (end - start) + 1
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, session.TotalSize))
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.WriteHeader(http.StatusPartialContent)

	s.serveByteRange(w, session, start, end)
}


// serveByteRange converts absolute byte offsets to 5MB encrypted chunk indices
func (s *LoopbackServer) serveByteRange(w http.ResponseWriter, session *StreamSession, start, end int64) {
	startChunkIndex := int(start / crypto.ChunkSize)
	endChunkIndex := int(end / crypto.ChunkSize)

	var bytesWritten int64 = 0
	targetBytesNeeded := (end - start) + 1

	for chunkIdx := startChunkIndex; chunkIdx <= endChunkIndex; chunkIdx++ {
		// Seek and decrypt single 5MB chunk directly into RAM memory
		plaintextChunk, err := crypto.DecryptChunkAt(session.EncryptedFilePath, chunkIdx, session.FEK)
		if err != nil {
			fmt.Printf("[StreamServer] Decryption error at chunk %d: %v\n", chunkIdx, err)
			return
		}

		chunkStartOffset := int64(chunkIdx) * crypto.ChunkSize

		// Compute slice start inside this specific decrypted chunk
		sliceStart := int64(0)
		if start > chunkStartOffset {
			sliceStart = start - chunkStartOffset
		}

		// Compute slice end inside this specific decrypted chunk
		sliceEnd := int64(len(plaintextChunk))
		chunkEndOffset := chunkStartOffset + int64(len(plaintextChunk)) - 1

		if end < chunkEndOffset {
			sliceEnd = sliceStart + (targetBytesNeeded - bytesWritten)
			if sliceEnd > int64(len(plaintextChunk)) {
				sliceEnd = int64(len(plaintextChunk))
			}
		}

		if sliceStart < 0 || sliceStart >= int64(len(plaintextChunk)) || sliceStart >= sliceEnd {
			continue
		}

		// Write unencrypted slice chunk payload to HTTP Response
		slice := plaintextChunk[sliceStart:sliceEnd]
		n, err := w.Write(slice)
		if err != nil {
			// Client disconnected early (e.g. seeked away)
			return
		}

		bytesWritten += int64(n)
		if bytesWritten >= targetBytesNeeded {
			break
		}
	}
}

// handleHealthCheck returns simple diagnostic response
func (s *LoopbackServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("CipherDrive Loopback Streamer OK"))
}


// parseRangeHeader parses standard HTTP Range headers like "bytes=0-1048575"
func parseRangeHeader(rangeHeader string, totalSize int64) (int64, int64, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, errors.New("invalid range prefix")
	}

	bytesSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(bytesSpec, "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid range format")
	}

	var start, end int64
	var err error

	if parts[0] == "" {
		// Range: bytes=-500 (suffix bytes)
		suffixLen, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		start = totalSize - suffixLen
		end = totalSize - 1
	} else if parts[1] == "" {
		// Range: bytes=500- (from offset to EOF)
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		end = totalSize - 1
	} else {
		// Range: bytes=500-1000
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}

	if start < 0 || start >= totalSize || end < start {
		return 0, 0, errors.New("range out of bounds")
	}

	if end >= totalSize {
		end = totalSize - 1
	}

	return start, end, nil
}