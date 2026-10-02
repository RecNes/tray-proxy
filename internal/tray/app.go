package tray

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/energye/systray"

	"proxy-tray/internal/config"
	"proxy-tray/internal/core"
	"proxy-tray/internal/proxyos"
)

const menuTopN = 10
const scanInterval = 30 * time.Minute

type Deps struct {
	Store   config.Store
	Config  config.Config
	Cache   []core.Proxy
	Scanner *core.Scanner
	Proxy   proxyos.Manager
	Log     *log.Logger
}

type App struct {
	deps       Deps
	mu         sync.Mutex
	cfg        config.Config
	cache      []core.Proxy
	active     string // applied proxy address
	status     string
	lastScanAt time.Time
}

func Run(deps Deps) {
	if deps.Log == nil {
		f, err := os.OpenFile(deps.Store.LogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			deps.Log = log.New(f, "", log.LstdFlags)
		} else {
			deps.Log = log.Default()
		}
	}
	app := &App{
		deps:  deps,
		cfg:   deps.Config,
		cache: append([]core.Proxy(nil), deps.Cache...),
	}
	if info, err := os.Stat(deps.Store.CachePath()); err == nil && !info.ModTime().IsZero() {
		app.lastScanAt = info.ModTime()
	}
	systray.Run(app.onReady, app.onExit)
}

func (a *App) onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Proxy Tray")
	systray.SetTooltip("Proxy Tray")
	systray.SetOnClick(func(menu systray.IMenu) {
		if menu != nil {
			menu.ShowMenu()
		}
	})
	systray.SetOnRClick(func(menu systray.IMenu) {
		if menu != nil {
			menu.ShowMenu()
		}
	})
	systray.CreateMenu()
	a.rebuildMenu()
	go a.scanAndRefresh()
	go a.periodicScan()
}

func (a *App) onExit() {
	if err := a.deps.Proxy.Restore(); err != nil {
		a.deps.Log.Printf("restore on exit: %v", err)
	}
}

func (a *App) periodicScan() {
	t := time.NewTicker(scanInterval)
	defer t.Stop()
	for range t.C {
		a.scanAndRefresh()
	}
}

func (a *App) scanAndRefresh() {
	if a.deps.Scanner.IsScanning() {
		return
	}
	a.setStatus("Scanning…")
	a.rebuildMenu()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	result, err := a.deps.Scanner.Run(ctx)
	if err != nil {
		if err == core.ErrScanInProgress {
			return
		}
		a.deps.Log.Printf("scan failed: %v", err)
		a.setStatus("Refresh failed")
		a.rebuildMenu()
		return
	}
	a.mu.Lock()
	a.cache = result
	a.lastScanAt = time.Now()
	a.mu.Unlock()
	if err := a.deps.Store.SaveCache(result); err != nil {
		a.deps.Log.Printf("save cache: %v", err)
	}
	a.setStatus("")
	a.rebuildMenu()
}

func (a *App) setStatus(s string) {
	a.mu.Lock()
	a.status = s
	a.mu.Unlock()
	tip := "Proxy Tray"
	if s != "" {
		tip = s
	} else if a.deps.Proxy.IsApplied() {
		tip = "Proxy: " + a.deps.Proxy.Active()
	}
	systray.SetTooltip(tip)
}

