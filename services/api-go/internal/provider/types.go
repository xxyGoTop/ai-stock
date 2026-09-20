package provider

type Market string

const (
	MarketSH Market = "SH"
	MarketSZ Market = "SZ"
)

type Stock struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market Market `json:"market"`
}

type Quote struct {
	Stock
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Open          float64 `json:"open"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	PrevClose     float64 `json:"prevClose"`
	Volume        float64 `json:"volume"`
	Amount        float64 `json:"amount"`
	Turnover      float64 `json:"turnover"`
	VolumeRatio   float64 `json:"volumeRatio"`
	Amplitude     float64 `json:"amplitude"`
	Industry      string  `json:"industry"`
}

type KlineBar struct {
	Date          string  `json:"date"`
	Open          float64 `json:"open"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Close         float64 `json:"close"`
	Volume        float64 `json:"volume"`
	Amount        float64 `json:"amount"`
	ChangePercent float64 `json:"changePercent"`
	Turnover      float64 `json:"turnover"`
	Intraday      bool    `json:"intraday,omitempty"`
}

type Freshness struct {
	KlineDate       string `json:"klineDate"`
	QuoteDate       string `json:"quoteDate"`
	Stale           bool   `json:"stale"`
	PatchedIntraday bool   `json:"patchedIntraday"`
	Message         string `json:"message"`
}

func GuessMarket(symbol string) Market {
	s := PadSymbol(symbol)
	if len(s) == 0 {
		return MarketSZ
	}
	switch s[0] {
	case '6', '9', '5':
		return MarketSH
	default:
		return MarketSZ
	}
}

func PadSymbol(symbol string) string {
	digits := make([]byte, 0, 6)
	for i := 0; i < len(symbol); i++ {
		c := symbol[i]
		if c >= '0' && c <= '9' {
			digits = append(digits, c)
		}
	}
	for len(digits) < 6 {
		digits = append([]byte{'0'}, digits...)
	}
	if len(digits) > 6 {
		digits = digits[len(digits)-6:]
	}
	return string(digits)
}

func SecID(symbol string, market Market) string {
	s := PadSymbol(symbol)
	if market == "" {
		market = GuessMarket(s)
	}
	if market == MarketSH {
		return "1." + s
	}
	return "0." + s
}

func MarketSymbol(symbol string, market Market) string {
	s := PadSymbol(symbol)
	if market == "" {
		market = GuessMarket(s)
	}
	if market == MarketSH {
		return "sh" + s
	}
	return "sz" + s
}
