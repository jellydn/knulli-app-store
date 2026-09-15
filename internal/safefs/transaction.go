package safefs

import (
	"fmt"
	"os"
	"path/filepath"
)

type snapshot struct {
	path    string
	backup  string
	mode    os.FileMode
	existed bool
}

type Transaction struct {
	guard     *Guard
	directory string
	snapshots []snapshot
	seen      map[string]bool
	closed    bool
}

func Begin(guard *Guard, transactionParent string) (*Transaction, error) {
	directory, err := os.MkdirTemp(transactionParent, "transaction-*")
	if err != nil {
		return nil, err
	}
	return &Transaction{guard: guard, directory: directory, seen: make(map[string]bool)}, nil
}

func (t *Transaction) Write(virtual string, data []byte, mode os.FileMode) error {
	host, err := t.prepare(virtual)
	if err != nil {
		return err
	}
	return AtomicWrite(host, data, mode)
}

func (t *Transaction) Copy(source, virtual string, mode os.FileMode) error {
	host, err := t.prepare(virtual)
	if err != nil {
		return err
	}
	return Copy(source, host, mode)
}

func (t *Transaction) Remove(virtual string) error {
	host, err := t.prepare(virtual)
	if err != nil {
		return err
	}
	if err := os.Remove(host); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (t *Transaction) Commit() error {
	if t.closed {
		return fmt.Errorf("transaction is closed")
	}
	if err := os.RemoveAll(t.directory); err != nil {
		return err
	}
	t.closed = true
	return nil
}

func (t *Transaction) Rollback() error {
	if t.closed {
		return nil
	}
	t.closed = true
	var first error
	for index := len(t.snapshots) - 1; index >= 0; index-- {
		item := t.snapshots[index]
		if item.existed {
			if err := Copy(item.backup, item.path, item.mode); err != nil && first == nil {
				first = err
			}
		} else if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
	}
	if err := os.RemoveAll(t.directory); err != nil && first == nil {
		first = err
	}
	return first
}

func (t *Transaction) prepare(virtual string) (string, error) {
	if t.closed {
		return "", fmt.Errorf("transaction is closed")
	}
	host, err := t.guard.Resolve(virtual)
	if err != nil {
		return "", err
	}
	if t.seen[host] {
		return host, nil
	}
	item := snapshot{path: host}
	info, err := os.Lstat(host)
	if err == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("refusing to replace non-regular file %s", virtual)
		}
		item.existed = true
		item.mode = info.Mode()
		item.backup = filepath.Join(t.directory, fmt.Sprintf("%06d", len(t.snapshots)))
		if err := Copy(host, item.backup, info.Mode()); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	t.seen[host] = true
	t.snapshots = append(t.snapshots, item)
	return host, nil
}
