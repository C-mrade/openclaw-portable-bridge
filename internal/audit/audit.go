package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Logger struct {
	mu sync.Mutex
	f  *os.File
}

func Open(path string) (*Logger, error) {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	return &Logger{f: f}, e
}
func (l *Logger) Event(kind string, fields map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fields["at"] = time.Now().UTC()
	fields["event"] = kind
	_ = json.NewEncoder(l.f).Encode(fields)
}
func (l *Logger) Close() error { return l.f.Close() }