func (a *App) rebuildMenu() {
	a.mu.Lock()
	cfg := a.cfg
	cache := append([]core.Proxy(nil), a.cache...)
	active := a.active
	if active == "" {
		active = a.deps.Proxy.Active()
	}
	scanning := a.deps.Scanner.IsScanning()
	lastScan := a.lastScanAt
	a.mu.Unlock()

	systray.ResetMenu()

	off := systray.AddMenuItemCheckbox("Off", "Disable system proxy", active == "")
	off.Click(func() { a.onOff() })

	systray.AddSeparator()

	updatedLabel := "Updated: never"
	if scanning {
		updatedLabel = "Updated: scanning…"
	} else if !lastScan.IsZero() {
		updatedLabel = "Updated: " + lastScan.Format("02.01.2006 15:04:05")
	}
	updatedItem := systray.AddMenuItem(updatedLabel, "Last successful proxy list refresh")
	updatedItem.Disable()

	items := core.MenuProxies(cache, cfg.Filters, menuTopN)
	shown := map[string]bool{}
	for _, p := range items {
		shown[p.Address()] = true
		code := strings.ToUpper(p.Code)
		label := fmt.Sprintf("%s  %s  (%.1fs)", code, p.Address(), p.LatencyS)
		checked := p.Address() == active
		item := systray.AddMenuItemCheckbox(label, p.Country, checked)
		if ico := flagIconICO(p.Code); len(ico) > 0 {
			item.SetIcon(ico)
		}
		addr := p.Address()
		item.Click(func() { a.onSelect(addr) })
	}
	if active != "" && !shown[active] {
		var orphan core.Proxy
		found := false
		for _, p := range cache {
			if p.Address() == active {
				orphan = p
				found = true
				break
			}
		}
		label := "Active  " + active
		if found {
			label = fmt.Sprintf("%s  %s  (active)", strings.ToUpper(orphan.Code), orphan.Address())
		}
		item := systray.AddMenuItemCheckbox(label, "Currently applied", true)
		if found {
			if ico := flagIconICO(orphan.Code); len(ico) > 0 {
				item.SetIcon(ico)
			}
		}
		addr := active
		item.Click(func() { a.onSelect(addr) })
	}

	systray.AddSeparator()

	refreshTitle := "Refresh now"
	if scanning {
		refreshTitle = "Scanning…"
	}
	refresh := systray.AddMenuItem(refreshTitle, "Fetch and test proxies")
	if scanning {
		refresh.Disable()
	} else {
		refresh.Click(func() { go a.scanAndRefresh() })
	}

	filters := systray.AddMenuItem("Filters", "Filter menu list")
	httpsItem := filters.AddSubMenuItemCheckbox("HTTPS", "Require HTTPS", cfg.Filters.HTTPS)
	eliteItem := filters.AddSubMenuItemCheckbox("Elite", "Include elite", cfg.Filters.Elite)
	anonItem := filters.AddSubMenuItemCheckbox("Anonymous", "Include anonymous", cfg.Filters.Anonymous)
	transItem := filters.AddSubMenuItemCheckbox("Transparent", "Include transparent", cfg.Filters.Transparent)
	httpsItem.Click(func() { a.toggleFilter("https") })
	eliteItem.Click(func() { a.toggleFilter("elite") })
	anonItem.Click(func() { a.toggleFilter("anonymous") })
	transItem.Click(func() { a.toggleFilter("transparent") })

	startWin, _ := proxyos.IsStartWithWindowsEnabled()
	startItem := systray.AddMenuItemCheckbox("Start with Windows", "Launch at login", startWin || cfg.StartWithWindows)
	startItem.Click(func() { a.toggleStartWithWindows() })

	quit := systray.AddMenuItem("Quit", "Restore proxy and exit")
	quit.Click(func() {
		_ = a.deps.Proxy.Restore()
		systray.Quit()
	})
}

func (a *App) onOff() {
	if err := a.deps.Proxy.Restore(); err != nil {
		a.deps.Log.Printf("off failed: %v", err)
		a.setStatus("Failed to clear proxy")
		return
	}
	a.mu.Lock()
	a.active = ""
	a.mu.Unlock()
	a.setStatus("")
	a.rebuildMenu()
}

func (a *App) onSelect(addr string) {
	if err := a.deps.Proxy.Apply(addr); err != nil {
		a.deps.Log.Printf("apply %s: %v", addr, err)
		a.setStatus("Failed to apply proxy")
		return
	}
	a.mu.Lock()
	a.active = addr
	a.mu.Unlock()
	a.setStatus("")
	a.rebuildMenu()
}

func (a *App) toggleFilter(name string) {
	a.mu.Lock()
	switch name {
	case "https":
		a.cfg.Filters.HTTPS = !a.cfg.Filters.HTTPS
	case "elite":
		a.cfg.Filters.Elite = !a.cfg.Filters.Elite
	case "anonymous":
		a.cfg.Filters.Anonymous = !a.cfg.Filters.Anonymous
	case "transparent":
		a.cfg.Filters.Transparent = !a.cfg.Filters.Transparent
	}
	cfg := a.cfg
	a.mu.Unlock()
	if err := a.deps.Store.Save(cfg); err != nil {
		a.deps.Log.Printf("save config: %v", err)
	}
	a.rebuildMenu()
}

func (a *App) toggleStartWithWindows() {
	a.mu.Lock()
	enable := !a.cfg.StartWithWindows
	a.cfg.StartWithWindows = enable
	cfg := a.cfg
	a.mu.Unlock()
	var err error
	if enable {
		err = proxyos.EnableStartWithWindows("")
	} else {
		err = proxyos.DisableStartWithWindows()
	}
	if err != nil {
		a.deps.Log.Printf("start with windows: %v", err)
		a.setStatus("Startup toggle failed")
		return
	}
	if err := a.deps.Store.Save(cfg); err != nil {
		a.deps.Log.Printf("save config: %v", err)
	}
	a.rebuildMenu()
}
