package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Schema = "org.knulli.app-store/controller-mappings/v1"
	Path   = "/userdata/system/configs/knulli-app-store/controller-mappings.json"
)

type Identity struct {
	Device string `json:"device"`
	GUID   string `json:"guid,omitempty"`
	Name   string `json:"name,omitempty"`
}

func (identity Identity) Key() (string, error) {
	device := normalizeIdentity(identity.Device)
	guid := normalizeIdentity(identity.GUID)
	name := normalizeIdentity(identity.Name)
	if strings.Trim(guid, "0") == "" {
		guid = ""
	}
	if device == "" {
		return "", fmt.Errorf("device identity is missing")
	}
	if guid != "" {
		return device + "|guid:" + guid, nil
	}
	if name == "" {
		return "", fmt.Errorf("controller has neither a GUID nor a name")
	}
	return device + "|name:" + name, nil
}

type Record struct {
	Identity Identity `json:"identity"`
	Mapping  Mapping  `json:"mapping"`
}

type fileData struct {
	Schema  string   `json:"schema"`
	Records []Record `json:"records"`
}

type Store struct {
	root string
}

type corruptFileError struct {
	err error
}

func (err corruptFileError) Error() string {
	return err.err.Error()
}

func (err corruptFileError) Unwrap() error {
	return err.err
}

func NewStore(root string) Store {
	if root == "" {
		root = "/"
	}
	return Store{root: root}
}

func (store Store) Load(identity Identity) (Mapping, bool, error) {
	key, err := identity.Key()
	if err != nil {
		return nil, false, err
	}
	data, err := store.read()
	if err != nil {
		return nil, false, err
	}
	for _, record := range data.Records {
		recordKey, keyErr := record.Identity.Key()
		if keyErr != nil {
			return nil, false, fmt.Errorf("invalid saved controller identity: %w", keyErr)
		}
		if recordKey == key {
			migrated := record.Mapping.KeepKnown()
			if err := migrated.Validate(); err != nil {
				return nil, false, fmt.Errorf("invalid saved mapping: %w", err)
			}
			return migrated, true, nil
		}
	}
	return nil, false, nil
}

func (store Store) Save(identity Identity, mapping Mapping) error {
	key, err := identity.Key()
	if err != nil {
		return err
	}
	if err := mapping.Validate(); err != nil {
		return err
	}
	data, err := store.readForUpdate()
	if err != nil {
		return err
	}
	record := Record{Identity: identity, Mapping: mapping.Clone()}
	replaced := false
	for index := range data.Records {
		recordKey, keyErr := data.Records[index].Identity.Key()
		if keyErr != nil {
			return fmt.Errorf("invalid saved controller identity: %w", keyErr)
		}
		if recordKey == key {
			data.Records[index] = record
			replaced = true
			break
		}
	}
	if !replaced {
		data.Records = append(data.Records, record)
	}
	return store.write(data)
}

func (store Store) Reset(identity Identity) error {
	key, err := identity.Key()
	if err != nil {
		return err
	}
	data, err := store.readForUpdate()
	if err != nil {
		return err
	}
	kept := data.Records[:0]
	for _, record := range data.Records {
		recordKey, keyErr := record.Identity.Key()
		if keyErr != nil {
			return fmt.Errorf("invalid saved controller identity: %w", keyErr)
		}
		if recordKey != key {
			kept = append(kept, record)
		}
	}
	data.Records = kept
	return store.write(data)
}

func (store Store) readForUpdate() (fileData, error) {
	data, err := store.read()
	if err == nil {
		return data, nil
	}
	var corrupt corruptFileError
	if !errors.As(err, &corrupt) {
		return data, err
	}
	path := store.hostPath()
	corruptPath := path + ".corrupt"
	for suffix := 1; ; suffix++ {
		if _, statErr := os.Stat(corruptPath); os.IsNotExist(statErr) {
			break
		} else if statErr != nil {
			return data, fmt.Errorf("inspect corrupt controller mapping destination: %w", statErr)
		}
		corruptPath = fmt.Sprintf("%s.corrupt.%d", path, suffix)
	}
	if renameErr := os.Rename(path, corruptPath); renameErr != nil {
		return data, fmt.Errorf("preserve corrupt controller mappings: %w", renameErr)
	}
	return fileData{Schema: Schema}, nil
}

func (store Store) read() (fileData, error) {
	data := fileData{Schema: Schema}
	encoded, err := os.ReadFile(store.hostPath())
	if os.IsNotExist(err) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(encoded, &data); err != nil {
		return data, corruptFileError{err: fmt.Errorf("decode controller mappings: %w", err)}
	}
	if data.Schema != Schema {
		return data, corruptFileError{err: fmt.Errorf("unsupported controller mapping schema %q", data.Schema)}
	}
	keys := make(map[string]struct{}, len(data.Records))
	usable := make([]Record, 0, len(data.Records))
	for _, record := range data.Records {
		key, err := record.Identity.Key()
		if err != nil {
			return data, corruptFileError{err: fmt.Errorf("invalid saved controller identity: %w", err)}
		}
		if _, exists := keys[key]; exists {
			return data, corruptFileError{err: fmt.Errorf("duplicate saved controller identity %q", key)}
		}
		// A record saved by an older build may still bind the exit button that
		// quitting used to carry. Dropping it here migrates the record instead of
		// treating the whole file as corrupt, and the next write stores the
		// migrated form.
		record.Mapping = record.Mapping.KeepKnown()
		if missing := record.Mapping.Missing(); len(missing) > 0 {
			// The action set has grown since this record was written, so it
			// cannot drive the current screens and no button can be invented for
			// the gap. It is stale, not damaged: the file is shared by every
			// controller, so one stale record is dropped while every other
			// identity keeps the mapping it saved. The dropped identity simply
			// has no saved mapping, which sends that user back through the setup
			// that binds the new actions.
			continue
		}
		if err := record.Mapping.Validate(); err != nil {
			return data, corruptFileError{err: fmt.Errorf("invalid saved mapping for %q: %w", key, err)}
		}
		keys[key] = struct{}{}
		usable = append(usable, record)
	}
	data.Records = usable
	return data, nil
}

func (store Store) write(data fileData) error {
	path := store.hostPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".controller-mappings-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
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
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func (store Store) hostPath() string {
	return filepath.Join(store.root, filepath.FromSlash(strings.TrimPrefix(Path, "/")))
}

func normalizeIdentity(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}
