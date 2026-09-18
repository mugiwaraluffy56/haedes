package archive

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const MaxArchiveBytes int64 = 100 * 1024 * 1024

var ErrTooLarge = errors.New("snapshot archive exceeds the configured limit")

func Digest(reader io.Reader) (int64, string, error) {
	hash := sha256.New()
	var size int64
	buffer := make([]byte, 32*1024)
	for {
		read, err := reader.Read(buffer)
		if read > 0 {
			size += int64(read)
			if size > MaxArchiveBytes {
				return 0, "", ErrTooLarge
			}
			if _, writeErr := hash.Write(buffer[:read]); writeErr != nil {
				return 0, "", fmt.Errorf("hash snapshot archive: %w", writeErr)
			}
		}
		if err == io.EOF {
			return size, hex.EncodeToString(hash.Sum(nil)), nil
		}
		if err != nil {
			return 0, "", fmt.Errorf("read snapshot archive: %w", err)
		}
	}
}
