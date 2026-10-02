package core

type Anonymity string

const (
	AnonymityElite       Anonymity = "elite proxy"
	AnonymityAnonymous   Anonymity = "anonymous"
	AnonymityTransparent Anonymity = "transparent"
)

type Proxy struct {
	IP        string    `json:"ip"`
	Port      string    `json:"port"`
	Code      string    `json:"code"`
	Country   string    `json:"country"`
	Anonymity Anonymity `json:"anonymity"`
	HTTPS     bool      `json:"https"`
	LatencyS  float64   `json:"latency_s"`
}

func (p Proxy) Address() string {
	return p.IP + ":" + p.Port
}

type Filters struct {
	HTTPS       bool `json:"https"`
	Elite       bool `json:"elite"`
	Anonymous   bool `json:"anonymous"`
	Transparent bool `json:"transparent"`
}

func DefaultFilters() Filters {
	return Filters{HTTPS: true, Elite: true, Anonymous: true, Transparent: true}
}
