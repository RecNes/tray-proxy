package core

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const ProxyListURL = "https://free-proxy-list.net/"

func ParseProxies(html string, limit int) ([]Proxy, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	var out []Proxy
	doc.Find("tbody tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		if limit > 0 && len(out) >= limit {
			return false
		}
		tds := tr.Find("td")
		if tds.Length() < 7 {
			return true
		}
		ip := strings.TrimSpace(tds.Eq(0).Text())
		port := strings.TrimSpace(tds.Eq(1).Text())
		if ip == "" || port == "" {
			return true
		}
		anon := Anonymity(strings.ToLower(strings.TrimSpace(tds.Eq(4).Text())))
		https := strings.EqualFold(strings.TrimSpace(tds.Eq(6).Text()), "yes")
		out = append(out, Proxy{
			IP:        ip,
			Port:      port,
			Code:      strings.TrimSpace(tds.Eq(2).Text()),
			Country:   strings.TrimSpace(tds.Eq(3).Text()),
			Anonymity: anon,
			HTTPS:     https,
		})
		return true
	})
	return out, nil
}

func FetchProxyList(client *http.Client) ([]Proxy, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Get(ProxyListURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("proxy list HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ParseProxies(string(body), 100)
}
