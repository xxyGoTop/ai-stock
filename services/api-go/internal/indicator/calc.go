package indicator

import "github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"

type Point struct {
	Date  string   `json:"date"`
	MA5   *float64 `json:"ma5"`
	MA10  *float64 `json:"ma10"`
	MA20  *float64 `json:"ma20"`
	DIF   *float64 `json:"dif"`
	DEA   *float64 `json:"dea"`
	Hist  *float64 `json:"hist"`
	RSI6  *float64 `json:"rsi6"`
	K     *float64 `json:"k"`
	D     *float64 `json:"d"`
	J     *float64 `json:"j"`
	Bias5 *float64 `json:"bias5"`
}

type Result struct {
	Symbol    string             `json:"symbol"`
	Freshness provider.Freshness `json:"freshness"`
	Latest    *Point             `json:"latest"`
	Series    []Point            `json:"series"`
}

func Compute(bars []provider.KlineBar) []Point {
	n := len(bars)
	closes := make([]float64, n)
	for i, b := range bars {
		closes[i] = b.Close
	}
	ma5 := sma(closes, 5)
	ma10 := sma(closes, 10)
	ma20 := sma(closes, 20)
	dif, dea, hist := macd(closes, 12, 26, 9)
	rsi6 := rsi(closes, 6)
	k, d, j := kdj(bars, 9, 3, 3)

	out := make([]Point, n)
	for i, b := range bars {
		p := Point{Date: b.Date, MA5: ma5[i], MA10: ma10[i], MA20: ma20[i], DIF: dif[i], DEA: dea[i], Hist: hist[i], RSI6: rsi6[i], K: k[i], D: d[i], J: j[i]}
		if ma5[i] != nil && *ma5[i] != 0 {
			v := (b.Close - *ma5[i]) / *ma5[i] * 100
			p.Bias5 = &v
		}
		out[i] = p
	}
	return out
}

func sma(values []float64, period int) []*float64 {
	out := make([]*float64, len(values))
	if period <= 0 {
		return out
	}
	sum := 0.0
	for i := 0; i < len(values); i++ {
		sum += values[i]
		if i >= period {
			sum -= values[i-period]
		}
		if i >= period-1 {
			v := sum / float64(period)
			out[i] = &v
		}
	}
	return out
}

func ema(values []float64, period int) []*float64 {
	out := make([]*float64, len(values))
	k := 2 / (float64(period) + 1)
	var prev *float64
	sum := 0.0
	for i := 0; i < len(values); i++ {
		if i < period {
			sum += values[i]
			if i == period-1 {
				v := sum / float64(period)
				out[i] = &v
				prev = &v
			}
			continue
		}
		if prev == nil {
			continue
		}
		v := values[i]*k + *prev*(1-k)
		out[i] = &v
		prev = &v
	}
	return out
}

func macd(closes []float64, fast, slow, signal int) ([]*float64, []*float64, []*float64) {
	ef := ema(closes, fast)
	es := ema(closes, slow)
	dif := make([]*float64, len(closes))
	raw := make([]float64, len(closes))
	for i := range closes {
		if ef[i] != nil && es[i] != nil {
			v := *ef[i] - *es[i]
			dif[i] = &v
			raw[i] = v
		}
	}
	dea := ema(raw, signal)
	hist := make([]*float64, len(closes))
	for i := range closes {
		if dif[i] != nil && dea[i] != nil {
			v := (*dif[i] - *dea[i]) * 2
			hist[i] = &v
		} else {
			dea[i] = nil
		}
	}
	return dif, dea, hist
}

func rsi(closes []float64, period int) []*float64 {
	out := make([]*float64, len(closes))
	if len(closes) <= period {
		return out
	}
	gain, loss := 0.0, 0.0
	for i := 1; i <= period; i++ {
		diff := closes[i] - closes[i-1]
		if diff >= 0 {
			gain += diff
		} else {
			loss -= diff
		}
	}
	avgGain := gain / float64(period)
	avgLoss := loss / float64(period)
	out[period] = rsiValue(avgGain, avgLoss)
	for i := period + 1; i < len(closes); i++ {
		diff := closes[i] - closes[i-1]
		g, l := 0.0, 0.0
		if diff > 0 {
			g = diff
		} else {
			l = -diff
		}
		avgGain = (avgGain*float64(period-1) + g) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + l) / float64(period)
		out[i] = rsiValue(avgGain, avgLoss)
	}
	return out
}

func rsiValue(avgGain, avgLoss float64) *float64 {
	var v float64
	if avgLoss == 0 {
		v = 100
	} else {
		v = 100 - 100/(1+avgGain/avgLoss)
	}
	return &v
}

func kdj(bars []provider.KlineBar, n, m1, m2 int) ([]*float64, []*float64, []*float64) {
	kArr := make([]*float64, len(bars))
	dArr := make([]*float64, len(bars))
	jArr := make([]*float64, len(bars))
	k, d := 50.0, 50.0
	for i := range bars {
		if i < n-1 {
			continue
		}
		highest, lowest := bars[i].High, bars[i].Low
		for j := i - n + 1; j <= i; j++ {
			if bars[j].High > highest {
				highest = bars[j].High
			}
			if bars[j].Low < lowest {
				lowest = bars[j].Low
			}
		}
		rsv := 50.0
		if highest != lowest {
			rsv = (bars[i].Close - lowest) / (highest - lowest) * 100
		}
		k = (rsv + float64(m1-1)*k) / float64(m1)
		d = (k + float64(m2-1)*d) / float64(m2)
		j := 3*k - 2*d
		kk, dd, jj := k, d, j
		kArr[i], dArr[i], jArr[i] = &kk, &dd, &jj
	}
	return kArr, dArr, jArr
}
