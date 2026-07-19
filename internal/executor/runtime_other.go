//go:build !windows

package executor

import "os"

type portableProcessContainer struct{ process *os.Process }

func newProcessContainer(process *os.Process) (processContainer, error) {
	return &portableProcessContainer{process: process}, nil
}

func (p *portableProcessContainer) Terminate(_ uint32) error { return p.process.Kill() }
func (p *portableProcessContainer) Close() error             { return nil }
