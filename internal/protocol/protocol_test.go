package protocol

import "testing"

func TestCanonicalCoversCapabilities(t *testing.T) {
	q := PairRequest{Protocol: 1, USBID: "usb", PublicKey: "pub", Nonce: "nonce", DurationSeconds: 60, Hostname: "host", OS: "windows", Arch: "amd64", User: "user", Requested: []string{"system.info"}}
	a := string(CanonicalPairRequest(q))
	q.Requested = []string{"shell.run"}
	b := string(CanonicalPairRequest(q))
	if a == b {
		t.Fatal("capability tamper not covered by signature payload")
	}
}
