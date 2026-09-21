package companion

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/dailypicks"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

var (
	reSymbol = regexp.MustCompile(`\b([036]\d{5})\b`)
	reCode   = regexp.MustCompile(`(\d{6})`)
)

func (s *Service) Chat(req ChatRequest) (*ChatResponse, error) {
	msg := strings.TrimSpace(req.Message)
	action := strings.TrimSpace(req.Action)
	if action == "" {
		action = detectIntent(msg, req.Symbol)
	}
	symbol := provider.PadSymbol(req.Symbol)
	if symbol == "000000" || symbol == "" {
		symbol = extractSymbol(msg)
	}

	switch action {
	case "briefing", "market":
		b, err := s.BuildBriefing()
		if err != nil {
			return nil, err
		}
		return &ChatResponse{
			Reply:  b.MarketSummary,
			Intent: "briefing",
			Blocks: b.Blocks,
			Workspace: &WorkspaceHint{Type: "market", Tab: "overview"},
		}, nil
	case "hot":
		return s.hotFeed()
	case "screening":
		return s.runScreening()
	case "recommend", "preopen", "intraday", "close_auction", "review":
		return s.recommend(action, msg)
	case "analyze":
		return s.analyzeStock(symbol, msg, req)
	case "watch":
		return s.addWatch(symbol, msg)
	case "unwatch":
		return s.removeWatch(symbol, msg)
	case "watch_clear":
		return s.clearWatchlist()
	case "paper":
		return s.paperHint(symbol, msg)
	case "kline":
		return s.klineHint(symbol, msg)
	case "watchlist":
		return s.showWatchlist()
	case "watch_anomaly", "anomaly":
		return s.showAnomalies()
	case "tomorrow_plan":
		// 有标的则加入，否则展示列表
		if symbol != "" && symbol != "000000" {
			return s.addTomorrowPlan(symbol, msg)
		}
		if extractSymbol(msg) != "" || strings.Contains(msg, "加入") || strings.Contains(msg, "添加") {
			return s.addTomorrowPlan(symbol, msg)
		}
		return s.showTomorrowPlans()
	case "today_ops", "today_plan":
		return s.showTodayOps()
	default:
		if symbol != "" && symbol != "000000" {
			return s.analyzeStock(symbol, msg, req)
		}
		if looksLikeHot(msg) {
			return s.hotFeed()
		}
		if looksLikeScreening(msg) {
			return s.runScreening()
		}
		if looksLikeMarket(msg) {
			return s.Chat(ChatRequest{Message: msg, Action: "briefing"})
		}
		if looksLikeRecommend(msg) {
			return s.recommend("recommend", msg)
		}
		return s.modelChat(req)
	}
}

func (s *Service) modelChat(req ChatRequest) (*ChatResponse, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"message":   strings.TrimSpace(req.Message),
		"messages":  req.Messages,
		"modelCode": strings.TrimSpace(req.ModelCode),
	})
	raw, err := s.py.Chat(payload)
	if err != nil {
		return &ChatResponse{
			Reply:  "对话模型暂时不可用：" + err.Error() + "。也可以直接说「今天行情」「帮我选股」或「分析茅台」。",
			Intent: "chat",
			Blocks: []Block{{Type: "suggestions", Items: []string{"今天行情", "帮我选股", "分析茅台"}}},
		}, nil
	}
	var data struct {
		Reply     string `json:"reply"`
		ModelCode string `json:"modelCode"`
	}
	if err := json.Unmarshal(raw, &data); err != nil || strings.TrimSpace(data.Reply) == "" {
		return &ChatResponse{
			Reply:  "模型没有返回内容，换一句再问，或说「分析茅台」。",
			Intent: "chat",
		}, nil
	}
	return &ChatResponse{
		Reply:  strings.TrimSpace(data.Reply),
		Intent: "chat",
		Blocks: []Block{
			{Type: "suggestions", Title: "也可以", Items: []string{"今天行情", "今日热点", "帮我选股", "分析茅台"}},
		},
		Workspace: &WorkspaceHint{Type: "empty"},
	}, nil
}

