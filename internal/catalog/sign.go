package catalog

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	SignatureSchemaV1  = "org.knulli.app-store/catalog-signature/v1"
	SignatureAlgorithm = "ed25519"
)

// embeddedPublicKeyHex is the ed25519 public key compiled into device artifacts.
var embeddedPublicKeyHex string

type Signature struct {
	Schema    string `json:"schema"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`
}

func SignaturePath(indexPath string) string {
	return indexPath + ".sig"
}

func EmbeddedPublicKey() (ed25519.PublicKey, error) {
	if strings.TrimSpace(embeddedPublicKeyHex) == "" {
		return nil, nil
	}
	return ParsePublicKey(embeddedPublicKeyHex)
}

func ParsePublicKey(text string) (ed25519.PublicKey, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(text))
	if err != nil {
		return nil, fmt.Errorf("decode catalogue public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("catalogue public key must be %d bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

func ParsePrivateKey(text string) (ed25519.PrivateKey, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(text))
	if err != nil {
		return nil, fmt.Errorf("decode catalogue private key: %w", err)
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	default:
		return nil, fmt.Errorf("catalogue private key must be %d or %d bytes", ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}

func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func Sign(indexBytes []byte, private ed25519.PrivateKey) (Signature, error) {
	if l := len(private); l != ed25519.PrivateKeySize {
		return Signature{}, fmt.Errorf("invalid catalogue private key size %d", l)
	}
	return Signature{
		Schema:    SignatureSchemaV1,
		Algorithm: SignatureAlgorithm,
		Signature: hex.EncodeToString(ed25519.Sign(private, indexBytes)),
	}, nil
}

func Verify(indexBytes []byte, signature Signature, public ed25519.PublicKey) error {
	if signature.Schema != SignatureSchemaV1 {
		return fmt.Errorf("unsupported catalogue signature schema %q", signature.Schema)
	}
	if signature.Algorithm != SignatureAlgorithm {
		return fmt.Errorf("unsupported catalogue signature algorithm %q", signature.Algorithm)
	}
	if len(public) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid catalogue public key")
	}
	raw, err := hex.DecodeString(signature.Signature)
	if err != nil {
		return fmt.Errorf("decode catalogue signature: %w", err)
	}
	if !ed25519.Verify(public, indexBytes, raw) {
		return fmt.Errorf("catalogue signature is invalid")
	}
	return nil
}

func SignFile(indexPath string, private ed25519.PrivateKey) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	signature, err := Sign(data, private)
	if err != nil {
		return err
	}
	return writeSignature(SignaturePath(indexPath), signature)
}

func loadSignature(path string) (Signature, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Signature{}, err
	}
	var signature Signature
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&signature); err != nil {
		return Signature{}, fmt.Errorf("decode catalogue signature: %w", err)
	}
	return signature, nil
}

func writeSignature(path string, signature Signature) error {
	data, err := json.MarshalIndent(signature, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".catalog-signature-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0644); err != nil {
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
	return os.Rename(temporaryName, path)
}
