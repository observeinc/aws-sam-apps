package seekable

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const defaultBufferSize = 32 * 1024

// FromReader converts a reader into a rewindable stream.
// It keeps data in memory up to memoryLimitBytes, then spills to /tmp.
func FromReader(r io.Reader, memoryLimitBytes int64) (io.ReadSeeker, func() error, error) {
	if r == nil {
		return nil, func() error { return nil }, nil
	}

	if rs, ok := r.(io.ReadSeeker); ok {
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, nil, fmt.Errorf("failed to seek existing reader: %w", err)
		}
		return rs, func() error { return nil }, nil
	}

	if memoryLimitBytes < 0 {
		memoryLimitBytes = 0
	}

	buf := bytes.NewBuffer(nil)
	chunk := make([]byte, defaultBufferSize)
	var (
		written int64
		tmpFile *os.File
	)

	for {
		n, readErr := r.Read(chunk)
		if n > 0 {
			data := chunk[:n]
			if tmpFile == nil && written+int64(n) <= memoryLimitBytes {
				if _, err := buf.Write(data); err != nil {
					return nil, nil, fmt.Errorf("failed to write to memory buffer: %w", err)
				}
			} else {
				if tmpFile == nil {
					f, err := os.CreateTemp("", "forwarder-seekable-*")
					if err != nil {
						return nil, nil, fmt.Errorf("failed to create temp file: %w", err)
					}
					tmpFile = f
					if _, err := tmpFile.Write(buf.Bytes()); err != nil {
						path := tmpFile.Name()
						_ = tmpFile.Close()
						_ = os.Remove(path)
						return nil, nil, fmt.Errorf("failed to spill buffered data to temp file: %w", err)
					}
				}
				if _, err := tmpFile.Write(data); err != nil {
					path := tmpFile.Name()
					_ = tmpFile.Close()
					_ = os.Remove(path)
					return nil, nil, fmt.Errorf("failed to write to temp file: %w", err)
				}
			}
			written += int64(n)
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if tmpFile != nil {
				path := tmpFile.Name()
				_ = tmpFile.Close()
				_ = os.Remove(path)
			}
			return nil, nil, fmt.Errorf("failed to read input stream: %w", readErr)
		}
	}

	if tmpFile == nil {
		return bytes.NewReader(buf.Bytes()), func() error { return nil }, nil
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		path := tmpFile.Name()
		_ = tmpFile.Close()
		_ = os.Remove(path)
		return nil, nil, fmt.Errorf("failed to rewind temp file: %w", err)
	}

	cleanup := func() error {
		path := tmpFile.Name()
		closeErr := tmpFile.Close()
		removeErr := os.Remove(path)
		if closeErr != nil {
			return fmt.Errorf("failed to close temp file: %w", closeErr)
		}
		if removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("failed to remove temp file: %w", removeErr)
		}
		return nil
	}

	return tmpFile, cleanup, nil
}
