package core

import "sort"

func ApplyFilters(proxies []Proxy, f Filters) []Proxy {
	out := make([]Proxy, 0, len(proxies))
	for _, p := range proxies {
		if f.HTTPS && !p.HTTPS {
			continue
		}
		switch p.Anonymity {
		case AnonymityElite:
			if !f.Elite {
				continue
			}
		case AnonymityAnonymous:
			if !f.Anonymous {
				continue
			}
		case AnonymityTransparent:
			if !f.Transparent {
				continue
			}
		default:
			continue
		}
		out = append(out, p)
	}
	return out
}

func TopN(proxies []Proxy, n int) []Proxy {
	cp := append([]Proxy(nil), proxies...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].LatencyS < cp[j].LatencyS })
	if n < 0 {
		n = 0
	}
	if len(cp) > n {
		cp = cp[:n]
	}
	return cp
}

// MenuProxies applies filters then returns fastest n.
func MenuProxies(validated []Proxy, f Filters, n int) []Proxy {
	return TopN(ApplyFilters(validated, f), n)
}
