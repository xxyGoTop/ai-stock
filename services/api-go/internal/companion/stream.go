package companion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

// EmitFunc 推送一条 SSE 事件。event 对齐 V2 协议。
type EmitFunc func(event string, data interface{})

type PlanStep struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // pending | running | done | error
}

type ToolEvent struct {
	Tool    string      `json:"tool"`
	Title   string      `json:"title,omitempty"`
	OK      bool        `json:"ok,omitempty"`
	Summary string      `json:"summary,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ChatStream 以 Agent Run 模式流式推送研究过程。
func (s *Service) ChatStream(ctx context.Context, req ChatRequest, emit EmitFunc) error {
	msg := strings.TrimSpace(req.Message)
	action := strings.TrimSpace(req.Action)
	if action == "" {
		action = detectIntent(msg, req.Symbol)
	}
	symbol := provider.PadSymbol(req.Symbol)
	if symbol == "000000" || symbol == "" {
		symbol = extractSymbol(msg)
	}

	// 归一化 default 分支
	if action == "help" || action == "" {
		if symbol != "" && symbol != "000000" {
			action = "analyze"
		} else if looksLikeHot(msg) {
			action = "hot"
		} else if looksLikeScreening(msg) {
			action = "screening"
		} else if looksLikeMarket(msg) {
			action = "briefing"
		} else if looksLikeRecommend(msg) {
			action = "recommend"
		} else if strings.Contains(msg, "异动") || strings.Contains(msg, "盯盘") {
			action = "watch_anomaly"
		} else {
			action = "help"
		}
	}

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	emit("message.start", map[string]interface{}{
		"runId":  runID,
		"intent": action,
		"mode":   "agent_run",
	})

	plan := planForAction(action)
	emit("research.plan", map[string]interface{}{"steps": plan})

	if err := ctx.Err(); err != nil {
		emit("error", map[string]string{"message": "已取消"})
		return err
	}

	var (
		res *ChatResponse
		err error
	)

	switch action {
	case "briefing", "market":
		res, err = s.streamBriefing(ctx, emit, plan)
	case "hot":
		res, err = s.streamHot(ctx, emit, plan)
	case "screening":
		res, err = s.streamScreening(ctx, emit, plan)
	case "recommend", "preopen", "intraday", "close_auction", "review":
		res, err = s.streamRecommend(ctx, emit, plan, action)
	case "analyze":
		res, err = s.streamAnalyze(ctx, emit, plan, symbol, msg, req)
	case "watch":
		res, err = s.addWatch(symbol, msg)
		s.emitStaticRun(emit, plan, res, err)
	case "unwatch":
		res, err = s.removeWatch(symbol, msg)
		s.emitStaticRun(emit, plan, res, err)
	case "watch_clear":
		res, err = s.clearWatchlist()
		s.emitStaticRun(emit, plan, res, err)
	case "paper":
		res, err = s.paperHint(symbol, msg)
		s.emitStaticRun(emit, plan, res, err)
	case "kline":
		res, err = s.klineHint(symbol, msg)
		s.emitStaticRun(emit, plan, res, err)
	case "watchlist":
		res, err = s.showWatchlist()
		s.emitStaticRun(emit, plan, res, err)
	case "tomorrow_plan":
		if symbol != "" && symbol != "000000" {
			res, err = s.addTomorrowPlan(symbol, msg)
		} else if extractSymbol(msg) != "" || strings.Contains(msg, "加入") {
			res, err = s.addTomorrowPlan(symbol, msg)
		} else {
			res, err = s.showTomorrowPlans()
		}
		s.emitStaticRun(emit, plan, res, err)
	case "today_ops", "today_plan":
		res, err = s.showTodayOps()
		s.emitStaticRun(emit, plan, res, err)
	case "watch_anomaly", "anomaly":
		res, err = s.streamAnomalies(ctx, emit, plan)
	default:
		res, err = s.streamChat(ctx, emit, plan, req)
	}

	if err != nil {
		emit("error", map[string]string{"message": err.Error()})
		emit("message.end", map[string]interface{}{"ok": false, "runId": runID})
		return err
	}
	if res == nil {
		emit("error", map[string]string{"message": "empty response"})
		emit("message.end", map[string]interface{}{"ok": false, "runId": runID})
		return fmt.Errorf("empty response")
	}

	// 文本增量（按句粗切，形成流式感）
	emitDelta(emit, res.Reply)

	emit("message.end", map[string]interface{}{
		"ok":        true,
		"runId":     runID,
		"reply":     res.Reply,
		"intent":    res.Intent,
		"blocks":    res.Blocks,
		"workspace": res.Workspace,
	})
	return nil
}

func planForAction(action string) []PlanStep {
	switch action {
	case "briefing", "market":
		return []PlanStep{
			{ID: "index", Title: "获取指数情况", Status: "pending"},
			{ID: "nb", Title: "北向资金", Status: "pending"},
			{ID: "boards", Title: "主流板块 / 热点", Status: "pending"},
			{ID: "summary", Title: "生成今日摘要", Status: "pending"},
		}
	case "hot":
		return []PlanStep{
			{ID: "boards", Title: "拉取热门板块", Status: "pending"},
			{ID: "news", Title: "整理快讯主题", Status: "pending"},
			{ID: "render", Title: "生成热点摘要", Status: "pending"},
		}
	case "screening":
		return []PlanStep{
			{ID: "plan", Title: "制定选股方案", Status: "pending"},
			{ID: "scan", Title: "扫描候选池", Status: "pending"},
			{ID: "score", Title: "五算法评分", Status: "pending"},
			{ID: "save", Title: "写入今日记录", Status: "pending"},
		}
	case "recommend", "preopen", "intraday", "close_auction", "review":
		return []PlanStep{
			{ID: "boards", Title: "扫描主流板块", Status: "pending"},
			{ID: "leaders", Title: "提取领涨股", Status: "pending"},
			{ID: "save", Title: "写入今日记录", Status: "pending"},
		}
	case "analyze":
		return []PlanStep{
			{ID: "quote", Title: "获取实时行情", Status: "pending"},
			{ID: "note", Title: "生成每日笔记", Status: "pending"},
			{ID: "news", Title: "个股新闻与公告", Status: "pending"},
			{ID: "ai", Title: "AI 解读整理", Status: "pending"},
			{ID: "actions", Title: "准备后续动作", Status: "pending"},
		}
	case "watch_anomaly", "anomaly":
		return []PlanStep{
			{ID: "load", Title: "读取自选列表", Status: "pending"},
			{ID: "scan", Title: "扫描涨跌与量能", Status: "pending"},
			{ID: "judge", Title: "生成异动结论", Status: "pending"},
		}
	default:
		return []PlanStep{
			{ID: "think", Title: "理解问题", Status: "pending"},
			{ID: "llm", Title: "调用对话模型", Status: "pending"},
			{ID: "render", Title: "整理回复", Status: "pending"},
		}
	}
}

func markPlan(emit EmitFunc, plan []PlanStep, id, status string) []PlanStep {
	out := make([]PlanStep, len(plan))
	copy(out, plan)
	for i := range out {
		if out[i].ID == id {
			out[i].Status = status
		} else if status == "running" && out[i].Status == "running" {
			out[i].Status = "done"
		}
	}
	emit("research.plan", map[string]interface{}{"steps": out})
	return out
}

func toolStart(emit EmitFunc, tool, title string) {
	emit("tool.start", ToolEvent{Tool: tool, Title: title})
}

func toolResult(emit EmitFunc, tool, title string, ok bool, summary string, data interface{}, errMsg string) {
	emit("tool.result", ToolEvent{
		Tool: tool, Title: title, OK: ok, Summary: summary, Data: data, Error: errMsg,
	})
}

func emitBlock(emit EmitFunc, b Block) {
	emit("block", b)
}

func emitDelta(emit EmitFunc, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	// 粗粒度分片，便于前端流式显示
	parts := splitReply(text)
	for _, p := range parts {
		emit("message.delta", map[string]string{"content": p})
	}
}

func splitReply(text string) []string {
	seps := []rune{'。', '！', '？', '\n', '；'}
	var out []string
	start := 0
	runes := []rune(text)
	for i, r := range runes {
		for _, sep := range seps {
			if r == sep {
				chunk := strings.TrimSpace(string(runes[start : i+1]))
				if chunk != "" {
					out = append(out, chunk)
				}
				start = i + 1
				break
			}
		}
	}
	if start < len(runes) {
		rest := strings.TrimSpace(string(runes[start:]))
		if rest != "" {
			out = append(out, rest)
		}
	}
	if len(out) == 0 {
		return []string{text}
	}
	return out
}

func (s *Service) streamChat(ctx context.Context, emit EmitFunc, plan []PlanStep, req ChatRequest) (*ChatResponse, error) {
	plan = markPlan(emit, plan, "think", "running")
	toolStart(emit, "understand", "理解问题")
	toolResult(emit, "understand", "理解问题", true, "自由问答", nil, "")
	if aborted(ctx) {
		return nil, ctx.Err()
	}
	plan = markPlan(emit, plan, "llm", "running")
	toolStart(emit, "llm_chat", "调用对话模型")
	res, err := s.modelChat(req)
	if err != nil {
		toolResult(emit, "llm_chat", "调用对话模型", false, "", nil, err.Error())
		return res, err
	}
	toolResult(emit, "llm_chat", "调用对话模型", true, "已生成回复", nil, "")
	plan = markPlan(emit, plan, "render", "done")
	if res != nil {
		for _, b := range res.Blocks {
			emitBlock(emit, b)
		}
	}
	return res, nil
}

func (s *Service) emitStaticRun(emit EmitFunc, plan []PlanStep, res *ChatResponse, err error) {
	plan = markPlan(emit, plan, plan[0].ID, "running")
	toolStart(emit, "handler", "执行操作")
	if err != nil {
		toolResult(emit, "handler", "执行操作", false, "", nil, err.Error())
		plan = markPlan(emit, plan, plan[0].ID, "error")
		return
	}
	toolResult(emit, "handler", "执行操作", true, res.Reply, nil, "")
	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})
	if res != nil {
		for _, b := range res.Blocks {
			emitBlock(emit, b)
		}
	}
}

func aborted(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (s *Service) streamBriefing(ctx context.Context, emit EmitFunc, plan []PlanStep) (*ChatResponse, error) {
	plan = markPlan(emit, plan, "index", "running")
	toolStart(emit, "get_indices", "获取指数")
	indices, err := s.bundle.IndexQuotes()
	if err != nil {
		toolResult(emit, "get_indices", "获取指数", false, "", nil, err.Error())
	} else {
		toolResult(emit, "get_indices", "获取指数", true, fmt.Sprintf("%d 个指数", len(indices)), indices, "")
	}
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "nb", "running")
	toolStart(emit, "get_northbound", "北向资金")
	nb, nbErr := s.bundle.NorthboundFlow()
	if nbErr != nil {
		toolResult(emit, "get_northbound", "北向资金", false, "", nil, nbErr.Error())
	} else {
		toolResult(emit, "get_northbound", "北向资金", true, nb.Text, nb, "")
	}
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "boards", "running")
	toolStart(emit, "get_hot", "热点板块")
	feed, feedErr := s.bundle.HotFeed(10)
	if feedErr != nil {
		toolResult(emit, "get_hot", "热点板块", false, "", nil, feedErr.Error())
	} else {
		toolResult(emit, "get_hot", "热点板块", true, fmt.Sprintf("%d 个板块", len(feed.Boards)), nil, "")
	}

	plan = markPlan(emit, plan, "summary", "running")
	b, err := s.BuildBriefing()
	if err != nil {
		plan = markPlan(emit, plan, "summary", "error")
		return nil, err
	}
	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})
	for _, block := range b.Blocks {
		emitBlock(emit, block)
	}
	return &ChatResponse{
		Reply: b.MarketSummary, Intent: "briefing", Blocks: b.Blocks,
		Workspace: &WorkspaceHint{Type: "market", Tab: "overview"},
	}, nil
}

func (s *Service) streamHot(ctx context.Context, emit EmitFunc, plan []PlanStep) (*ChatResponse, error) {
	plan = markPlan(emit, plan, "boards", "running")
	toolStart(emit, "get_hot_boards", "热门板块")
	feed, err := s.bundle.HotFeed(12)
	if err != nil {
		toolResult(emit, "get_hot_boards", "热门板块", false, "", nil, err.Error())
		return nil, err
	}
	toolResult(emit, "get_hot_boards", "热门板块", true, fmt.Sprintf("%d 板块", len(feed.Boards)), nil, "")
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "news", "running")
	toolStart(emit, "get_hot_news", "重要快讯")
	toolResult(emit, "get_hot_news", "重要快讯", true, fmt.Sprintf("%d 条", len(feed.News)), nil, "")

	plan = markPlan(emit, plan, "render", "running")
	res, err := s.hotFeed()
	if err != nil {
		return nil, err
	}
	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})
	for _, b := range res.Blocks {
		emitBlock(emit, b)
	}
	return res, nil
}

func (s *Service) streamRecommend(ctx context.Context, emit EmitFunc, plan []PlanStep, action string) (*ChatResponse, error) {
	plan = markPlan(emit, plan, "boards", "running")
	toolStart(emit, "get_hot_boards", "扫描板块")
	boards, err := s.bundle.HotBoards(40)
	if err != nil {
		toolResult(emit, "get_hot_boards", "扫描板块", false, "", nil, err.Error())
		return s.recommend(action, "")
	}
	toolResult(emit, "get_hot_boards", "扫描板块", true, fmt.Sprintf("%d 个板块", len(boards)), nil, "")
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "leaders", "running")
	toolStart(emit, "extract_leaders", "提取领涨股")
	picks := picksFromBoards(boards, 30)
	toolResult(emit, "extract_leaders", "提取领涨股", true, fmt.Sprintf("%d 只", len(picks)), nil, "")

	plan = markPlan(emit, plan, "save", "running")
	toolStart(emit, "save_daily_picks", "写入今日记录")
	res, err := s.recommend(action, "")
	if err != nil {
		toolResult(emit, "save_daily_picks", "写入今日记录", false, "", nil, err.Error())
		return nil, err
	}
	toolResult(emit, "save_daily_picks", "写入今日记录", true, "已覆盖同日记录", nil, "")
	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})
	for _, b := range res.Blocks {
		emitBlock(emit, b)
	}
	return res, nil
}

func (s *Service) streamScreening(ctx context.Context, emit EmitFunc, plan []PlanStep) (*ChatResponse, error) {
	plan = markPlan(emit, plan, "plan", "running")
	toolStart(emit, "screen_plan", "制定选股方案")
	toolResult(emit, "screen_plan", "制定选股方案", true, "五算法 · 前 30 只", nil, "")
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "scan", "running")
	toolStart(emit, "screen_scan", "扫描候选池")
	plan = markPlan(emit, plan, "score", "running")
	toolStart(emit, "screen_score", "五算法评分")

	res, err := s.runScreening()
	if err != nil {
		toolResult(emit, "screen_scan", "扫描候选池", false, "", nil, err.Error())
		toolResult(emit, "screen_score", "五算法评分", false, "", nil, err.Error())
		return nil, err
	}
	toolResult(emit, "screen_scan", "扫描候选池", true, "扫描完成", nil, "")
	toolResult(emit, "screen_score", "五算法评分", true, res.Reply, nil, "")
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "save", "running")
	toolStart(emit, "save_daily_picks", "写入今日记录")
	toolResult(emit, "save_daily_picks", "写入今日记录", true, "已覆盖同日选股记录", nil, "")

	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})
	for _, b := range res.Blocks {
		emitBlock(emit, b)
	}
	return res, nil
}

func (s *Service) streamAnalyze(ctx context.Context, emit EmitFunc, plan []PlanStep, symbol, msg string, req ChatRequest) (*ChatResponse, error) {
	if symbol == "" || symbol == "000000" {
		q := strings.TrimSpace(msg)
		for _, cut := range []string{"分析", "看看", "研究", "帮我"} {
			q = strings.ReplaceAll(q, cut, "")
		}
		q = strings.TrimSpace(q)
		if q != "" {
			plan = markPlan(emit, plan, "quote", "running")
			toolStart(emit, "search_stock", "解析股票")
			items, err := s.bundle.Search(q)
			if err == nil && len(items) > 0 {
				symbol = items[0].Symbol
				toolResult(emit, "search_stock", "解析股票", true, items[0].Name+" "+symbol, items[0], "")
			} else {
				toolResult(emit, "search_stock", "解析股票", false, "", nil, "未找到")
			}
		}
	}
	if symbol == "" || symbol == "000000" {
		res := &ChatResponse{
			Reply:  "告诉我股票代码或名称，例如「分析 600519」或「帮我看看贵州茅台」。",
			Intent: "analyze",
			Blocks: []Block{{Type: "suggestions", Items: []string{"分析 600519", "看看宁德时代", "分析比亚迪"}}},
		}
		for _, b := range res.Blocks {
			emitBlock(emit, b)
		}
		return res, nil
	}
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "quote", "running")
	toolStart(emit, "get_quote", "获取实时行情")
	quote, err := s.bundle.Quote(symbol)
	if err != nil || quote == nil {
		msgErr := "行情为空"
		if err != nil {
			msgErr = err.Error()
		}
		toolResult(emit, "get_quote", "获取实时行情", false, "", nil, msgErr)
		return &ChatResponse{Reply: "没拿到 " + symbol + " 的行情，换只股票试试。", Intent: "analyze"}, nil
	}
	toolResult(emit, "get_quote", "获取实时行情", true,
		fmt.Sprintf("%s %.2f (%+.2f%%)", quote.Name, quote.Price, quote.ChangePercent), quote, "")

	stockBlock := Block{
		Type: "stock_card", Title: "个股快照", Quote: quote, Symbol: quote.Symbol,
		Actions: []string{"analyze", "watch", "paper", "kline"},
		Text:    fmt.Sprintf("%s（%s）现价 %.2f，%+.2f%%。", quote.Name, quote.Symbol, quote.Price, quote.ChangePercent),
	}
	emitBlock(emit, stockBlock)
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	blocks := []Block{stockBlock}

	plan = markPlan(emit, plan, "note", "running")
	toolStart(emit, "daily_note", "生成每日笔记")
	if raw, err := s.py.DailyNote(symbol, false); err == nil && len(raw) > 0 {
		var note interface{}
		if json.Unmarshal(raw, &note) == nil {
			b := Block{Type: "daily_note", Title: "今日笔记", Data: note, Symbol: symbol}
			blocks = append(blocks, b)
			emitBlock(emit, b)
			toolResult(emit, "daily_note", "生成每日笔记", true, "笔记已就绪", nil, "")
		} else {
			toolResult(emit, "daily_note", "生成每日笔记", false, "", nil, "解析失败")
		}
	} else {
		errMsg := "不可用"
		if err != nil {
			errMsg = err.Error()
		}
		toolResult(emit, "daily_note", "生成每日笔记", false, "", nil, errMsg)
	}
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "news", "running")
	toolStart(emit, "stock_news", "个股新闻与公告")
	if feed, err := s.bundle.StockNews(symbol, 5); err == nil && feed != nil {
		if len(feed.News) > 0 {
			b := Block{Type: "news", Title: "相关新闻", Items: feed.News, Symbol: symbol}
			blocks = append(blocks, b)
			emitBlock(emit, b)
		}
		if len(feed.Notices) > 0 {
			b := Block{Type: "news", Title: "近期公告", Items: feed.Notices, Symbol: symbol}
			blocks = append(blocks, b)
			emitBlock(emit, b)
		}
		toolResult(emit, "stock_news", "个股新闻与公告", true, fmt.Sprintf("新闻 %d · 公告 %d", len(feed.News), len(feed.Notices)), nil, "")
	} else {
		errMsg := "暂无"
		if err != nil {
			errMsg = err.Error()
		}
		toolResult(emit, "stock_news", "个股新闻与公告", false, "", nil, errMsg)
	}
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "ai", "running")
	toolStart(emit, "ai_analyze", "AI 解读")
	if raw, err := s.py.Analyze(analyzePayload(symbol, req)); err == nil && len(raw) > 0 {
		var analysis interface{}
		if json.Unmarshal(raw, &analysis) == nil {
			b := Block{Type: "analysis", Title: "AI 解读", Data: analysis, Symbol: symbol}
			blocks = append(blocks, b)
			emitBlock(emit, b)
			toolResult(emit, "ai_analyze", "AI 解读", true, "解读完成", nil, "")
		} else {
			toolResult(emit, "ai_analyze", "AI 解读", false, "", nil, "解析失败")
		}
	} else {
		errMsg := "不可用"
		if err != nil {
			errMsg = err.Error()
		}
		toolResult(emit, "ai_analyze", "AI 解读", false, "", nil, errMsg)
	}

	plan = markPlan(emit, plan, "actions", "running")
	actionBlock := Block{
		Type: "actions", Title: "接下来可以", Symbol: symbol,
		Actions: []string{"watch", "tomorrow_plan", "paper", "kline"},
		Items:   []string{"加入自选", "加入明日计划", "加入模拟交易", "打开今日K线"},
	}
	blocks = append(blocks, actionBlock)
	emitBlock(emit, actionBlock)

	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})

	reply := fmt.Sprintf("已整理 %s 的快照、新闻与分析。右侧可看 K 线和公告，也可一键自选/模拟。", quote.Name)
	return &ChatResponse{
		Reply: reply, Intent: "analyze", Blocks: blocks,
		Workspace: &WorkspaceHint{Type: "stock", Symbol: quote.Symbol, Name: quote.Name, Tab: "analysis"},
	}, nil
}

func (s *Service) streamAnomalies(ctx context.Context, emit EmitFunc, plan []PlanStep) (*ChatResponse, error) {
	if aborted(ctx) {
		return nil, ctx.Err()
	}
	plan = markPlan(emit, plan, "load", "running")
	toolStart(emit, "watchlist", "读取自选")
	n := 0
	if s.watch != nil {
		if items, err := s.watch.All(); err == nil {
			n = len(items)
			toolResult(emit, "watchlist", "读取自选", true, fmt.Sprintf("%d 只", n), nil, "")
		} else {
			toolResult(emit, "watchlist", "读取自选", false, "", nil, err.Error())
			return nil, err
		}
	} else {
		toolResult(emit, "watchlist", "读取自选", true, "0 只", nil, "")
	}
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "scan", "running")
	toolStart(emit, "scan_anomaly", "扫描涨跌与量能")
	scan, err := s.ScanWatchAnomalies()
	if err != nil {
		toolResult(emit, "scan_anomaly", "扫描涨跌与量能", false, "", nil, err.Error())
		return nil, err
	}
	toolResult(emit, "scan_anomaly", "扫描涨跌与量能", true, scan.Summary, scan, "")
	if aborted(ctx) {
		return nil, ctx.Err()
	}

	plan = markPlan(emit, plan, "judge", "running")
	res, err := s.showAnomalies()
	if err != nil {
		return nil, err
	}
	for _, b := range res.Blocks {
		emitBlock(emit, b)
	}
	for i := range plan {
		plan[i].Status = "done"
	}
	emit("research.plan", map[string]interface{}{"steps": plan})
	return res, nil
}
