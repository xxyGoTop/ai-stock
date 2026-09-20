package companion

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/watchlist"
)

type TodayOpsScan struct {
	AsOf    string           `json:"asOf"`
	Date    string           `json:"date"`
	Count   int              `json:"count"`
	Items   []watchlist.Item `json:"items"`
	Summary string           `json:"summary"`
}

func (s *Service) ScanTodayOps() (*TodayOpsScan, error) {
	now := time.Now()
	asOf := now.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04")
	day := watchlist.TodayCST(now)
	out := &TodayOpsScan{AsOf: asOf, Date: day, Items: []watchlist.Item{}, Summary: "今日暂无计划操作"}
	if s.watch == nil {
		return out, nil
	}
	items, err := s.watch.DueTodayOps(now)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if q, e := s.bundle.Quote(items[i].Symbol); e == nil && q != nil {
			items[i].Name = q.Name
			items[i].Price = q.Price
			items[i].Change = q.Change
			items[i].ChangePercent = q.ChangePercent
			items[i].Industry = q.Industry
			items[i].Market = string(q.Market)
		}
	}
	out.Items = items
	out.Count = len(items)
	if out.Count == 0 {
		out.Summary = "今日暂无「明日计划」到期操作。可先把股票加入明日计划。"
	} else {
		out.Summary = fmt.Sprintf("今日有 %d 只自选进入操作日，请按计划执行（站稳买入 / 止损 / 分批）。", out.Count)
	}
	return out, nil
}

func (s *Service) showWatchlist() (*ChatResponse, error) {
	items, err := s.watch.All()
	if err != nil {
		return &ChatResponse{Reply: "自选读取失败：" + err.Error(), Intent: "watchlist"}, nil
	}
	for i := range items {
		if q, e := s.bundle.Quote(items[i].Symbol); e == nil && q != nil {
			items[i].Name = q.Name
			items[i].Price = q.Price
			items[i].ChangePercent = q.ChangePercent
			items[i].Industry = q.Industry
		}
	}
	if len(items) == 0 {
		return &ChatResponse{
			Reply:  "自选还是空的。分析个股后可「加入自选」或「加入明日计划」。",
			Intent: "watchlist",
			Blocks: []Block{{
				Type:  "suggestions",
				Title: "试试",
				Items: []string{"分析贵州茅台", "帮我选股", "看看自选异动"},
			}},
		}, nil
	}
	def, plan := splitWatchCategories(items)
	blocks := []Block{{
		Type:  "watchlist",
		Title: "我的自选",
		Items: items,
		Meta: map[string]interface{}{
			"pageSize":       8,
			"defaultCount":   len(def),
			"tomorrowCount":  len(plan),
			"categories":     []string{watchlist.CategoryDefault, watchlist.CategoryTomorrowPlan},
		},
	}}
	reply := fmt.Sprintf("自选共 %d 只：普通 %d · 明日计划 %d。多的话可分页查看。", len(items), len(def), len(plan))
	return &ChatResponse{Reply: reply, Intent: "watchlist", Blocks: blocks}, nil
}

func (s *Service) showTomorrowPlans() (*ChatResponse, error) {
	items, err := s.watch.ByCategory(watchlist.CategoryTomorrowPlan)
	if err != nil {
		return &ChatResponse{Reply: "读取明日计划失败：" + err.Error(), Intent: "tomorrow_plan"}, nil
	}
	for i := range items {
		if q, e := s.bundle.Quote(items[i].Symbol); e == nil && q != nil {
			items[i].Name = q.Name
			items[i].Price = q.Price
			items[i].ChangePercent = q.ChangePercent
		}
	}
	if len(items) == 0 {
		return &ChatResponse{
			Reply:  "明日计划还是空的。分析后点「加入明日计划」，或说「把茅台加入明日计划」。",
			Intent: "tomorrow_plan",
			Blocks: []Block{{Type: "suggestions", Title: "可以", Items: []string{"分析贵州茅台", "我的自选", "今日操作"}}},
		}, nil
	}
	return &ChatResponse{
		Reply:  fmt.Sprintf("明日计划共 %d 只，按计划日与买卖节奏整理如下。", len(items)),
		Intent: "tomorrow_plan",
		Blocks: []Block{{
			Type:  "tomorrow_plan",
			Title: "明日计划",
			Text:  "站稳区间买入 · 止损 · 目标 · 分批次数",
			Items: items,
			Meta:  map[string]interface{}{"pageSize": 6},
		}},
	}, nil
}

func (s *Service) showTodayOps() (*ChatResponse, error) {
	scan, err := s.ScanTodayOps()
	if err != nil {
		return &ChatResponse{Reply: "今日操作读取失败：" + err.Error(), Intent: "today_ops"}, nil
	}
	if scan.Count == 0 {
		return &ChatResponse{
			Reply:  scan.Summary,
			Intent: "today_ops",
			Blocks: []Block{{Type: "suggestions", Title: "可以", Items: []string{"明日计划", "我的自选", "看看自选异动"}}},
		}, nil
	}
	return &ChatResponse{
		Reply:  scan.Summary,
		Intent: "today_ops",
		Blocks: []Block{{
			Type:  "today_ops",
			Title: "今日操作",
			Text:  scan.Summary,
			Items: scan.Items,
			Meta:  map[string]interface{}{"asOf": scan.AsOf, "date": scan.Date, "pageSize": 6, "count": scan.Count},
		}},
		Workspace: &WorkspaceHint{Type: "stock", Symbol: scan.Items[0].Symbol, Name: scan.Items[0].Name, Tab: "overview"},
	}, nil
}

