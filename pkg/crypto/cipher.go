package crypto


import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// ChunkSize defines 5 MB streaming blocks for memory efficiency
	ChunkSize = 5 * 1024 * 1024
	// NonceSize defines standard 12-byte IV for AES-GCM
	NonceSize = 12
	// TagSize defines standard 16-byte GCM authentication tag
	TagSize = 16
	// FrameHeaderSize defines nonce (12B) + length prefix (4B)
	FrameHeaderSize = NonceSize + 4
)

// ProgressCallback function signature for tracking streaming progress in real-time
type ProgressCallback func(chunksProcessed int, bytesProcessed int64)

// EncryptStreamWithProgress reads a source file and writes AES-256-GCM encrypted chunks with real-time progress callbacks
func EncryptStreamWithProgress(srcPath, dstPath string, fek []byte, onProgress ProgressCallback) (int, int64, error) {
	if len(fek) != KeySize {
		return 0, 0, errors.New("invalid key size for AES-256")
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to create destination file: %w", err)
	}
	defer dstFile.Close()

	block, err := aes.NewCipher(fek)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create GCM AEAD: %w", err)
	}

	buffer := make([]byte, ChunkSize)
	chunkCount := 0
	var totalBytesEncrypted int64 = 0

	for {
		bytesRead, readErr := srcFile.Read(buffer)
		if bytesRead > 0 {
			chunkCount++

			// Generate fresh random 12-byte nonce for every chunk to prevent nonce reuse attacks
			nonce := make([]byte, NonceSize)
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
				return 0, 0, fmt.Errorf("failed to generate chunk nonce: %w", err)
			}

			// Encrypt chunk payload using AES-256-GCM
			ciphertext := gcm.Seal(nil, nonce, buffer[:bytesRead], nil)
			cipherLen := uint32(len(ciphertext))

			// Write Chunk Frame: [12-byte Nonce] [4-byte Length] [Ciphertext]
			if _, err := dstFile.Write(nonce); err != nil {
				return 0, 0, fmt.Errorf("failed to write nonce to stream: %w", err)
			}
			if err := binary.Write(dstFile, binary.BigEndian, cipherLen); err != nil {
				return 0, 0, fmt.Errorf("failed to write chunk header length: %w", err)
			}
			if _, err := dstFile.Write(ciphertext); err != nil {
				return 0, 0, fmt.Errorf("failed to write chunk payload: %w", err)
			}

			totalBytesEncrypted += int64(bytesRead)

			if onProgress != nil {
				onProgress(chunkCount, totalBytesEncrypted)
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return 0, 0, fmt.Errorf("error reading source file stream: %w", readErr)
		}
	}

	return chunkCount, totalBytesEncrypted, nil
}

// EncryptStream wraps EncryptStreamWithProgress for standard synchronous calls
func EncryptStream(srcPath, dstPath string, fek []byte) (int, int64, error) {
	return EncryptStreamWithProgress(srcPath, dstPath, fek, nil)
}

// DecryptChunkAt seeks directly to a specific 0-indexed chunk in the encrypted container and decrypts it into RAM.
// Used by the local loopback HTTP server (pkg/stream) to handle HTTP Byte-Range requests for seamless media playback.
func DecryptChunkAt(srcPath string, chunkIndex int, fek []byte) ([]byte, error) {
	if len(fek) != KeySize {
		return nil, errors.New("invalid key size for AES-256")
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open encrypted source file: %w", err)
	}
	defer srcFile.Close()

	block, err := aes.NewCipher(fek)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Calculate offset for standard 5MB chunk frame (Nonce + LengthPrefix + Payload + Tag)
	standardFrameSize := int64(FrameHeaderSize + ChunkSize + TagSize)
	targetOffset := int64(chunkIndex) * standardFrameSize

	if _, err := srcFile.Seek(targetOffset, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to chunk index %d at offset %d: %w", chunkIndex, targetOffset, err)
	}

	// Read 12-byte Nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(srcFile, nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce at chunk %d: %w", chunkIndex, err)
	}

	// Read 4-byte BigEndian length header
	var cipherLen uint32
	if err := binary.Read(srcFile, binary.BigEndian, &cipherLen); err != nil {
		return nil, fmt.Errorf("failed to read length header at chunk %d: %w", chunkIndex, err)
	}

	// Read Ciphertext Payload
	ciphertext := make([]byte, cipherLen)
	if _, err := io.ReadFull(srcFile, ciphertext); err != nil {
		return nil, fmt.Errorf("failed to read ciphertext payload at chunk %d: %w", chunkIndex, err)
	}

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("cryptographic authentication failed for chunk: invalid key or tampered payload")
	}

	return plaintext, nil
}

// DecryptStream reads an encrypted chunked container and writes unencrypted plaintext to destination path
func DecryptStream(srcPath, dstPath string, fek []byte) error {
	if len(fek) != KeySize {
		return errors.New("invalid key size for AES-256")
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("unable to open encrypted file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("unable to create target export file: %w", err)
	}
	defer dstFile.Close()

	block, err := aes.NewCipher(fek)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM AEAD: %w", err)
	}

	for {
		// Read 12-byte Nonce
		nonce := make([]byte, NonceSize)
		_, err := io.ReadFull(srcFile, nonce)
		if err == io.EOF {
			break // Reached end of stream successfully
		}
		if err != nil {
			return fmt.Errorf("corrupted chunk header (nonce read error): %w", err)
		}

		// Read 4-byte BigEndian length header
		var cipherLen uint32
		if err := binary.Read(srcFile, binary.BigEndian, &cipherLen); err != nil {
			return fmt.Errorf("corrupted chunk header (length read error): %w", err)
		}

		// Read ciphertext payload
		ciphertext := make([]byte, cipherLen)
		if _, err := io.ReadFull(srcFile, ciphertext); err != nil {
			return fmt.Errorf("truncated chunk ciphertext payload: %w", err)
		}

		// Decrypt and verify authentication tag
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return errors.New("cryptographic authentication failed: invalid key or tampered ciphertext")
		}

		// Write unencrypted plaintext chunk to output file
		if _, err := dstFile.Write(plaintext); err != nil {
			return fmt.Errorf("error writing decrypted plaintext to disk: %w", err)
		}
	}

	return nil
}


// EncryptBlock encrypts a small byte slice using AES-256-GCM (used for wrapping FEKs or vault tokens)
func EncryptBlock(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptBlock decrypts a small byte slice using AES-256-GCM
func DecryptBlock(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext block too short")
	}
	nonce, cipherPayload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, cipherPayload, nil)
}