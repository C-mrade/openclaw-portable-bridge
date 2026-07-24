package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
	"github.com/C-mrade/openclaw-portable-bridge/internal/release"
	"os"
	"path/filepath"
	"time"
)

func main() {
	mode := flag.String("mode", "sign", "keygen or sign")
	key := flag.String("key", "", "private key path")
	publicKey := flag.String("public-key", "", "base64 public key")
	input := flag.String("input", "", "file to sign")
	signature := flag.String("signature", "", "detached signature output path")
	payload := flag.String("payload", "", "client executable")
	out := flag.String("out", "", "release directory")
	version := flag.String("version", "0.1.0", "release version")
	launcherVersion := flag.String("launcher-version", "", "minimum compatible launcher version")
	targetOS := flag.String("target-os", "windows", "manifest operating system")
	targetArch := flag.String("target-arch", "amd64", "manifest architecture")
	filename := flag.String("filename", "bridge-client.exe", "payload filename in the manifest")
	flag.Parse()
	if *mode == "keygen" {
		pub, priv, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			panic(e)
		}
		if *key == "" {
			panic("-key required")
		}
		if e = os.WriteFile(*key, []byte(base64.RawStdEncoding.EncodeToString(priv)), 0600); e != nil {
			panic(e)
		}
		fmt.Println(base64.RawStdEncoding.EncodeToString(pub))
		return
	}
	if *mode == "sign-file" {
		if *key == "" || *input == "" || *signature == "" {
			panic("-key, -input and -signature required")
		}
		priv := loadPrivateKey(*key)
		data, err := os.ReadFile(*input)
		if err != nil {
			panic(err)
		}
		must(os.WriteFile(*signature, []byte(release.Sign(priv, data)), 0644))
		return
	}
	if *mode == "check-keypair" {
		if *key == "" || *publicKey == "" {
			panic("-key and -public-key required")
		}
		priv := loadPrivateKey(*key)
		pub, err := release.DecodePublicKey(*publicKey)
		if err != nil || !bytes.Equal(priv.Public().(ed25519.PublicKey), pub) {
			panic("release private and public keys do not match")
		}
		return
	}
	if *mode != "sign" {
		panic("unsupported mode")
	}
	if *key == "" || *payload == "" || *out == "" {
		panic("-key, -payload and -out required")
	}
	if *launcherVersion == "" {
		*launcherVersion = *version
	}
	if !release.VersionAtLeast(*version, *launcherVersion) {
		panic("invalid release or minimum launcher version")
	}
	priv := loadPrivateKey(*key)
	data, e := os.ReadFile(*payload)
	if e != nil {
		panic(e)
	}
	if e = os.MkdirAll(*out, 0755); e != nil {
		panic(e)
	}
	name := filepath.Base(*filename)
	if name == "." || name != *filename || (*targetOS != "windows" && *targetOS != "linux" && *targetOS != "darwin") || (*targetArch != "amd64" && *targetArch != "arm64") {
		panic("invalid release target or filename")
	}
	m := release.Manifest{Version: *version, OS: *targetOS, Architecture: *targetArch, Filename: name, SHA256: release.Hash(data), Size: int64(len(data)), Date: time.Now().UTC().Format(time.RFC3339), MinimumLauncher: *launcherVersion, MinimumProtocol: protocol.Version}
	mb, _ := json.MarshalIndent(m, "", "  ")
	mb = append(mb, '\n')
	must(os.WriteFile(filepath.Join(*out, name), data, 0644))
	must(os.WriteFile(filepath.Join(*out, name+".sig"), []byte(release.Sign(priv, data)), 0644))
	must(os.WriteFile(filepath.Join(*out, "manifest.json"), mb, 0644))
	must(os.WriteFile(filepath.Join(*out, "manifest.json.sig"), []byte(release.Sign(priv, mb)), 0644))
	fmt.Println(m.SHA256)
}

func loadPrivateKey(path string) ed25519.PrivateKey {
	rawKey, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	priv, err := release.DecodePrivateKey(string(rawKey))
	if err != nil {
		panic(err)
	}
	return priv
}

func must(e error) {
	if e != nil {
		panic(e)
	}
}
