package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func Load(path string) (Package, error) {
	file, err := os.Open(path)
	if err != nil {
		return Package{}, err
	}
	defer file.Close()
	return Decode(file)
}

func Decode(reader io.Reader) (Package, error) {
	var pkg Package
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pkg); err != nil {
		return Package{}, fmt.Errorf("decode manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Package{}, fmt.Errorf("decode manifest: trailing JSON value")
		}
		return Package{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := pkg.Validate(); err != nil {
		return Package{}, fmt.Errorf("validate manifest: %w", err)
	}
	return pkg, nil
}

func Canonical(pkg Package) ([]byte, error) {
	data, err := json.Marshal(pkg)
	if err != nil {
		return nil, err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, err
	}
	return compact.Bytes(), nil
}
