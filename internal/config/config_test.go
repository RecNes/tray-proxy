package config

import (
	"testing"

	"proxy-tray/internal/core"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	st := Store{Root: t.TempDir()}
	cfg := Config{Filters: core.DefaultFilters(), StartWithWindows: false}
	cfg.Filters.Transparent = false
	if err := st.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Filters.Transparent || !got.Filters.Elite {
		t.Fatalf("%+v", got)
	}
	proxies := []core.Proxy{{IP: "1.1.1.1", Port: "80", LatencyS: 0.5}}
	if err := st.SaveCache(proxies); err != nil {
		t.Fatal(err)
	}
	c2, err := st.LoadCache()
	if err != nil || len(c2) != 1 || c2[0].IP != "1.1.1.1" {
		t.Fatalf("%v %+v", err, c2)
	}
}
