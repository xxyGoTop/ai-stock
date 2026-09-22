package companion

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/indicator"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

var reAllCodes = regexp.MustCompile(`(?:sh|sz|SH|SZ)?[\s\.]*([0-9]{6})`)

func looksLikeCompare(msg string) bool {
	keys := []string{"对比", "比较", "比一比", "比一下", "相比", " vs ", " VS ", "vs", "VS"}
	for _, k := range keys {
		if strings.Contains(msg, k) {
			return true
		}
	}
	// 「A和B比」类，避免「比亚迪」里的「比」误触
	re := regexp.MustCompile(`(和|跟|与)[^和跟与]{1,10}(对比|比较|比$|比一|比下)`)
	return re.MatchString(msg)
}

func extractSymbols(msg string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 3)
	add := func(code string) {
		code = provider.PadSymbol(code)
		if code == "" || code == "000000" || seen[code] {
			return
		}
		seen[code] = true
		out = append(out, code)
	}
	for _, m := range reAllCodes.FindAllStringSubmatch(msg, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	aliases := []struct{ name, code string }{
		{"贵州茅台", "600519"}, {"茅台", "600519"},
		{"宁德时代", "300750"}, {"宁德", "300750"},
		{"比亚迪", "002594"},
		{"中国平安", "601318"}, {"平安", "601318"},
		{"招商银行", "600036"}, {"招行", "600036"},
		{"中芯国际", "688981"}, {"中芯", "688981"},
		{"寒武纪", "688256"}, {"海康威视", "002415"}, {"海康", "002415"},
		{"龙磁科技", "300835"}, {"风华高科", "000636"},
		{"江丰电子", "300666"}, {"科士达", "002518"},
	}
	for _, a := range aliases {
		if strings.Contains(msg, a.name) {
			add(a.code)
		}
	}
	if len(out) > 3 {
		return out[:3]
	}
	return out
}

func resolveNameToSymbol(s *Service, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if m := reAllCodes.FindStringSubmatch(name); len(m) > 1 {
		return provider.PadSymbol(m[1])
	}
	items, err := s.bundle.Search(name)
	if err != nil || len(items) == 0 {
		return ""
	}
	return items[0].Symbol
}

