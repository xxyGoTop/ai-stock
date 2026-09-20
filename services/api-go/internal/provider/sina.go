package provider

import (
	"fmt"
)

func (b *Bundle) KlinesSina(symbol string, limit int, market Market) ([]KlineBar, error) {
	if limit <= 0 {
		limit = 180
	}
	ms := MarketSymbol(symbol, market)
	u := fmt.Sprintf("https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData?symbol=%s&scale=240&ma=5&datalen=%d", ms, limit)
	var raw []map[string]interface{}
	if err := getJSON(b.Client, u, "https://finance.sina.com.cn/", &raw); err != nil {
		return nil, err
	}
	out := make([]KlineBar, 0, len(raw))
	var prev float64
	for _, d := range raw {
		open := asFloat(d["open"])
		close := asFloat(d["close"])
		high := asFloat(d["high"])
		low := asFloat(d["low"])
		volume := asFloat(d["volume"])
		chg := 0.0
		if prev > 0 {
			chg = (close - prev) / prev * 100
		}
		out = append(out, KlineBar{
			Date:          asString(d["day"]),
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
		return nil, fmt.Errorf("sina kline empty")
	}
	return out, nil
}
