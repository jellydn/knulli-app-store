package safefs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	journalName       = "journal.json"
	journalSchemaV1   = "org.knulli.app-store/transaction-journal/v1"
	journalStatusOpen = "open"
	journalCommitted  = "committed"
	transactionPrefix = "transaction-"
)

type snapshot struct {
	path    string
	virtual string
	backup  string
	mode    os.FileMode
	existed bool
}

type journalRecord struct {
	Schema    string            `json:"schema"`
	Status    string            `json:"status"`
	Snapshots []journalSnapshot `json:"snapshots"`
}

type journalSnapshot struct {
	Host    string `json:"host"`
	Virtual string `json:"virtual"`
	Backup  string `json:"backup,omitempty"`
	Mode    uint32 `json:"mode,omitempty"`
	Existed bool   `json:"existed"`
}

type Transaction struct {
	guard     *Guard
	directory string
	snapshots []snapshot
	seen      map[string]bool
	closed    bool
}

func Begin(guard *Guard, transactionParent string) (*Transaction, error) {
	directory, err := os.MkdirTemp(transactionParent, transactionPrefix+"*")
	if err != nil {
		return nil, err
	}
	return &Transaction{guard: guard, directory: directory, seen: make(map[string]bool)}, nil
}

func Recover(root, parent string) error {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return fmt.Errorf("resolve filesystem root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	matches, err := filepath.Glob(filepath.Join(parent, transactionPrefix+"*"))
	if err != nil {
		return err
	}
	for _, directory := range matches {
		info, err := os.Lstat(directory)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			continue
		}
		if err := recoverOne(absolute, directory); err != nil {
			return err
		}
	}
	return nil
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
	if err := t.persistJournal(journalCommitted); err != nil {
		return err
	}
	t.closed = true
	return os.RemoveAll(t.directory)
}

func (t *Transaction) Rollback() error {
	if t.closed {
		return nil
	}
	t.closed = true
	first := rollbackSnapshots(t.guard.root, t.snapshots)
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
	item := snapshot{path: host, virtual: virtual}
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
	// Record the snapshot before the caller mutates the destination so a restart can roll back.
	if err := t.persistJournal(journalStatusOpen); err != nil {
		return "", err
	}
	return host, nil
}

func (t *Transaction) persistJournal(status string) error {
	record := journalRecord{Schema: journalSchemaV1, Status: status, Snapshots: []journalSnapshot{}}
	for _, item := range t.snapshots {
		entry := journalSnapshot{Host: item.path, Virtual: item.virtual, Mode: uint32(item.mode), Existed: item.existed}
		if item.existed {
			entry.Backup = filepath.Base(item.backup)
		}
		record.Snapshots = append(record.Snapshots, entry)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWrite(filepath.Join(t.directory, journalName), append(data, '\n'), 0600)
}

func recoverOne(root, directory string) error {
	record, err := readJournal(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return os.RemoveAll(directory)
		}
		return err
	}
	if record.Status == journalCommitted {
		return os.RemoveAll(directory)
	}
	if record.Status != journalStatusOpen {
		return fmt.Errorf("unknown transaction journal status %q in %s", record.Status, directory)
	}
	snapshots, err := snapshotsFromRecord(root, directory, record)
	if err != nil {
		return err
	}
	if err := rollbackSnapshots(root, snapshots); err != nil {
		return err
	}
	return os.RemoveAll(directory)
}

func readJournal(directory string) (journalRecord, error) {
	data, err := os.ReadFile(filepath.Join(directory, journalName))
	if err != nil {
		return journalRecord{}, err
	}
	var record journalRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return journalRecord{}, fmt.Errorf("decode transaction journal: %w", err)
	}
	if record.Schema != journalSchemaV1 {
		return journalRecord{}, fmt.Errorf("unsupported transaction journal schema %q", record.Schema)
	}
	return record, nil
}

func snapshotsFromRecord(root, directory string, record journalRecord) ([]snapshot, error) {
	snapshots := make([]snapshot, 0, len(record.Snapshots))
	for _, item := range record.Snapshots {
		if err := withinRoot(root, item.Host); err != nil {
			return nil, err
		}
		snap := snapshot{path: item.Host, virtual: item.Virtual, mode: os.FileMode(item.Mode), existed: item.Existed}
		if item.Existed {
			if item.Backup == "" || item.Backup != filepath.Base(item.Backup) {
				return nil, fmt.Errorf("invalid journal backup name %q", item.Backup)
			}
			snap.backup = filepath.Join(directory, item.Backup)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, nil
}

func rollbackSnapshots(root string, snapshots []snapshot) error {
	var first error
	for index := len(snapshots) - 1; index >= 0; index-- {
		if err := restoreSnapshot(root, snapshots[index]); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func restoreSnapshot(root string, item snapshot) error {
	if err := withinRoot(root, item.path); err != nil {
		return err
	}
	if err := rejectSymlinkParents(root, item.path); err != nil {
		return err
	}
	if item.existed {
		return Copy(item.backup, item.path, item.mode)
	}
	if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func withinRoot(root, host string) error {
	relative, err := filepath.Rel(root, host)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("journal path is outside filesystem root")
	}
	return nil
}
