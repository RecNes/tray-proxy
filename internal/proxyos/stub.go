//go:build !windows

package proxyos

import "fmt"

type stubManager struct {
	active string
}

func New(backupPath string) Manager {
	return &stubManager{}
}

func (m *stubManager) Apply(addr string) error {
	if addr == "" {
		return fmt.Errorf("empty proxy address")
	}
	m.active = addr
	return nil
}

func (m *stubManager) Restore() error {
	m.active = ""
	return nil
}

func (m *stubManager) IsApplied() bool { return m.active != "" }
func (m *stubManager) Active() string  { return m.active }
