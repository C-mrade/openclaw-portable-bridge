package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/C-mrade/openclaw-portable-bridge/internal/auth"
	"github.com/C-mrade/openclaw-portable-bridge/internal/executor"
	"github.com/C-mrade/openclaw-portable-bridge/internal/protocol"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"syscall"
	"time"
)

type stringList []string

func (s *stringList) String() string     { return fmt.Sprint([]string(*s)) }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }
func capabilities(p string) []string {
	switch p {
	case "information":
		return []string{"system.info", "system.network", "disk.list", "service.list", "process.list", "files.list", "files.read", "files.download", "session.disconnect"}
	case "developer":
		return []string{"system.info", "system.network", "disk.list", "service.list", "process.list", "process.start", "process.stop-owned", "shell.run", "files.list", "files.read", "files.write", "files.upload", "files.download", "session.disconnect"}
	case "custom":
		return []string{"system.info", "session.disconnect"}
	}
	return nil
}
func call(method, url, token string, in, out any) error {
	var b bytes.Buffer
	if in != nil {
		_ = json.NewEncoder(&b).Encode(in)
	}
	req, e := http.NewRequest(method, url, &b)
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, e := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		return nil
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("server status %s", resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
func main() {
	broker := flag.String("broker", "http://127.0.0.1:17443", "")
	usb := flag.String("usb-id", "portable-bridge-example", "")
	profile := flag.String("profile", "information", "")
	var roots stringList
	flag.Var(&roots, "allow-dir", "approved directory; repeatable")
	flag.Parse()
	requested := capabilities(*profile)
	if requested == nil {
		log.Fatal("invalid capability profile")
	}
	local, e := executor.New(roots)
	if e != nil {
		log.Fatal(e)
	}
	pub, priv, e := auth.NewIdentity()
	if e != nil {
		log.Fatal(e)
	}
	defer func() {
		for i := range priv {
			priv[i] = 0
		}
	}()
	nonce := make([]byte, 24)
	_, _ = rand.Read(nonce)
	h, _ := os.Hostname()
	u, _ := user.Current()
	q := protocol.PairRequest{Protocol: protocol.Version, USBID: *usb, Hostname: h, OS: runtime.GOOS, Arch: runtime.GOARCH, User: u.Username, PublicKey: base64.RawURLEncoding.EncodeToString(pub), Nonce: base64.RawURLEncoding.EncodeToString(nonce), Requested: requested, DurationSeconds: 1800}
	q.Signature = auth.Sign(priv, protocol.CanonicalPairRequest(q))
	var rep protocol.PairReply
	if e = call("POST", *broker+"/v1/pair/request", "", q, &rep); e != nil {
		log.Fatal(e)
	}
	fmt.Printf("Pairing request %s | comparison code %s\nMachine: %s | OS: %s/%s | User (descriptive): %s\nRequested: %v\n", rep.RequestID, rep.CompareCode, h, runtime.GOOS, runtime.GOARCH, u.Username, requested)
	pairingToken := rep.PairingToken
	for rep.Status == "pending" {
		time.Sleep(time.Second)
		if e = call("GET", *broker+"/v1/pair/status?id="+rep.RequestID, pairingToken, nil, &rep); e != nil {
			log.Fatal(e)
		}
	}
	if rep.Status != "approved" {
		log.Fatal("pairing not approved")
	}
	fmt.Printf("CONNECTED until %s\n", rep.ExpiresAt.Format(time.RFC3339))
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	interrupted := make(chan struct{})
	go func() {
		select {
		case <-signals:
			_ = call("POST", *broker+"/v1/session/revoke", rep.SessionToken, map[string]string{}, nil)
			os.Exit(0)
		case <-interrupted:
		}
	}()
	for {
		var cmd protocol.Command
		e = call("POST", *broker+"/v1/session/poll", rep.SessionToken, map[string]string{}, &cmd)
		if e != nil {
			break
		}
		if cmd.Name == "" {
			time.Sleep(time.Second)
			continue
		}
		start := time.Now()
		output, runErr := local.Execute(cmd)
		res := protocol.Result{ID: cmd.ID, Name: cmd.Name, StartedAt: start, FinishedAt: time.Now(), Output: output}
		if runErr != nil {
			res.Error = runErr.Error()
		}
		_ = call("POST", *broker+"/v1/session/result", rep.SessionToken, res, nil)
		fmt.Printf("[%s] %s exit=%s\n", res.FinishedAt.Format(time.RFC3339), cmd.Name, res.Error)
		if cmd.Name == "session.disconnect" {
			break
		}
	}
	close(interrupted)
	_ = call("POST", *broker+"/v1/session/revoke", rep.SessionToken, map[string]string{}, nil)
	fmt.Println("DISCONNECTED; session token revoked")
}
