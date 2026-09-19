package installer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func applyExecutableModes(files []storearchive.File, executables []string) error {
	wanted := stringSet(executables)
	for index := range files {
		if wanted[files[index].Relative] {
			files[index].Mode = 0755
			delete(wanted, files[index].Relative)
		}
	}
	for executable := range wanted {
		return fmt.Errorf("release archive does not contain declared executable %s", executable)
	}
	return nil
}

func applyBinaryPatches(files []storearchive.File, patches []manifest.BinaryPatch) error {
	byPath := make(map[string]*storearchive.File, len(files))
	for index := range files {
		byPath[files[index].Relative] = &files[index]
	}
	for _, patch := range patches {
		file := byPath[patch.Path]
		if file == nil {
			return fmt.Errorf("binary patch target is absent: %s", patch.Path)
		}
		before, _ := hex.DecodeString(patch.BeforeHex)
		after, _ := hex.DecodeString(patch.AfterHex)
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return err
		}
		end := patch.Offset + int64(len(before))
		if patch.Offset < 0 || end > int64(len(data)) || !bytes.Equal(data[patch.Offset:end], before) {
			return fmt.Errorf("binary patch source mismatch for %s at offset %d", patch.Path, patch.Offset)
		}
		copy(data[patch.Offset:end], after)
		digest := sha256.Sum256(data)
		actual := hex.EncodeToString(digest[:])
		if actual != patch.SHA256 {
			return fmt.Errorf("binary patch result mismatch for %s: expected %s, got %s", patch.Path, patch.SHA256, actual)
		}
		if err := os.WriteFile(file.Path, data, file.Mode); err != nil {
			return err
		}
		file.SHA256 = actual
	}
	return nil
}

func isPreserved(relative string, preserved []string) bool {
	for _, entry := range preserved {
		if relative == entry || strings.HasPrefix(relative, entry+"/") {
			return true
		}
	}
	return false
}

func originalPath(id, target string) string {
	digest := sha256.Sum256([]byte(target))
	return managerPath + "/originals/" + id + "/" + hex.EncodeToString(digest[:])
}

func containsArchiveFile(files []storearchive.File, relative string) bool {
	for _, file := range files {
		if file.Relative == relative {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
