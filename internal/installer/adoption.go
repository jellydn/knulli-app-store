package installer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const maximumAdoptionBytes = 512 << 20

type existingFile struct {
	Virtual string
	Host    string
	Mode    os.FileMode
	SHA256  string
}

func inventoryExisting(destinationHost, destination string) ([]existingFile, uint64, error) {
	info, err := os.Lstat(destinationHost)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("destination is not a directory; move it aside before retrying")
	}
	var files []existingFile
	var total uint64
	err = filepath.Walk(destinationHost, func(host string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing non-regular pre-existing path %s; move the package directory aside before retrying", host)
		}
		relative, err := filepath.Rel(destinationHost, host)
		if err != nil {
			return err
		}
		size := uint64(info.Size())
		if size > maximumAdoptionBytes || total > maximumAdoptionBytes-size {
			return fmt.Errorf("pre-existing package exceeds the 512 MiB adoption limit; move it aside before retrying")
		}
		total += size
		digest, err := safefs.SHA256(host)
		if err != nil {
			return err
		}
		files = append(files, existingFile{Virtual: path.Join(destination, filepath.ToSlash(relative)), Host: host, Mode: info.Mode(), SHA256: digest})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Virtual < files[j].Virtual })
	return files, total, err
}
