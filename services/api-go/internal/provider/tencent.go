package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func searchTencent(client *http.Client, q string) ([]Stock, error) {
	u := "https://smartbox.gtimg.cn/s3/?v=2&q=" + url.QueryEscape(q) + "&t=gp"
	raw, err := getBytes(client, u, "https://gu.qq.com/")
	if err != nil {
		return nil, err
	}
	text := decodeMaybeGBK(raw)
	start := strings.Index(text, `="`)
	end := strings.LastIndex(text, `"`)
	if start < 0 || end <= start+2 {
		return nil, fmt.Errorf("tencent suggest empty")
	}
	body := unescapeUnicode(text[start+2 : end])
	rows := strings.Split(body, "^")
	out := make([]Stock, 0, len(rows))
	for _, row := range rows {
		parts := strings.Split(row, "~")
		if len(parts) < 3 {
			continue
		}
		mkt := strings.ToLower(parts[0])
		code := PadSymbol(parts[1])
		name := strings.TrimSpace(parts[2])
		if code == "" || name == "" {
			continue
		}
		market := MarketSZ
		if mkt == "sh" {
			market = MarketSH
		}
		if mkt != "sh" && mkt != "sz" {
			continue
		}
		out = append(out, Stock{Symbol: code, Name: name, Market: market})
	}
	return out, nil
}

func fetchTencentQuote(client *http.Client, symbol string) (*Quote, error) {
	ms := MarketSymbol(symbol, GuessMarket(symbol))
	u := "https://qt.gtimg.cn/q=" + ms
	raw, err := getBytes(client, u, "https://gu.qq.com/")
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(decodeMaybeGBK(raw))
	i := strings.Index(text, `"`)
	j := strings.LastIndex(text, `"`)
	if i < 0 || j <= i {
		return nil, fmt.Errorf("tencent quote empty")
	}
	parts := strings.Split(text[i+1:j], "~")
	if len(parts) < 35 {
		return nil, fmt.Errorf("tencent quote fields")
	}
	return &Quote{
		Stock: Stock{
			Symbol: PadSymbol(parts[2]),
			Name:   parts[1],
			Market: GuessMarket(parts[2]),
		},
		Price:         parseF(parts[3]),
		Change:        parseF(parts[31]),
		ChangePercent: parseF(parts[32]),
		Open:          parseF(parts[5]),
		High:          parseF(parts[33]),
		Low:           parseF(parts[34]),
		PrevClose:     parseF(parts[4]),
		Volume:        parseF(parts[36]),
		Amount:        parseF(parts[37]),
		Turnover:      parseF(parts[38]),
	}, nil
}

func (b *Bundle) KlinesTencent(symbol string, limit int, market Market) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 180
	}
	ms := MarketSymbol(symbol, market)
	urls := []string{
		fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/newfqkline/get?param=%s,day,,,%d,qfq", ms, limit),
		fmt.Sprintf("https://proxy.finance.qq.com/ifzqgtimg/appstock/app/fqkline/get?param=%s,day,,,%d,qfq", ms, limit),
		fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,day,,,%d,qfq", ms, limit),
	}
	var last error
	for _, u := range urls {
		var payload map[string]interface{}
		if err := getJSON(b.Client, u, "https://gu.qq.com/", &payload); err != nil {
			last = err
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		node, _ := data[ms].(map[string]interface{})
		raw, _ := node["qfqday"].([]interface{})
		if len(raw) == 0 {
			raw, _ = node["day"].([]interface{})
		}
		bars, err := parseTencentKlines(raw)
		if err != nil {
			last = err
			continue
		}
		return bars, nil
	}
	if last == nil {
		last = fmt.Errorf("tencent kline empty")
	}
	return nil, last
}

func parseTencentKlines(raw []interface{}) ([]KlineBar, error) {
	out := make([]KlineBar, 0, len(raw))
	var prev float64
	for _, row := range raw {
		arr, ok := row.([]interface{})
		if !ok || len(arr) < 6 {
			if s, ok := row.([]string); ok && len(s) >= 6 {
				arr = make([]interface{}, len(s))
				for i := range s {
					arr[i] = s[i]
				}
			} else {
				continue
			}
		}
		open := asFloat(arr[1])
		close := asFloat(arr[2])
		high := asFloat(arr[3])
		low := asFloat(arr[4])
		volume := asFloat(arr[5])
		chg := 0.0
		if prev > 0 {
			chg = (close - prev) / prev * 100
		}
		out = append(out, KlineBar{
			Date:          asString(arr[0]),
			Open:          open,
			Close:         close,
			High:          high,
			Low:           low,
			Volume:        volume,
			ChangePercent: chg,
		})
		prev = close
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tencent kline empty")
	}
	return out, nil
}

func unescapeUnicode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if i+5 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
			if n, err := strconv.ParseInt(s[i+2:i+6], 16, 32); err == nil {
				b.WriteRune(rune(n))
				i += 6
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