func detectIntent(msg, symbol string) string {
	sym := symbol
	if sym == "" {
		sym = extractSymbol(msg)
	}
	switch {
	case strings.Contains(msg, "自选") && (strings.Contains(msg, "清空") || strings.Contains(msg, "一键清空")):
		return "watch_clear"
	case strings.Contains(msg, "自选") && (strings.Contains(msg, "删除") || strings.Contains(msg, "移除") || strings.Contains(msg, "移出") || strings.Contains(msg, "取消自选")):
		return "unwatch"
	case strings.Contains(msg, "自选") && (strings.Contains(msg, "加入") || strings.Contains(msg, "添加") || strings.Contains(msg, "加自选")):
		return "watch"
	case strings.Contains(msg, "模拟") || strings.Contains(msg, "下单") || strings.Contains(msg, "买入") || strings.Contains(msg, "卖出"):
		return "paper"
	case strings.Contains(msg, "K线") || strings.Contains(msg, "k线") || strings.Contains(strings.ToLower(msg), "kline") || strings.Contains(msg, "分时"):
		return "kline"
	case strings.Contains(msg, "盘前"):
		return "preopen"
	case strings.Contains(msg, "盘中"):
		return "intraday"
	case strings.Contains(msg, "尾盘"):
		return "close_auction"
	case strings.Contains(msg, "复盘") || (strings.Contains(msg, "收盘") && !strings.Contains(msg, "分析")):
		return "review"
	case looksLikeScreening(msg):
		return "screening"
	case looksLikeHot(msg):
		return "hot"
	case strings.Contains(msg, "推荐"):
		return "recommend"
	case strings.Contains(msg, "今日操作") || strings.Contains(msg, "今天操作") || strings.Contains(msg, "操作计划"):
		return "today_ops"
	case strings.Contains(msg, "明日计划") || (strings.Contains(msg, "明天") && strings.Contains(msg, "计划")):
		if strings.Contains(msg, "加入") || strings.Contains(msg, "添加") || extractSymbol(msg) != "" {
			return "tomorrow_plan"
		}
		return "tomorrow_plan"
	case strings.Contains(msg, "异动") || strings.Contains(msg, "自选提醒") || strings.Contains(msg, "盯盘"):
		return "watch_anomaly"
	case strings.Contains(msg, "我的自选") || msg == "自选股" || strings.Contains(msg, "自选列表") || strings.Contains(msg, "看看自选"):
		return "watchlist"
	case looksLikeMarket(msg) && sym == "":
		return "briefing"
	case sym != "" || strings.Contains(msg, "分析") || strings.Contains(msg, "研究"):
		if sym != "" || strings.Contains(msg, "分析") || strings.Contains(msg, "研究") || strings.Contains(msg, "看看") {
			return "analyze"
		}
	}
	return "help"
}

