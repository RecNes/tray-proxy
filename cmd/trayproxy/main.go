package main

import (
	"log"
	"os"
	"path/filepath"

	"tray-proxy/internal/config"
	"tray-proxy/internal/core"
	"tray-proxy/internal/proxyos"
	"tray-proxy/internal/tray"
)

// Set via: go build -ldflags "-X main.version=0.1.0"
var version = "dev"

func main() {
	store := config.DefaultStore()
	_ = os.MkdirAll(store.Root, 0o755)

	logFile, err := os.OpenFile(store.LogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf("trayproxy starting version=%s", version)

	cfg, err := store.Load()
	if err != nil {
		logger.Fatal(err)
	}
	cache, err := store.LoadCache()
	if err != nil {
		logger.Printf("load cache: %v", err)
	}

	mgr := proxyos.New(filepath.Join(store.Root, "proxy_backup.json"))
	scanner := core.NewScanner(nil, nil)

	tray.Run(tray.Deps{
		Store:   store,
		Config:  cfg,
		Cache:   cache,
		Scanner: scanner,
		Proxy:   mgr,
		Log:     logger,
	})
}
