//go:build !windows

package proxyos

func EnableStartWithWindows(exePath string) error  { return nil }
func DisableStartWithWindows() error               { return nil }
func IsStartWithWindowsEnabled() (bool, error)     { return false, nil }