func looksLikeMarket(msg string) bool {
	keys := []string{"市场", "行情", "大盘", "指数", "板块", "北向", "今天怎么", "今日行情", "今日市场"}
	for _, k := range keys {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

func looksLikeHot(msg string) bool {
	keys := []string{"热点", "快讯", "新闻", "题材", "热门板块"}
	for _, k := range keys {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

func looksLikeScreening(msg string) bool {
	keys := []string{"选股", "扫描", "五算法", "筛选股票", "帮我选"}
	for _, k := range keys {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

func looksLikeRecommend(msg string) bool {
	keys := []string{"推荐", "买什么", "观察名单"}
	for _, k := range keys {
		if strings.Contains(msg, k) {
			return true
		}
	}
	return false
}

func (s *Service) hotFeed() (*ChatResponse, error) {
	feed, err := s.bundle.HotFeed(12)
	if err != nil {
		return nil, err
	}
	reply := "这是当前热点：热门板块、主题与重要快讯。点领涨股可继续分析。"
	if len(feed.Topics) > 0 {
		reply = fmt.Sprintf("当前最热主题偏「%s」。下面是板块与快讯。", feed.Topics[0].Topic)
	}
	blocks := []Block{
		{Type: "text", Text: reply},
		{Type: "boards", Title: "热门板块", Items: feed.Boards},
		{Type: "topics", Title: "热点主题", Items: feed.Topics},
		{Type: "news", Title: "重要快讯", Items: feed.News},
		{Type: "suggestions", Title: "继续", Items: []string{"帮我选股", "今天行情", "盘中推荐"}},
	}
	return &ChatResponse{
		Reply:     reply,
		Intent:    "hot",
		Blocks:    blocks,
		Workspace: &WorkspaceHint{Type: "market", Tab: "overview"},
	}, nil
}

func (s *Service) runScreening() (*ChatResponse, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"detail": 120,
		"limit":  30,
	})
	raw, err := s.py.Screening(payload)
	if err != nil {
		return &ChatResponse{
			Reply:  "选股服务暂不可用：" + err.Error() + "。可先看「今日热点」或「盘中推荐」。",
			Intent: "screening",
			Blocks: []Block{{Type: "suggestions", Items: []string{"今日热点", "盘中推荐", "今天行情"}}},
		}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	scanned, _ := result["scanned"].(float64)
	qualified, _ := result["qualified"].(float64)
	picks, _ := result["picks"].([]interface{})
	if len(picks) > 30 {
		picks = picks[:30]
		result["picks"] = picks
	}
	reply := fmt.Sprintf("五算法选股完成：扫描 %.0f 只，列出前 %d 只达标标的（共达标 %.0f）。已写入今日选股记录，可在设置中查看。", scanned, len(picks), qualified)
	if len(picks) == 0 {
		reply = "本轮扫描没有达标标的，可以稍后再扫，或先看热点板块领涨股。"
	} else {
		s.saveScreeningRecord(picks, reply)
	}
	blocks := []Block{
		{Type: "text", Text: reply},
		{Type: "screen_picks", Title: "选股结果", Data: result, Items: picks, Meta: map[string]interface{}{"pageSize": 10}},
		{Type: "suggestions", Items: []string{"今日热点", "今天行情", "我的自选"}},
	}
	ws := &WorkspaceHint{Type: "market", Tab: "overview"}
	if len(picks) > 0 {
		if m, ok := picks[0].(map[string]interface{}); ok {
			sym, _ := m["symbol"].(string)
			name, _ := m["name"].(string)
			if sym != "" {
				ws = &WorkspaceHint{Type: "stock", Symbol: provider.PadSymbol(sym), Name: name, Tab: "overview"}
			}
		}
	}
	return &ChatResponse{Reply: reply, Intent: "screening", Blocks: blocks, Workspace: ws}, nil
}

func (s *Service) saveScreeningRecord(picks []interface{}, summary string) {
	if s.picks == nil {
		return
	}
	items := make([]dailypicks.Pick, 0, len(picks))
	for _, it := range picks {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, dailypicks.Pick{
			Symbol:        provider.PadSymbol(asString(m["symbol"])),
			Name:          asString(m["name"]),
			ChangePercent: asFloat(m["changePercent"]),
			Score:         asFloat(m["score"]),
			PrimaryName:   asString(m["primaryName"]),
			Industry:      asString(m["industry"]),
			Reason:        asString(m["primaryName"]),
		})
	}
	_ = s.picks.Save(dailypicks.Record{
		Kind:    "screening",
		Title:   "今日选股",
		Summary: summary,
		Picks:   items,
	})
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func extractSymbol(msg string) string {
	if m := reSymbol.FindStringSubmatch(msg); len(m) > 1 {
		return provider.PadSymbol(m[1])
	}
	if m := reCode.FindStringSubmatch(msg); len(m) > 1 {
		return provider.PadSymbol(m[1])
	}
	// 常见简称
	aliases := map[string]string{
		"茅台": "600519", "贵州茅台": "600519", "宁德": "300750", "宁德时代": "300750",
		"比亚迪": "002594", "平安": "601318", "招商银行": "600036", "招行": "600036",
		"中芯": "688981", "寒武纪": "688256", "海康": "002415",
	}
	for name, code := range aliases {
		if strings.Contains(msg, name) {
			return code
		}
	}
	return ""
}

func (s *Service) recommend(action, _ string) (*ChatResponse, error) {
	boards, err := s.bundle.HotBoards(40)
	if err != nil {
		b, berr := s.BuildBriefing()
		if berr != nil {
			return nil, err
		}
		boards = b.Boards
	}
	picks := picksFromBoards(boards, 30)
	phase := DetectPhase(time.Now())
	title := "今日推荐"
	kind := "recommend"
	switch action {
	case "preopen":
		phase, title, kind = PhasePreOpen, "盘前推荐", "preopen"
	case "intraday":
		phase, title, kind = PhaseIntraday, "盘中推荐", "intraday"
	case "close_auction":
		phase, title, kind = PhaseCloseAuc, "尾盘推荐", "close_auction"
	case "review":
		phase, title, kind = PhaseReview, "收盘复盘观察", "review"
	}
	_ = phase
	summary := fmt.Sprintf("%s：共 %d 只，来自热门板块领涨股。同日再次生成会覆盖记录。", title, len(picks))
	if len(picks) == 0 {
		summary = title + "暂时没有足够的板块领涨样本，你可以先看热门板块，或说「帮我选股」。"
	} else if s.picks != nil {
		items := make([]dailypicks.Pick, 0, len(picks))
		for _, p := range picks {
			items = append(items, dailypicks.Pick{
				Symbol: p.Symbol, Name: p.Name, ChangePercent: p.ChangePercent,
				Reason: p.Reason, Board: p.Board,
			})
		}
		_ = s.picks.Save(dailypicks.Record{
			Kind: kind, Title: title, Summary: summary, Picks: items,
		})
		// 同时覆盖一份「今日推荐」总表，方便设置页默认查看
		_ = s.picks.Save(dailypicks.Record{
			Kind: "recommend", Title: "今日推荐", Summary: summary, Picks: items,
		})
	}
	blocks := []Block{
		{Type: "text", Text: summary},
		{Type: "picks", Title: title, Items: picks, Meta: map[string]interface{}{"pageSize": 10}},
		{Type: "boards", Title: "对应热门板块", Items: boards},
	}
	ws := &WorkspaceHint{Type: "market", Tab: "overview"}
	if len(picks) > 0 {
		ws = &WorkspaceHint{Type: "stock", Symbol: picks[0].Symbol, Name: picks[0].Name, Tab: "overview"}
	}
	return &ChatResponse{Reply: summary, Intent: "recommend", Blocks: blocks, Workspace: ws}, nil
}

func analyzePayload(symbol string, req ChatRequest) []byte {
	body := map[string]string{"symbol": symbol}
	if strings.TrimSpace(req.ProfileCode) != "" {
		body["profileCode"] = strings.TrimSpace(req.ProfileCode)
	}
	if strings.TrimSpace(req.ModelCode) != "" {
		body["modelCode"] = strings.TrimSpace(req.ModelCode)
	}
	raw, _ := json.Marshal(body)
	return raw
}

func (s *Service) analyzeStock(symbol, msg string, req ChatRequest) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		// 尝试搜索中文名
		q := strings.TrimSpace(msg)
		for _, cut := range []string{"分析", "看看", "研究", "帮我"} {
			q = strings.ReplaceAll(q, cut, "")
		}
		q = strings.TrimSpace(q)
		if q != "" {
			items, err := s.bundle.Search(q)
			if err == nil && len(items) > 0 {
				symbol = items[0].Symbol
			}
		}
	}
	if symbol == "" || symbol == "000000" {
		return &ChatResponse{
			Reply:  "告诉我股票代码或名称，例如「分析 600519」或「帮我看看贵州茅台」。",
			Intent: "analyze",
			Blocks: []Block{{Type: "suggestions", Items: []string{"分析 600519", "看看宁德时代", "分析比亚迪"}}},
		}, nil
	}

	quote, err := s.bundle.Quote(symbol)
	if err != nil || quote == nil {
		return &ChatResponse{Reply: "没拿到 " + symbol + " 的行情，换只股票试试。", Intent: "analyze"}, nil
	}

	blocks := []Block{
		{
			Type:    "stock_card",
			Title:   "个股快照",
			Quote:   quote,
			Symbol:  quote.Symbol,
			Actions: []string{"analyze", "watch", "paper", "kline"},
			Text:    fmt.Sprintf("%s（%s）现价 %.2f，%+.2f%%。", quote.Name, quote.Symbol, quote.Price, quote.ChangePercent),
		},
	}

	// 每日笔记（规则）
	if raw, err := s.py.DailyNote(symbol, false); err == nil && len(raw) > 0 {
		var note interface{}
		if json.Unmarshal(raw, &note) == nil {
			blocks = append(blocks, Block{Type: "daily_note", Title: "今日笔记", Data: note, Symbol: symbol})
		}
	}

	// LLM 分析（无 key 时会走 quant-rules）
	if raw, err := s.py.Analyze(analyzePayload(symbol, req)); err == nil && len(raw) > 0 {
		var analysis interface{}
		if json.Unmarshal(raw, &analysis) == nil {
			blocks = append(blocks, Block{Type: "analysis", Title: "AI 解读", Data: analysis, Symbol: symbol})
		}
	}

	if feed, nerr := s.bundle.StockNews(symbol, 5); nerr == nil && feed != nil {
		if len(feed.News) > 0 {
			blocks = append(blocks, Block{Type: "news", Title: "相关新闻", Items: feed.News, Symbol: symbol})
		}
		if len(feed.Notices) > 0 {
			blocks = append(blocks, Block{Type: "news", Title: "近期公告", Items: feed.Notices, Symbol: symbol})
		}
	}

	blocks = append(blocks, Block{
		Type:    "actions",
		Title:   "接下来可以",
		Symbol:  symbol,
		Actions: []string{"watch", "tomorrow_plan", "paper", "kline"},
		Items:   []string{"加入自选", "加入明日计划", "加入模拟交易", "打开今日K线"},
	})

	reply := fmt.Sprintf("已整理 %s 的快照、新闻与分析。可加入自选或明日计划，右侧可看 K 线和公告。", quote.Name)
	return &ChatResponse{
		Reply:  reply,
		Intent: "analyze",
		Blocks: blocks,
		Workspace: &WorkspaceHint{Type: "stock", Symbol: quote.Symbol, Name: quote.Name, Tab: "analysis"},
	}, nil
}

func (s *Service) addWatch(symbol, msg string) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		symbol = extractSymbol(msg)
	}
	if symbol == "" {
		return &ChatResponse{Reply: "请先指定股票，例如「把茅台加入自选」。", Intent: "watch"}, nil
	}
	name := ""
	if q, err := s.bundle.Quote(symbol); err == nil && q != nil {
		name = q.Name
		symbol = q.Symbol
	}
	item, err := s.watch.Add(symbol, name, string(provider.GuessMarket(symbol)))
	if err != nil {
		return &ChatResponse{Reply: "加入自选失败：" + err.Error(), Intent: "watch"}, nil
	}
	return &ChatResponse{
		Reply:  fmt.Sprintf("已将 %s（%s）加入自选。", item.Name, item.Symbol),
		Intent: "watch",
		Blocks: []Block{{
			Type: "stock_card", Symbol: item.Symbol, Title: "已加入自选",
			Text: fmt.Sprintf("%s %s", item.Name, item.Symbol),
			Actions: []string{"analyze", "paper", "kline"},
		}},
		Workspace: &WorkspaceHint{Type: "stock", Symbol: item.Symbol, Name: item.Name, Tab: "overview"},
	}, nil
}

func (s *Service) removeWatch(symbol, msg string) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		symbol = extractSymbol(msg)
	}
	if symbol == "" {
		return &ChatResponse{Reply: "请先指定要删除的股票，例如「删除自选茅台」。", Intent: "unwatch"}, nil
	}
	name := symbol
	if q, err := s.bundle.Quote(symbol); err == nil && q != nil {
		name = q.Name
		symbol = q.Symbol
	}
	if err := s.watch.Remove(symbol); err != nil {
		return &ChatResponse{Reply: "删除自选失败：" + err.Error(), Intent: "unwatch"}, nil
	}
	return &ChatResponse{
		Reply:  fmt.Sprintf("已将 %s（%s）移出自选。", name, symbol),
		Intent: "unwatch",
		Blocks: []Block{{
			Type:  "suggestions",
			Title: "接下来",
			Items: []string{"我的自选", "分析" + name, "加入自选"},
		}},
	}, nil
}

