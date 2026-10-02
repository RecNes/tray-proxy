package proxyos

// Manager applies and restores OS-wide HTTP(S) proxy settings.
type Manager interface {
	Apply(addr string) error
	Restore() error
	IsApplied() bool
	Active() string
}

// Snapshot is a backup of WinINET proxy settings before we change them.
type Snapshot struct {
	ProxyEnable   uint32 `json:"proxy_enable"`
	ProxyServer   string `json:"proxy_server"`
	ProxyOverride string `json:"proxy_override"`
	Valid         bool   `json:"valid"`
}
