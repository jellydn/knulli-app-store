package safefs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".write-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(mode.Perm()); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func Copy(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	// Replacing the inode keeps a running executable alive and leaves the old
	// destination intact if copying fails before the rename.
	output, err := os.CreateTemp(filepath.Dir(destination), ".write-*")
	if err != nil {
		return err
	}
	name := output.Name()
	defer os.Remove(name)
	if err := output.Chmod(mode.Perm()); err != nil {
		output.Close()
		return err
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Rename(name, destination); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(destination))
}

func SHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func AvailableBytes(path string) (uint64, error) {
	existing := path
	for {
		var stats syscall.Statfs_t
		if err := syscall.Statfs(existing, &stats); err == nil {
			return stats.Bavail * uint64(stats.Bsize), nil
		} else if !os.IsNotExist(err) {
			return 0, err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return 0, fmt.Errorf("no existing parent for %s", path)
		}
		existing = parent
	}
}
