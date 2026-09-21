package companion

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/watchlist"
)

// Anomaly 自选股异动结论（推研究摘要，不堆原始 tick）。
type Anomaly struct {
	Symbol        string   `json:"symbol"`
	Name          string   `json:"name"`
	Price         float64  `json:"price"`
	ChangePercent float64  `json:"changePercent"`
	VolumeRatio   float64  `json:"volumeRatio,omitempty"`
	Turnover      float64  `json:"turnover,omitempty"`
	Industry         string   `json:"industry,omitempty"`
	Level            string   `json:"level"` // mild | notable | strong
	Reasons          []string `json:"reasons"`
	Summary          string   `json:"summary"`
	Fingerprint      string   `json:"fingerprint"`
	MainNetInflow    float64  `json:"mainNetInflow,omitempty"`
	MainNetInflowPct float64  `json:"mainNetInflowPct,omitempty"`
	FundText         string   `json:"fundText,omitempty"`
	Lift             string   `json:"lift,omitempty"`
	LiftText         string   `json:"liftText,omitempty"`
}

type AnomalyScan struct {
	AsOf     string    `json:"asOf"`
	Count    int       `json:"count"`
	Items    []Anomaly `json:"items"`
	Summary  string    `json:"summary"`
	HasWatch bool      `json:"hasWatch"`
}

// ScanWatchAnomalies 扫描自选异动。
func (s *Service) ScanWatchAnomalies() (*AnomalyScan, error) {
	rules := s.GetAnomalyRules()
	asOf := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04")
	out := &AnomalyScan{AsOf: asOf, Items: []Anomaly{}, Summary: "暂无自选", HasWatch: false}
	if s.watch == nil {
		return out, nil
	}
	items, err := s.watch.All()
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		out.Summary = "自选为空，加入几只股票后我会盯盘异动。"
		return out, nil
	}
	out.HasWatch = true

	type row struct {
		item watchlist.Item
		q    *provider.Quote
	}
	rows := make([]row, len(items))
	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rows[i].item = items[i]
			if q, e := s.bundle.Quote(items[i].Symbol); e == nil && q != nil {
				rows[i].q = q
			}
		}(i)
	}
	wg.Wait()

	anomalies := make([]Anomaly, 0)
	for _, r := range rows {
		if r.q == nil {
			continue
		}
		a, ok := judgeAnomaly(r.item, r.q, asOf, rules)
		if ok {
			anomalies = append(anomalies, a)
		}
	}
	sort.Slice(anomalies, func(i, j int) bool {
		return levelRank(anomalies[i].Level) > levelRank(anomalies[j].Level) ||
			(levelRank(anomalies[i].Level) == levelRank(anomalies[j].Level) &&
				math.Abs(anomalies[i].ChangePercent) > math.Abs(anomalies[j].ChangePercent))
	})
	if len(anomalies) > 12 {
		anomalies = anomalies[:12]
	}
	out.Items = anomalies
	out.Count = len(anomalies)
	if out.Count == 0 {
		out.Summary = fmt.Sprintf("已扫过 %d 只自选，暂无明显异动。", len(items))
	} else {
		out.Summary = fmt.Sprintf("你的自选股中有 %d 只出现较明显异动。", out.Count)
	}
	return out, nil
}