func (s *Service) compareStocks(msg string, preferred string) (*ChatResponse, error) {
	syms := extractSymbols(msg)
	if preferred != "" && preferred != "000000" {
		p := provider.PadSymbol(preferred)
		found := false
		for _, x := range syms {
			if x == p {
				found = true
				break
			}
		}
		if !found {
			syms = append([]string{p}, syms...)
		}
	}
	// 尝试从「A和B」「A vs B」拆中文名
	if len(syms) < 2 {
		parts := splitCompareNames(msg)
		for _, p := range parts {
			if len(syms) >= 2 {
				break
			}
			if code := resolveNameToSymbol(s, p); code != "" {
				dup := false
				for _, x := range syms {
					if x == code {
						dup = true
						break
					}
				}
				if !dup {
					syms = append(syms, code)
				}
			}
		}
	}
	if len(syms) < 2 {
		return &ChatResponse{
			Reply:  "对比需要两只股票，例如「茅台和比亚迪对比」或「600519 vs 002594」。",
			Intent: "compare",
			Blocks: []Block{{Type: "suggestions", Title: "试试", Items: []string{"茅台和比亚迪对比", "宁德时代 vs 比亚迪", "分析茅台"}}},
		}, nil
	}
	left, right := syms[0], syms[1]

	type side struct {
		q    *provider.Quote
		ind  *indicator.Point
		err  error
	}
	sides := make([]side, 2)
	var wg sync.WaitGroup
	for i, sym := range []string{left, right} {
		wg.Add(1)
		go func(i int, sym string) {
			defer wg.Done()
			q, err := s.bundle.Quote(sym)
			sides[i].q = q
			sides[i].err = err
			if res, e := s.bundle.AggregatedKline(sym, 120); e == nil && res != nil {
				series := indicator.Compute(res.Bars)
				if len(series) > 0 {
					p := series[len(series)-1]
					sides[i].ind = &p
				}
			}
		}(i, sym)
	}
	wg.Wait()

	if sides[0].q == nil || sides[1].q == nil {
		miss := left
		if sides[0].q != nil {
			miss = right
		}
		return &ChatResponse{Reply: "没拿到 " + miss + " 的行情，换一对再比。", Intent: "compare"}, nil
	}
	a, b := sides[0].q, sides[1].q

	rows := []map[string]interface{}{
		{"metric": "现价", "left": fmt.Sprintf("%.2f", a.Price), "right": fmt.Sprintf("%.2f", b.Price)},
		{"metric": "涨跌幅", "left": fmt.Sprintf("%+.2f%%", a.ChangePercent), "right": fmt.Sprintf("%+.2f%%", b.ChangePercent), "winner": pctWinner(a.ChangePercent, b.ChangePercent)},
		{"metric": "换手率", "left": fmt.Sprintf("%.2f%%", a.Turnover), "right": fmt.Sprintf("%.2f%%", b.Turnover), "winner": pctWinner(a.Turnover, b.Turnover)},
		{"metric": "量比", "left": fmt.Sprintf("%.2f", a.VolumeRatio), "right": fmt.Sprintf("%.2f", b.VolumeRatio), "winner": pctWinner(a.VolumeRatio, b.VolumeRatio)},
		{"metric": "行业", "left": emptyDash(a.Industry), "right": emptyDash(b.Industry)},
	}
	if a.MainNetInflow != 0 || b.MainNetInflow != 0 {
		rows = append(rows, map[string]interface{}{
			"metric": "主力净流入",
			"left":   fmtFundYi(a.MainNetInflow),
			"right":  fmtFundYi(b.MainNetInflow),
			"winner": pctWinner(a.MainNetInflow, b.MainNetInflow),
		})
	}
	if sides[0].ind != nil || sides[1].ind != nil {
		rows = append(rows,
			map[string]interface{}{"metric": "MA20", "left": fmtMA(sides[0].ind), "right": fmtMA(sides[1].ind)},
			map[string]interface{}{"metric": "RSI6", "left": fmtRSI(sides[0].ind), "right": fmtRSI(sides[1].ind)},
			map[string]interface{}{"metric": "MACD", "left": fmtMACD(sides[0].ind), "right": fmtMACD(sides[1].ind)},
		)
	}

	riskNotes := []string{}
	if a.ChangePercent > 5 || b.ChangePercent > 5 {
		riskNotes = append(riskNotes, "日内涨幅较大，注意追高回撤风险")
	}
	if a.Turnover > 10 || b.Turnover > 10 {
		riskNotes = append(riskNotes, "换手偏高，波动可能放大")
	}
	if a.VolumeRatio > 2.5 || b.VolumeRatio > 2.5 {
		riskNotes = append(riskNotes, "量比明显抬升，短线情绪偏热")
	}

	summary := explainCompare(a, b, sides[0].ind, sides[1].ind)
	blocks := []Block{
		{
			Type:  "comparison",
			Title: fmt.Sprintf("%s vs %s", a.Name, b.Name),
			Text:  summary,
			Items: rows,
			Meta: map[string]interface{}{
				"left":  map[string]interface{}{"symbol": a.Symbol, "name": a.Name, "price": a.Price, "changePercent": a.ChangePercent},
				"right": map[string]interface{}{"symbol": b.Symbol, "name": b.Name, "price": b.Price, "changePercent": b.ChangePercent},
			},
		},
	}
	if len(riskNotes) > 0 {
		blocks = append(blocks, Block{
			Type:   "risk",
			Title:  "对比风险提示",
			Text:   strings.Join(riskNotes, "；"),
			Symbol: a.Symbol,
			Items:  riskNotes,
			Meta:   map[string]interface{}{"level": "notable", "scope": "compare"},
		})
	}
	blocks = append(blocks, Block{
		Type:  "suggestions",
		Title: "继续研究",
		Items: []string{fmt.Sprintf("分析%s", a.Name), fmt.Sprintf("分析%s", b.Name), "茅台和比亚迪对比", "我的自选"},
	})

	return &ChatResponse{
		Reply:  summary,
		Intent: "compare",
		Blocks: blocks,
		Workspace: &WorkspaceHint{
			Type:          "compare",
			Symbol:        a.Symbol,
			Name:          a.Name,
			Tab:           "overview",
			CompareSymbol: b.Symbol,
			CompareName:   b.Name,
		},
	}, nil
}

func splitCompareNames(msg string) []string {
	clean := msg
	for _, cut := range []string{"对比", "比较", "比一比", "比一下", "相比", "帮我", "请", "一下"} {
		clean = strings.ReplaceAll(clean, cut, " ")
	}
	clean = strings.ReplaceAll(clean, "vs", " ")
	clean = strings.ReplaceAll(clean, "VS", " ")
	for _, sep := range []string{"和", "跟", "与", "/", "|", "、", ","} {
		clean = strings.ReplaceAll(clean, sep, "|")
	}
	parts := strings.Split(clean, "|")
	out := make([]string, 0, 2)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "的了吗呢比")
		if p == "" || len([]rune(p)) > 12 {
			continue
		}
		out = append(out, p)
	}
	return out
}

