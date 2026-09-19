package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type recoveryBackup struct {
	Schema    string               `json:"schema"`
	PackageID string               `json:"package_id"`
	CreatedAt string               `json:"created_at"`
	Files     []recoveryBackupFile `json:"files"`
}

type recoveryBackupFile struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
	SHA256       string `json:"sha256"`
	Mode         uint32 `json:"mode"`
	Size         int64  `json:"size"`
}

func (m Manager) recoveryBackupDirectory(pkg manifest.Package) string {
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now().UTC()
	}
	return managerPath + "/recovery-backups/" + pkg.ID + "/" + now.Format("20060102T150405.000000000Z")
}

func (m Manager) backupRecoveryDestination(tx *safefs.Transaction, pkg manifest.Package, existing []existingFile, directory string) error {
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now().UTC()
	}
	record := recoveryBackup{
		Schema:    "org.knulli.app-store/recovery-backup/v1",
		PackageID: pkg.ID,
		CreatedAt: now.Format(time.RFC3339Nano),
		Files:     make([]recoveryBackupFile, 0, len(existing)),
	}
	for _, file := range existing {
		digest := sha256.Sum256([]byte(file.Virtual))
		backup := directory + "/files/" + hex.EncodeToString(digest[:])
		if err := tx.Copy(file.Host, backup, file.Mode); err != nil {
			return fmt.Errorf("create recovery backup for %s: %w", file.Virtual, err)
		}
		info, err := os.Stat(file.Host)
		if err != nil {
			return err
		}
		record.Files = append(record.Files, recoveryBackupFile{
			OriginalPath: file.Virtual,
			BackupPath:   backup,
			SHA256:       file.SHA256,
			Mode:         uint32(file.Mode.Perm()),
			Size:         info.Size(),
		})
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	manifestPath := directory + "/manifest.json"
	if err := tx.Write(manifestPath, append(data, '\n'), 0600); err != nil {
		return err
	}
	return nil
}