func judgeAnomaly(it watchlist.Item, q *provider.Quote, asOf string, rules AnomalyRules) (Anomaly, bool) {
	name := q.Name
	if name == "" {
		name = it.Name
	}
	reasons := make([]string, 0, 6)
	level := ""

	absChg := math.Abs(q.ChangePercent)
	if absChg >= rules.ChangeStrong {
		dir := "大涨"
		if q.ChangePercent < 0 {
			dir = "大跌"
		}
		reasons = append(reasons, fmt.Sprintf("今日%s %.2f%%", dir, q.ChangePercent))
		level = bumpLevel(level, "strong")
	} else if absChg >= rules.ChangeMild {
		dir := "上涨"
		if q.ChangePercent < 0 {
			dir = "下跌"
		}
		reasons = append(reasons, fmt.Sprintf("涨跌幅异动：%s %.2f%%", dir, q.ChangePercent))
		level = bumpLevel(level, "notable")
	}

	if q.VolumeRatio >= rules.VolumeRatioStrong {
		reasons = append(reasons, fmt.Sprintf("量比放大至 %.2f", q.VolumeRatio))
		level = bumpLevel(level, "strong")
	} else if q.VolumeRatio >= rules.VolumeRatioMild {
		reasons = append(reasons, fmt.Sprintf("量比 %.2f，成交活跃", q.VolumeRatio))
		level = bumpLevel(level, "mild")
	}

	if q.Turnover >= rules.TurnoverMild {
		reasons = append(reasons, fmt.Sprintf("换手率 %.2f%%", q.Turnover))
		level = bumpLevel(level, "mild")
	}

	fundIn, fundOut, fundText, fundLv := fundMove(q, rules)
	if fundText != "" {
		reasons = append(reasons, fundText)
		level = bumpLevel(level, fundLv)
	}
	if math.Abs(q.SuperNetInflow) >= rules.SuperMildYi*1e8 {
		if q.SuperNetInflow > 0 {
			reasons = append(reasons, "超大单净买入 "+fmtFund(q.SuperNetInflow))
		} else {
			reasons = append(reasons, "超大单净卖出 "+fmtFund(q.SuperNetInflow))
		}
		level = bumpLevel(level, "notable")
	}

	liftKind, liftText, liftLv := liftState(q, rules)
	if liftKind == "lifting" {
		reasons = append(reasons, liftText)
		level = bumpLevel(level, liftLv)
		if fundIn {
			reasons = append(reasons, "资金推动拉升")
			level = bumpLevel(level, "notable")
		} else if fundOut {
			reasons = append(reasons, "拉升过程主力兑现")
			level = bumpLevel(level, "notable")
		}
	} else if fundText != "" {
		reasons = append(reasons, liftText)
	}

	if len(reasons) == 0 {
		return Anomaly{}, false
	}
	if level == "" {
		level = "mild"
	}
	summary := fmt.Sprintf("%s（%s）现价 %.2f，%+.2f%%。%s",
		name, q.Symbol, q.Price, q.ChangePercent, strings.Join(reasons, "；"))
	day := strings.Split(asOf, " ")[0]
	fp := fmt.Sprintf("%s|%s|%s", q.Symbol, day, strings.Join(reasons, ","))
	return Anomaly{
		Symbol:           q.Symbol,
		Name:             name,
		Price:            q.Price,
		ChangePercent:    q.ChangePercent,
		VolumeRatio:      q.VolumeRatio,
		Turnover:         q.Turnover,
		Industry:         q.Industry,
		Level:            level,
		Reasons:          reasons,
		Summary:          summary,
		Fingerprint:      fp,
		MainNetInflow:    q.MainNetInflow,
		MainNetInflowPct: q.MainNetInflowPct,
		FundText:         fundText,
		Lift:             liftKind,
		LiftText:         liftText,
	}, true
}

func fundMove(q *provider.Quote, rules AnomalyRules) (in, out bool, text, level string) {
	main := q.MainNetInflow
	pct := q.MainNetInflowPct
	absMain := math.Abs(main)
	absPct := math.Abs(pct)
	mild := rules.FundMildYi * 1e8
	strong := rules.FundStrongYi * 1e8
	if absMain < mild && absPct < rules.FundPctMild {
		return false, false, "", ""
	}
	if main > 0 {
		in = true
		text = "主力净买入 " + fmtFund(main)
		if pct != 0 {
			text += fmt.Sprintf("（占成交 %+.1f%%）", pct)
		}
		level = "notable"
		if absMain >= strong || absPct >= rules.FundPctStrong {
			text = "主力大幅买入 " + fmtFund(main)
			if pct != 0 {
				text += fmt.Sprintf("（占成交 %+.1f%%）", pct)
			}
			level = "strong"
		}
		return in, false, text, level
	}
	out = true
	text = "主力净卖出 " + fmtFund(main)
	if pct != 0 {
		text += fmt.Sprintf("（占成交 %+.1f%%）", pct)
	}
	level = "notable"
	if absMain >= strong || absPct >= rules.FundPctStrong {
		text = "主力大幅卖出 " + fmtFund(main)
		if pct != 0 {
			text += fmt.Sprintf("（占成交 %+.1f%%）", pct)
		}
		level = "strong"
	}
	return false, out, text, level
}

