package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var emHosts = []string{
	"https://push2.eastmoney.com",
	"https://82.push2.eastmoney.com",
	"https://push2delay.eastmoney.com",
}

var emKlineHosts = []string{
	"https://push2his.eastmoney.com",
	"https://push2delay.eastmoney.com",
}

func NewProviders(timeout time.Duration) *Bundle {
	client := newClient(timeout)
	return &Bundle{Client: client, Timeout: timeout}
}

type Bundle struct {
	Client  *http.Client
	Timeout time.Duration
}

func (b *Bundle) Search(q string) ([]Stock, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, fmt.Errorf("empty query")
	}
	if items, err := searchTencent(b.Client, q); err == nil && len(items) > 0 {
		return items, nil
	}
	return searchEastmoney(b.Client, q)
}

func searchEastmoney(client *http.Client, q string) ([]Stock, error) {
	u := "https://searchapi.eastmoney.com/api/suggest/get?input=" + url.QueryEscape(q) +
		"&type=14&token=D43BF722C8E33BDC906FB84D85E326E8&count=8"
	var payload map[string]interface{}
	if err := getJSON(client, u, "https://www.eastmoney.com/", &payload); err != nil {
		return nil, err
	}
	raw, _ := payload["QuotationCodeTable"].(map[string]interface{})
	list, _ := raw["Data"].([]interface{})
	out := make([]Stock, 0, len(list))
	for _, item := range list {
		m, _ := item.(map[string]interface{})
		code := PadSymbol(asString(m["Code"]))
		name := asString(m["Name"])
		sec := strings.ToLower(asString(m["SecurityTypeName"]) + asString(m["MktNum"]))
		if code == "" || name == "" {
			continue
		}
		if strings.Contains(sec, "指数") {
			continue
		}
		market := GuessMarket(code)
		if asString(m["MktNum"]) == "1" {
			market = MarketSH
		}
		if asString(m["MktNum"]) == "0" {
			market = MarketSZ
		}
		out = append(out, Stock{Symbol: code, Name: name, Market: market})
	}
	return out, nil
}

func (b *Bundle) Quote(symbol string) (*Quote, error) {
	symbol = PadSymbol(symbol)
	q, err := fetchEMQuote(b.Client, symbol)
	if err != nil || q == nil || q.Price <= 0 {
		q, err = fetchTencentQuote(b.Client, symbol)
		if err != nil {
			return nil, err
		}
	}
	enrichQuoteMeta(b.Client, q)
	return q, nil
}