func (s *Service) clearWatchlist() (*ChatResponse, error) {
	items, err := s.watch.All()
	if err != nil {
		return &ChatResponse{Reply: "读取自选失败：" + err.Error(), Intent: "watch_clear"}, nil
	}
	n := len(items)
	if err := s.watch.Clear(); err != nil {
		return &ChatResponse{Reply: "清空自选失败：" + err.Error(), Intent: "watch_clear"}, nil
	}
	return &ChatResponse{
		Reply:  fmt.Sprintf("已清空自选，共移除 %d 只。", n),
		Intent: "watch_clear",
		Blocks: []Block{{
			Type:  "suggestions",
			Title: "可以",
			Items: []string{"分析贵州茅台", "帮我选股", "看看自选异动"},
		}},
	}, nil
}

func (s *Service) paperHint(symbol, msg string) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		symbol = extractSymbol(msg)
	}
	if symbol == "" {
		return &ChatResponse{
			Reply:  "模拟交易需要指定股票。可以说「模拟买入茅台」，或在分析后点「加入模拟交易」。",
			Intent: "paper",
		}, nil
	}
	quote, err := s.bundle.Quote(symbol)
	if err != nil || quote == nil {
		return &ChatResponse{Reply: "行情获取失败，暂无法准备模拟单。", Intent: "paper"}, nil
	}
	return &ChatResponse{
		Reply:  fmt.Sprintf("已为 %s 打开模拟交易面板。现价 %.2f，请在右侧确认买入/卖出（模拟盘，非实盘）。", quote.Name, quote.Price),
		Intent: "paper",
		Blocks: []Block{{
			Type: "stock_card", Quote: quote, Symbol: quote.Symbol, Title: "模拟交易",
			Actions: []string{"paper", "analyze", "watch", "kline"},
			Text:    "需你确认后才会记入模拟账户（非实盘）。",
		}},
		Workspace: &WorkspaceHint{Type: "stock", Symbol: quote.Symbol, Name: quote.Name, Tab: "paper"},
	}, nil
}

func (s *Service) klineHint(symbol, msg string) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		symbol = extractSymbol(msg)
	}
	if symbol == "" {
		return &ChatResponse{Reply: "告诉我要看哪只股票的 K 线，例如「打开茅台K线」。", Intent: "kline"}, nil
	}
	quote, _ := s.bundle.Quote(symbol)
	name := symbol
	if quote != nil {
		name = quote.Name
		symbol = quote.Symbol
	}
	return &ChatResponse{
		Reply:  fmt.Sprintf("已打开 %s 的日 K 线，可在右侧研究工作台查看。", name),
		Intent: "kline",
		Blocks: []Block{{
			Type: "kline", Title: "日 K", Symbol: symbol,
			Quote: quote, Actions: []string{"analyze", "watch", "paper"},
		}},
		Workspace: &WorkspaceHint{Type: "stock", Symbol: symbol, Name: name, Tab: "kline"},
	}, nil
}
