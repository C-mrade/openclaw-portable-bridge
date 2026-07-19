package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func NewIdentity() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
func Sign(priv ed25519.PrivateKey, msg []byte) string {
	return base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, msg))
}
func Verify(pubText, sigText string, msg []byte) bool {
	pub, e1 := base64.RawURLEncoding.DecodeString(pubText)
	sig, e2 := base64.RawURLEncoding.DecodeString(sigText)
	return e1 == nil && e2 == nil && len(pub) == ed25519.PublicKeySize && ed25519.Verify(pub, msg, sig)
}
func Token() (string, error) {
	b := make([]byte, 32)
	_, e := rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b), e
}
func Hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func CompareCode(pub, nonce string) string {
	h := sha256.Sum256([]byte(pub + "\x00" + nonce))
	return hex.EncodeToString(h[:3])
}