func liftState(q *provider.Quote, rules AnomalyRules) (kind, text, level string) {
	base := q.Open
	if base <= 0 {
		base = q.PrevClose
	}
	if base <= 0 {
		return "none", "未见明显拉升", ""
	}
	fromOpen := (q.Price - base) / base * 100
	nearHigh := q.High > 0 && (q.High-q.Price)/q.High <= 0.012 && q.Price >= q.Open
	if fromOpen >= rules.LiftStrong || (q.ChangePercent >= rules.ChangeStrong && fromOpen >= rules.LiftMild) {
		if q.VolumeRatio >= 1.5 {
			return "lifting", fmt.Sprintf("放量拉升，较开盘 %+.2f%%", fromOpen), "strong"
		}
		return "lifting", fmt.Sprintf("明显拉升，较开盘 %+.2f%%", fromOpen), "strong"
	}
	if fromOpen >= rules.LiftNotable && nearHigh {
		return "lifting", fmt.Sprintf("盘中拉升，现价贴近最高，较开盘 %+.2f%%", fromOpen), "notable"
	}
	if fromOpen >= rules.LiftMild && q.ChangePercent >= rules.LiftMild-0.3 {
		return "lifting", fmt.Sprintf("有拉升迹象，较开盘 %+.2f%%", fromOpen), "mild"
	}
	if fromOpen <= -1.5 {
		return "none", "未见拉升，现价低于开盘", ""
	}
	return "none", "未见明显拉升", ""
}

func fmtFund(n float64) string {
	yi := n / 1e8
	if math.Abs(yi) >= 0.01 {
		return fmt.Sprintf("%+.2f亿", yi)
	}
	return fmt.Sprintf("%+.0f万", n/1e4)
}

func bumpLevel(cur, next string) string {
	if levelRank(next) > levelRank(cur) {
		return next
	}
	if cur == "" {
		return next
	}
	return cur
}

func levelRank(l string) int {
	switch l {
	case "strong":
		return 3
	case "notable":
		return 2
	case "mild":
		return 1
	default:
		return 0
	}
}

func (s *Service) showAnomalies() (*ChatResponse, error) {
	scan, err := s.ScanWatchAnomalies()
	if err != nil {
		return &ChatResponse{Reply: "异动扫描失败：" + err.Error(), Intent: "watch_anomaly"}, nil
	}
	if !scan.HasWatch {
		return &ChatResponse{
			Reply:  scan.Summary,
			Intent: "watch_anomaly",
			Blocks: []Block{{
				Type:  "suggestions",
				Title: "可以先",
				Items: []string{"分析贵州茅台", "帮我选股", "今日热点"},
			}},
		}, nil
	}
	blocks := []Block{}
	if scan.Count == 0 {
		blocks = append(blocks, Block{Type: "text", Title: "自选异动", Text: scan.Summary})
		blocks = append(blocks, Block{Type: "suggestions", Title: "接下来", Items: []string{"我的自选", "今天行情", "帮我选股"}})
		return &ChatResponse{Reply: scan.Summary, Intent: "watch_anomaly", Blocks: blocks}, nil
	}
	blocks = append(blocks, Block{
		Type:  "anomaly",
		Title: "自选异动",
		Text:  scan.Summary,
		Items: scan.Items,
		Meta:  map[string]interface{}{"asOf": scan.AsOf, "count": scan.Count},
	})
	top := scan.Items[0]
	blocks = append(blocks, Block{
		Type:    "actions",
		Title:   "值得继续看",
		Symbol:  top.Symbol,
		Items:   []string{fmt.Sprintf("分析 %s", top.Name), "打开 K 线", "全部自选"},
		Actions: []string{"analyze", "kline", "watchlist"},
	})
	blocks = append(blocks, Block{
		Type:  "suggestions",
		Title: "你可以",
		Items: []string{fmt.Sprintf("分析%s", top.Name), "看看我的自选股", "今天行情"},
	})
	reply := scan.Summary + " 我先标出结论，点一只可继续研究。"
	return &ChatResponse{
		Reply:     reply,
		Intent:    "watch_anomaly",
		Blocks:    blocks,
		Workspace: &WorkspaceHint{Type: "stock", Symbol: top.Symbol, Name: top.Name, Tab: "overview"},
	}, nil
}
