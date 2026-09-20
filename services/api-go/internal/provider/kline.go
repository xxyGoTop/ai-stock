package provider

import (
	"fmt"
	"time"
)

type KlineResult struct {
	Bars       []KlineBar `json:"bars"`
	Freshness  Freshness  `json:"freshness"`
	SourceUsed string     `json:"sourceUsed"`
}

func lastDate(bars []KlineBar) string {
	if len(bars) == 0 {
		return ""
	}
	d := bars[len(bars)-1].Date
	if len(d) >= 10 {
		return d[:10]
	}
	return d
}

func (b *Bundle) AggregatedKline(symbol string, limit int) (*KlineResult, error) {
	symbol = PadSymbol(symbol)
	market := GuessMarket(symbol)
	type attempt struct {
		name string
		fn   func() ([]KlineBar, error)
	}
	fetchLimit := limit
	if fetchLimit < 60 {
		fetchLimit = 60
	}
	tries := []attempt{
		{"tencent", func() ([]KlineBar, error) { return b.KlinesTencent(symbol, fetchLimit, market) }},
		{"sina", func() ([]KlineBar, error) { return b.KlinesSina(symbol, fetchLimit, market) }},
		{"eastmoney", func() ([]KlineBar, error) { return b.KlinesEM(symbol, fetchLimit, market) }},
	}

	quoteDate, _ := b.MarketClock()
	var best []KlineBar
	used := ""
	for _, t := range tries {
		bars, err := t.fn()
		if err != nil || len(bars) < 10 {
			continue
		}
		if lastDate(bars) > lastDate(best) {
			best = bars
			used = t.name
		}
		if quoteDate != "" && lastDate(bars) >= quoteDate {
			best = bars
			used = t.name
			break
		}
	}
	if len(best) == 0 {
		return nil, fmt.Errorf("all kline sources failed")
	}

	patched := false
	if quoteDate != "" && lastDate(best) < quoteDate {
		if q, err := b.Quote(symbol); err == nil && q != nil && q.Price > 0 {
			best = mergeLiveBar(best, q, quoteDate)
			patched = true
		}
	}

	klineDate := lastDate(best)
	stale := false
	msg := fmt.Sprintf("K线基准日 %s", klineDate)
	if quoteDate != "" && klineDate != "" && klineDate < quoteDate && isTradingDay(time.Now()) {
		stale = true
		msg = fmt.Sprintf("实时行情已到 %s，但K线最新只到 %s，指标不含当日走势", quoteDate, klineDate)
	}
	if patched {
		msg += "（已用实时快照补当日K）"
	}
	if limit > 0 && len(best) > limit {
		best = best[len(best)-limit:]
	}

	return &KlineResult{
		Bars: best,
		Freshness: Freshness{
			KlineDate:       klineDate,
			QuoteDate:       quoteDate,
			Stale:           stale,
			PatchedIntraday: patched,
			Message:         msg,
		},
		SourceUsed: used,
	}, nil
}

func mergeLiveBar(bars []KlineBar, live *Quote, date string) []KlineBar {
	if len(bars) == 0 || live == nil || live.Price <= 0 || date == "" {
		return bars
	}
	last := bars[len(bars)-1]
	lastDay := last.Date
	if len(lastDay) >= 10 {
		lastDay = lastDay[:10]
	}
	if lastDay > date {
		return bars
	}
	prev := last.Close
	if lastDay == date && live.PrevClose > 0 {
		prev = live.PrevClose
	}
	open := live.Open
	if open <= 0 {
		open = prev
	}
	high := live.High
	if high < open {
		high = open
	}
	if high < live.Price {
		high = live.Price
	}
	low := live.Low
	if low <= 0 || low > live.Price {
		low = live.Price
	}
	bar := KlineBar{
		Date:          date,
		Open:          open,
		High:          high,
		Low:           low,
		Close:         live.Price,
		Volume:        live.Volume,
		Amount:        live.Amount,
		ChangePercent: live.ChangePercent,
		Turnover:      live.Turnover,
		Intraday:      true,
	}
	if lastDay == date {
		out := make([]KlineBar, len(bars))
		copy(out, bars)
		out[len(out)-1] = bar
		return out
	}
	return append(bars, bar)
}

func isTradingDay(t time.Time) bool {
	t = t.In(time.FixedZone("CST", 8*3600))
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 9*60+15
}
