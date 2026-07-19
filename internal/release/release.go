package release

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Manifest struct {
	Version         string `json:"version"`
	OS              string `json:"os"`
	Architecture    string `json:"architecture"`
	Filename        string `json:"filename"`
	URL             string `json:"url,omitempty"`
	SHA256          string `json:"sha256"`
	Size            int64  `json:"size"`
	Date            string `json:"date"`
	MinimumLauncher string `json:"minimumLauncher"`
	MinimumProtocol int    `json:"minimumProtocol"`
}

func DecodePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, errors.New("invalid release public key")
	}
	return ed25519.PublicKey(b), nil
}

func DecodePrivateKey(s string) (ed25519.PrivateKey, error) {
	b, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil || len(b) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid release private key")
	}
	return ed25519.PrivateKey(b), nil
}

func Sign(priv ed25519.PrivateKey, data []byte) string {
	return base64.RawStdEncoding.EncodeToString(ed25519.Sign(priv, data))
}

func Verify(pub ed25519.PublicKey, data []byte, signature string) bool {
	sig, err := base64.RawStdEncoding.DecodeString(signature)
	return err == nil && len(sig) == ed25519.SignatureSize && ed25519.Verify(pub, data, sig)
}

func Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func LoadAndVerify(payloadDir string, pub ed25519.PublicKey) (Manifest, []byte, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(payloadDir, "manifest.json"))
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read manifest: %w", err)
	}
	manifestSig, err := os.ReadFile(filepath.Join(payloadDir, "manifest.json.sig"))
	if err != nil || !Verify(pub, manifestBytes, string(manifestSig)) {
		return Manifest{}, nil, errors.New("manifest signature verification failed")
	}
	var m Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return Manifest{}, nil, fmt.Errorf("parse manifest: %w", err)
	}
	if m.OS != "windows" || m.Architecture != "amd64" || filepath.Base(m.Filename) != m.Filename || m.Filename == "." {
		return Manifest{}, nil, errors.New("manifest target rejected")
	}
	payload, err := os.ReadFile(filepath.Join(payloadDir, m.Filename))
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("read payload: %w", err)
	}
	payloadSig, err := os.ReadFile(filepath.Join(payloadDir, m.Filename+".sig"))
	if err != nil || !Verify(pub, payload, string(payloadSig)) {
		return Manifest{}, nil, errors.New("payload signature verification failed")
	}
	if int64(len(payload)) != m.Size || Hash(payload) != m.SHA256 {
		return Manifest{}, nil, errors.New("payload hash or size mismatch")
	}
	return m, payload, nil
}
