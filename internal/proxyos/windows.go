//go:build windows

package proxyos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const internetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type WinManager struct {
	backupPath string
	snap       Snapshot
	active     string
}

func New(backupPath string) *WinManager {
	m := &WinManager{backupPath: backupPath}
	_ = m.loadBackupFile()
	return m
}

func (m *WinManager) Active() string   { return m.active }
func (m *WinManager) IsApplied() bool  { return m.active != "" }

func (m *WinManager) Apply(addr string) error {
	if addr == "" {
		return fmt.Errorf("empty proxy address")
	}
	if !m.snap.Valid {
		if err := m.capture(); err != nil {
			return err
		}
	}
	if err := m.write(1, addr, m.snap.ProxyOverride); err != nil {
		return err
	}
	if err := notify(); err != nil {
		return err
	}
	m.active = addr
	return nil
}

func (m *WinManager) Restore() error {
	if m.snap.Valid {
		if err := m.write(m.snap.ProxyEnable, m.snap.ProxyServer, m.snap.ProxyOverride); err != nil {
			_ = m.write(0, "", "")
			_ = notify()
			m.active = ""
			return err
		}
	} else {
		_ = m.write(0, "", "")
	}
	_ = notify()
	m.active = ""
	return nil
}

func (m *WinManager) capture() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	enable, _, _ := k.GetIntegerValue("ProxyEnable")
	server, _, _ := k.GetStringValue("ProxyServer")
	override, _, _ := k.GetStringValue("ProxyOverride")
	m.snap = Snapshot{
		ProxyEnable:   uint32(enable),
		ProxyServer:   server,
		ProxyOverride: override,
		Valid:         true,
	}
	return m.saveBackupFile()
}

func (m *WinManager) write(enable uint32, server, override string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetDWordValue("ProxyEnable", enable); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyServer", server); err != nil {
		return err
	}
	if override != "" || m.snap.Valid {
		_ = k.SetStringValue("ProxyOverride", override)
	}
	return nil
}

func (m *WinManager) saveBackupFile() error {
	if m.backupPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.backupPath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m.snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.backupPath, b, 0o644)
}

func (m *WinManager) loadBackupFile() error {
	if m.backupPath == "" {
		return nil
	}
	b, err := os.ReadFile(m.backupPath)
	if err != nil {
		return err
	}
	var s Snapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.Valid {
		m.snap = s
	}
	return nil
}

func notify() error {
	wininet := windows.NewLazySystemDLL("wininet.dll")
	proc := wininet.NewProc("InternetSetOptionW")
	const (
		INTERNET_OPTION_SETTINGS_CHANGED = 39
		INTERNET_OPTION_REFRESH          = 37
	)
	r1, _, err := proc.Call(0, INTERNET_OPTION_SETTINGS_CHANGED, 0, 0)
	if r1 == 0 {
		return fmt.Errorf("InternetSetOption SETTINGS_CHANGED: %w", err)
	}
	r1, _, err = proc.Call(0, INTERNET_OPTION_REFRESH, 0, 0)
	if r1 == 0 {
		return fmt.Errorf("InternetSetOption REFRESH: %w", err)
	}
	// Also broadcast WM_SETTINGCHANGE for good measure (best effort).
	user32 := windows.NewLazySystemDLL("user32.dll")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")
	hwndBroadcast := uintptr(0xffff)
	wmSettingChange := uintptr(0x001A)
	ptr, err2 := windows.UTF16PtrFromString("Internet Settings")
	if err2 != nil {
		return nil
	}
	var result uintptr
	_, _, _ = sendMessageTimeout.Call(
		hwndBroadcast,
		wmSettingChange,
		0,
		uintptr(unsafe.Pointer(ptr)),
		0x0002, // SMTO_ABORTIFHUNG
		1000,
		uintptr(unsafe.Pointer(&result)),
	)
	return nil
}