func fetchEMQuote(client *http.Client, symbol string) (*Quote, error) {
	fields := "f12,f13,f14,f2,f3,f4,f5,f6,f7,f8,f10,f15,f16,f17,f18,f20,f62,f66,f72,f100,f102,f103,f184"
	query := "fltt=2&invt=2&fields=" + url.QueryEscape(fields) + "&secids=" + url.QueryEscape(SecID(symbol, GuessMarket(symbol)))
	var lastErr error
	for _, host := range emHosts {
		var payload map[string]interface{}
		err := getJSON(client, host+"/api/qt/ulist.np/get?"+query, "https://quote.eastmoney.com/", &payload)
		if err != nil {
			lastErr = err
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		diff, _ := data["diff"].([]interface{})
		if len(diff) == 0 {
			continue
		}
		m, _ := diff[0].(map[string]interface{})
		q := quoteFromEM(m)
		if q != nil {
			return q, nil
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("empty quote")
	}
	return nil, lastErr
}

func quoteFromEM(m map[string]interface{}) *Quote {
	code := PadSymbol(asString(m["f12"]))
	name := asString(m["f14"])
	if code == "" || name == "" {
		return nil
	}
	market := MarketSZ
	if asFloat(m["f13"]) == 1 {
		market = MarketSH
	}
	price := asFloat(m["f2"])
	prev := asFloat(m["f18"])
	change := asFloat(m["f4"])
	if change == 0 && prev > 0 {
		change = price - prev
	}
	concepts := splitConcepts(asString(m["f103"]))
	return &Quote{
		Stock:            Stock{Symbol: code, Name: name, Market: market},
		Price:            price,
		Change:           change,
		ChangePercent:    asFloat(m["f3"]),
		Open:             asFloat(m["f17"]),
		High:             asFloat(m["f15"]),
		Low:              asFloat(m["f16"]),
		PrevClose:        prev,
		Volume:           asFloat(m["f5"]),
		Amount:           asFloat(m["f6"]),
		Turnover:         asFloat(m["f8"]),
		VolumeRatio:      asFloat(m["f10"]),
		Amplitude:        asFloat(m["f7"]),
		Industry:         asString(m["f100"]),
		Region:           asString(m["f102"]),
		Concepts:         concepts,
		MainNetInflow:    asFloat(m["f62"]),
		MainNetInflowPct: asFloat(m["f184"]),
		SuperNetInflow:   asFloat(m["f66"]),
		BigNetInflow:     asFloat(m["f72"]),
	}
}

// enrichQuoteMeta 用 stock/get 补行业（f127 优先）与概念；ulist 的 f100 经常为空。
// stock/get 若被断开，再走 F10 CoreConception（ssbk 板块列表）。
func enrichQuoteMeta(client *http.Client, q *Quote) {
	if q == nil || q.Symbol == "" {
		return
	}
	needIndustry := strings.TrimSpace(q.Industry) == ""
	needConcepts := len(q.Concepts) == 0
	if !needIndustry && !needConcepts {
		return
	}
	secid := SecID(q.Symbol, q.Market)
	fields := "f57,f58,f100,f102,f103,f127"
	path := "/api/qt/stock/get?secid=" + url.QueryEscape(secid) + "&fields=" + url.QueryEscape(fields)
	for _, host := range emHosts {
		var payload map[string]interface{}
		if err := getJSON(client, host+path, "https://quote.eastmoney.com/", &payload); err != nil {
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		if data == nil {
			continue
		}
		if needIndustry {
			ind := asString(data["f127"])
			if ind == "" {
				ind = asString(data["f100"])
			}
			if ind != "" {
				q.Industry = ind
				needIndustry = false
			}
		}
		if q.Region == "" {
			q.Region = asString(data["f102"])
		}
		if needConcepts {
			concepts := splitConcepts(asString(data["f103"]))
			if len(concepts) > 0 {
				q.Concepts = concepts
				needConcepts = false
			}
		}
		if !needIndustry && !needConcepts {
			return
		}
	}
	if needIndustry || needConcepts {
		enrichFromF10Boards(client, q)
	}
}

// enrichFromF10Boards 走东财 F10「核心题材」接口，ssbk 第一项通常为行业，其余为概念板块。
func enrichFromF10Boards(client *http.Client, q *Quote) {
	prefix := "SZ"
	if q.Market == MarketSH || GuessMarket(q.Symbol) == MarketSH {
		prefix = "SH"
	}
	u := "https://emweb.securities.eastmoney.com/PC_HSF10/CoreConception/PageAjax?code=" + prefix + q.Symbol
	var payload map[string]interface{}
	if err := getJSON(client, u, "https://emweb.securities.eastmoney.com/", &payload); err != nil {
		return
	}
	raw, _ := payload["ssbk"].([]interface{})
	if len(raw) == 0 {
		return
	}
	names := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, row := range raw {
		m, _ := row.(map[string]interface{})
		name := asString(m["BOARD_NAME"])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	if len(names) == 0 {
		return
	}
	if strings.TrimSpace(q.Industry) == "" {
		q.Industry = names[0]
	}
	if len(q.Concepts) == 0 {
		start := 0
		if q.Industry == names[0] {
			start = 1
		}
		if start < len(names) {
			q.Concepts = names[start:]
			if len(q.Concepts) > 8 {
				q.Concepts = q.Concepts[:8]
			}
		}
	}
}

func splitConcepts(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '/' || r == '|'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func (b *Bundle) IndexQuotes() ([]Quote, error) {
	items := []struct {
		symbol string
		market Market
	}{
		{"000001", MarketSH},
		{"399001", MarketSZ},
		{"399006", MarketSZ},
	}
	secids := make([]string, 0, len(items))
	for _, it := range items {
		secids = append(secids, SecID(it.symbol, it.market))
	}
	fields := "f12,f13,f14,f2,f3,f4,f15,f16,f17,f18,f5,f6"
	query := "fltt=2&invt=2&fields=" + url.QueryEscape(fields) + "&secids=" + url.QueryEscape(strings.Join(secids, ","))
	for _, host := range emHosts {
		var payload map[string]interface{}
		if err := getJSON(b.Client, host+"/api/qt/ulist.np/get?"+query, "https://quote.eastmoney.com/", &payload); err != nil {
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		diff, _ := data["diff"].([]interface{})
		out := make([]Quote, 0, len(diff))
		for _, item := range diff {
			m, _ := item.(map[string]interface{})
			q := quoteFromEM(m)
			if q != nil {
				out = append(out, *q)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("index quotes empty")
}

func (b *Bundle) KlinesEM(symbol string, limit int, market Market) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 180
	}
	path := fmt.Sprintf("/api/qt/stock/kline/get?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61&klt=101&fqt=1&end=20500101&lmt=%d",
		SecID(symbol, market), limit)
	var lastErr error
	for _, host := range emKlineHosts {
		var payload map[string]interface{}
		if err := getJSON(b.Client, host+path, "https://quote.eastmoney.com/", &payload); err != nil {
			lastErr = err
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		raw, _ := data["klines"].([]interface{})
		if len(raw) == 0 {
			continue
		}
		out := make([]KlineBar, 0, len(raw))
		for _, line := range raw {
			parts := strings.Split(asString(line), ",")
			if len(parts) < 7 {
				continue
			}
			out = append(out, KlineBar{
				Date:          parts[0],
				Open:          parseF(parts[1]),
				Close:         parseF(parts[2]),
				High:          parseF(parts[3]),
				Low:           parseF(parts[4]),
				Volume:        parseF(parts[5]),
				Amount:        parseF(parts[6]),
				ChangePercent: pickF(parts, 8),
				Turnover:      pickF(parts, 9),
			})
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("em kline empty")
	}
	return nil, lastErr
}

func (b *Bundle) MarketClock() (string, error) {
	for _, host := range emHosts {
		var payload map[string]interface{}
		if err := getJSON(b.Client, host+"/api/qt/stock/get?secid=1.000001&fields=f43,f86,f58", "https://quote.eastmoney.com/", &payload); err != nil {
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		ts := asFloat(data["f86"])
		if ts <= 0 {
			continue
		}
		t := time.Unix(int64(ts), 0).In(time.FixedZone("CST", 8*3600))
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("market clock unavailable")
}

func parseF(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func pickF(parts []string, i int) float64 {
	if i >= len(parts) {
		return 0
	}
	return parseF(parts[i])
}
