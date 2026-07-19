package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

const Version = 1

type PairRequest struct {
	Protocol                                                     int `json:"protocol"`
	USBID, Hostname, OS, Arch, User, PublicKey, Nonce, Signature string
	Requested                                                    []string `json:"requested"`
	DurationSeconds                                              int64    `json:"durationSeconds"`
}
type PairReply struct {
	RequestID, Status, CompareCode, PairingToken, SessionToken, Error string
	ExpiresAt                                                         time.Time
}
type Command struct {
	ID, Name string
	Deadline time.Time
	Params   json.RawMessage `json:"params,omitempty"`
}
type Result struct {
	ID, Name              string
	StartedAt, FinishedAt time.Time
	Output                string
	Error                 string
}
type Info struct{ Hostname, OS, Arch, User string }

func CanonicalPairRequest(q PairRequest) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%d\n%s\n%s\n%s\n%d\n%s\n%s\n%s\n%s\n", q.Protocol, q.USBID, q.PublicKey, q.Nonce, q.DurationSeconds, q.Hostname, q.OS, q.Arch, q.User)
	for _, capability := range q.Requested {
		fmt.Fprintf(&b, "%s\n", capability)
	}
	return b.Bytes()
}
