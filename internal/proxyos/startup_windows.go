//go:build windows

package proxyos

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const runValueName = "ProxyTray"

func EnableStartWithWindows(exePath string) error {
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return err
		}
	}
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(runValueName, fmt.Sprintf("%q", exePath))
}

func DisableStartWithWindows() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	err = k.DeleteValue(runValueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

func IsStartWithWindowsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValueName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