func (s *Service) addTomorrowPlan(symbol, msg string) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		symbol = extractSymbol(msg)
	}
	if symbol == "" {
		return &ChatResponse{Reply: "请先指定股票，例如「把茅台加入明日计划」。", Intent: "tomorrow_plan"}, nil
	}
	name := ""
	var quote *provider.Quote
	if q, err := s.bundle.Quote(symbol); err == nil && q != nil {
		quote = q
		name = q.Name
		symbol = q.Symbol
	}
	plan := &watchlist.TradePlan{BuyBatches: 2, Action: "观察", Note: "待补充买卖节奏"}
	// 尽量用每日笔记里的交易计划填充
	if s.py != nil {
		if raw, err := s.py.DailyNote(symbol, false); err == nil && len(raw) > 0 {
			var note map[string]interface{}
			if json.Unmarshal(raw, &note) == nil {
				fillPlanFromNote(plan, note)
				if n, ok := note["name"].(string); ok && n != "" {
					name = n
				}
			}
		}
	}
	if quote != nil && plan.BuyLow == 0 && plan.BuyHigh == 0 {
		p := quote.Price
		plan.BuyLow = round2(p * 0.985)
		plan.BuyHigh = round2(p * 1.005)
		plan.Stop = round2(p * 0.97)
		plan.Target1 = round2(p * 1.03)
		plan.Target2 = round2(p * 1.06)
		plan.BuyBatches = 3
		plan.Action = "分批试探"
		plan.Note = "未取到笔记计划，已按现价给出参考区间，请自行核对。"
	}
	item, err := s.watch.Upsert(watchlist.UpsertInput{
		Symbol:   symbol,
		Name:     name,
		Market:   string(provider.GuessMarket(symbol)),
		Category: watchlist.CategoryTomorrowPlan,
		Plan:     plan,
	})
	if err != nil {
		return &ChatResponse{Reply: "加入明日计划失败：" + err.Error(), Intent: "tomorrow_plan"}, nil
	}
	return &ChatResponse{
		Reply: fmt.Sprintf("已将 %s 加入「明日计划」（生效 %s）。到那天会推送今日操作：站稳买入 / 止损 / 分批。",
			item.Name, item.PlanForDate),
		Intent: "tomorrow_plan",
		Blocks: []Block{{
			Type:  "tomorrow_plan",
			Title: "已加入明日计划",
			Items: []watchlist.Item{*item},
			Meta:  map[string]interface{}{"pageSize": 6},
		}},
		Workspace: &WorkspaceHint{Type: "stock", Symbol: item.Symbol, Name: item.Name, Tab: "overview"},
	}, nil
}

func fillPlanFromNote(plan *watchlist.TradePlan, note map[string]interface{}) {
	if action, ok := note["action"].(string); ok && action != "" {
		plan.Action = action
	}
	if an, ok := note["actionNote"].(string); ok && an != "" {
		plan.Note = an
	}
	raw, ok := note["plan"].(map[string]interface{})
	if !ok {
		return
	}
	plan.BuyLow = asFloat(raw["buyLow"])
	plan.BuyHigh = asFloat(raw["buyHigh"])
	plan.Stop = asFloat(raw["stop"])
	plan.Target1 = asFloat(raw["target1"])
	plan.Target2 = asFloat(raw["target2"])
	plan.Position = asFloat(raw["position"])
	if v, ok := raw["stance"].(string); ok {
		plan.Stance = v
	}
	if v, ok := raw["entryType"].(string); ok {
		plan.EntryType = v
	}
	if v, ok := raw["note"].(string); ok && v != "" {
		plan.Note = v
	}
	// 按仓位粗分批：仓位越大批次数略多
	plan.BuyBatches = 2
	if plan.Position >= 0.15 {
		plan.BuyBatches = 3
	}
	if plan.Position >= 0.25 {
		plan.BuyBatches = 4
	}
	if strings.Contains(plan.Action, "买") || plan.BuyLow > 0 {
		if plan.Action == "" || plan.Action == "观察" {
			plan.Action = "分批买入"
		}
	}
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func splitWatchCategories(items []watchlist.Item) (def, plan []watchlist.Item) {
	for _, it := range items {
		if watchlist.NormalizeCategory(it.Category) == watchlist.CategoryTomorrowPlan {
			plan = append(plan, it)
		} else {
			def = append(def, it)
		}
	}
	return
}

func formatPlanLine(it watchlist.Item) string {
	if it.Plan == nil {
		return "计划待补充"
	}
	p := it.Plan
	parts := []string{}
	if p.Action != "" {
		parts = append(parts, p.Action)
	}
	if p.BuyLow > 0 || p.BuyHigh > 0 {
		parts = append(parts, fmt.Sprintf("站稳 %.2f-%.2f 买", p.BuyLow, p.BuyHigh))
	}
	if p.Stop > 0 {
		parts = append(parts, fmt.Sprintf("止损 %.2f", p.Stop))
	}
	if p.Target1 > 0 {
		tg := fmt.Sprintf("目标 %.2f", p.Target1)
		if p.Target2 > 0 {
			tg += fmt.Sprintf("/%.2f", p.Target2)
		}
		parts = append(parts, tg)
	}
	if p.BuyBatches > 0 {
		parts = append(parts, fmt.Sprintf("分 %d 次买", p.BuyBatches))
	}
	if len(parts) == 0 {
		return "计划待补充"
	}
	return strings.Join(parts, " · ")
}
