package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/C-mrade/openclaw-portable-bridge/internal/release"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var releasePublicKey string

type publicConfig struct {
	USBID     string `json:"usbId"`
	BrokerURL string `json:"brokerUrl"`
}

func run() error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return errors.New("questa release supporta soltanto Windows 10/11 x64")
	}
	if releasePublicKey == "" {
		return errors.New("launcher privo della chiave pubblica di release")
	}
	pub, e := release.DecodePublicKey(releasePublicKey)
	if e != nil {
		return e
	}
	self, e := os.Executable()
	if e != nil {
		return e
	}
	root := filepath.Dir(self)
	configBytes, e := os.ReadFile(filepath.Join(root, "config", "bridge-public.json"))
	if e != nil {
		return fmt.Errorf("configurazione pubblica: %w", e)
	}
	var cfg publicConfig
	if e = json.Unmarshal(configBytes, &cfg); e != nil || cfg.USBID == "" || (!strings.HasPrefix(cfg.BrokerURL, "https://") && !strings.HasPrefix(cfg.BrokerURL, "http://127.0.0.1:")) {
		return errors.New("configurazione pubblica non valida o broker non TLS")
	}
	m, payload, e := release.LoadAndVerify(filepath.Join(root, "payload", "windows-amd64"), pub)
	if e != nil {
		return e
	}
	session := fmt.Sprintf("%d-%d", time.Now().UTC().Unix(), os.Getpid())
	stage := filepath.Join(os.TempDir(), "OpenClawBridge", session)
	if e = os.MkdirAll(stage, 0700); e != nil {
		return e
	}
	defer func() {
		if e := os.RemoveAll(stage); e != nil {
			fmt.Fprintln(os.Stderr, "ATTENZIONE: file temporanei rimasti:", stage, e)
		}
	}()
	clientPath := filepath.Join(stage, "bridge-client.exe")
	if e = os.WriteFile(clientPath, payload, 0700); e != nil {
		return e
	}
	fmt.Printf("OpenClaw Portable Bridge %s\nPayload verificato. Directory temporanea: %s\n", m.Version, stage)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Consenso: [1] Profilo Informazioni (sola lettura)  [2] Profilo Sviluppatore")
	fmt.Print("Selezione [1]: ")
	choice, _ := reader.ReadString('\n')
	profile := "information"
	args := []string{"-broker", cfg.BrokerURL, "-usb-id", cfg.USBID, "-profile", profile}
	if strings.TrimSpace(choice) == "2" {
		profile = "developer"
		fmt.Println("Profilo Sviluppatore: shell utente, processi e file nelle sole cartelle selezionate.")
		fmt.Print("Percorso assoluto della cartella consentita: ")
		root, _ := reader.ReadString('\n')
		root = strings.TrimSpace(root)
		if root == "" {
			return errors.New("il profilo Sviluppatore richiede una cartella esplicitamente selezionata")
		}
		info, e := os.Stat(root)
		if e != nil || !info.IsDir() {
			return errors.New("cartella consentita non valida")
		}
		args = []string{"-broker", cfg.BrokerURL, "-usb-id", cfg.USBID, "-profile", profile, "-allow-dir", root}
	} else {
		fmt.Println("Profilo Informazioni: inventario di sistema/rete/dischi/servizi e lettura file opzionale.")
		fmt.Print("Cartella consultabile (INVIO per nessun accesso file): ")
		readRoot, _ := reader.ReadString('\n')
		readRoot = strings.TrimSpace(readRoot)
		if readRoot != "" {
			info, statErr := os.Stat(readRoot)
			if statErr != nil || !info.IsDir() {
				return errors.New("cartella consultabile non valida")
			}
			args = append(args, "-allow-dir", readRoot)
		}
	}
	fmt.Printf("Avvio visibile con profilo %s. La sessione richiederà approvazione remota.\n", profile)
	cmd := exec.Command(clientPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	e = cmd.Run()
	if e != nil {
		return fmt.Errorf("client terminato con errore: %w", e)
	}
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "OpenClaw Bridge - ERRORE:", e)
		fmt.Println("Premi INVIO per chiudere.")
		var s string
		_, _ = fmt.Scanln(&s)
		os.Exit(1)
	}
}