func pctWinner(left, right float64) string {
	if left > right {
		return "left"
	}
	if right > left {
		return "right"
	}
	return ""
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func fmtFundYi(v float64) string {
	yi := v / 1e8
	if yi >= 0 {
		return fmt.Sprintf("+%.2f亿", yi)
	}
	return fmt.Sprintf("%.2f亿", yi)
}

func fmtMA(p *indicator.Point) string {
	if p == nil || p.MA20 == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", *p.MA20)
}

func fmtRSI(p *indicator.Point) string {
	if p == nil || p.RSI6 == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f", *p.RSI6)
}

func fmtMACD(p *indicator.Point) string {
	if p == nil || p.Hist == nil {
		return "—"
	}
	h := *p.Hist
	if h > 0.01 {
		return "↑"
	}
	if h < -0.01 {
		return "↓"
	}
	return "→"
}

func explainCompare(a, b *provider.Quote, ia, ib *indicator.Point) string {
	stronger := a.Name
	if b.ChangePercent > a.ChangePercent {
		stronger = b.Name
	}
	bits := []string{
		fmt.Sprintf("%s（%s）现价 %.2f（%+.2f%%），%s（%s）现价 %.2f（%+.2f%%）。",
			a.Name, a.Symbol, a.Price, a.ChangePercent, b.Name, b.Symbol, b.Price, b.ChangePercent),
		fmt.Sprintf("今日涨跌幅看，%s 相对更强。", stronger),
	}
	if a.Industry != "" && b.Industry != "" && a.Industry != b.Industry {
		bits = append(bits, fmt.Sprintf("分属 %s / %s，行业不同，对比时更宜看相对强弱而非绝对价格。", a.Industry, b.Industry))
	}
	if ia != nil && ib != nil && ia.RSI6 != nil && ib.RSI6 != nil {
		if *ia.RSI6 > 70 || *ib.RSI6 > 70 {
			bits = append(bits, "RSI 已偏高，短线注意过热回撤。")
		}
	}
	bits = append(bits, "数字只反映当下强弱，决策仍需结合仓位与计划。")
	return strings.Join(bits, "")
}

func buildRiskBlock(symbol string, analysis interface{}) *Block {
	m, ok := asMap(analysis)
	if !ok {
		return nil
	}
	final, _ := asMap(m["final"])
	level := strings.TrimSpace(asString(final["risk"]))
	summary := strings.TrimSpace(asString(final["summary"]))
	action := strings.TrimSpace(asString(final["action"]))
	score := asFloat(final["score"])
	points := make([]string, 0, 4)
	if cards, ok := m["cards"].([]interface{}); ok {
		for _, c := range cards {
			cm, ok := asMap(c)
			if !ok {
				continue
			}
			ct := strings.ToLower(asString(cm["cardType"]))
			if ct == "risk" || strings.Contains(asString(cm["title"]), "风险") {
				title := asString(cm["title"])
				body := asString(cm["body"])
				if body == "" {
					body = asString(cm["content"])
				}
				if body == "" {
					body = asString(cm["text"])
				}
				line := strings.TrimSpace(title + "：" + body)
				if line != "：" && line != "" {
					points = append(points, strings.Trim(line, "："))
				}
			}
		}
	}
	if level == "" && summary == "" && len(points) == 0 {
		return nil
	}
	if summary == "" && len(points) > 0 {
		summary = points[0]
	}
	if summary == "" {
		summary = "请关注波动与仓位风险。"
	}
	text := summary
	if level != "" {
		text = fmt.Sprintf("风险等级 %s。%s", riskLabel(level), summary)
	}
	return &Block{
		Type:   "risk",
		Title:  "风险提示",
		Text:   text,
		Symbol: symbol,
		Items:  points,
		Meta: map[string]interface{}{
			"level":  normalizeRisk(level),
			"score":  score,
			"action": action,
			"raw":    level,
		},
	}
}

func riskLabel(level string) string {
	switch normalizeRisk(level) {
	case "low":
		return "偏低"
	case "high":
		return "偏高"
	default:
		return "中等"
	}
}

func normalizeRisk(level string) string {
	l := strings.ToLower(strings.TrimSpace(level))
	switch {
	case l == "low" || strings.Contains(l, "低"):
		return "low"
	case l == "high" || strings.Contains(l, "高"):
		return "high"
	default:
		return "medium"
	}
}

func asMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}
