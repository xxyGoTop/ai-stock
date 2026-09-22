package companion

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/dailypicks"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

var (
	reBoardInAction = regexp.MustCompile(`在([^\s，,。！？、/]{1,12}?)(?:板块|概念|行业)?(?:里|中|内)?(?:选股|推荐|扫描|筛选|找票|选几只|挑几只)`)
	reBoardThenAct  = regexp.MustCompile(`(?:帮我|请|给我)?([^\s，,。！？、/]{2,10}?)(?:板块|概念|行业)?(?:里|中|内)?(?:选股|推荐|扫描|筛选|找票)`)
)

type screeningNLResult struct {
	Summary     string   `json:"summary"`
	BoardHints  []string `json:"boardHints"`
	Algorithms  []string `json:"algorithms"`
	Limit       int      `json:"limit"`
	Detail      int      `json:"detail"`
	Filters     string   `json:"filters"`
	ModelCode   string   `json:"modelCode"`
}

func extractBoardHint(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	if m := reBoardInAction.FindStringSubmatch(msg); len(m) > 1 {
		return cleanBoardHint(m[1])
	}
	if m := reBoardThenAct.FindStringSubmatch(msg); len(m) > 1 {
		return cleanBoardHint(m[1])
	}
	return ""
}

func cleanBoardHint(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimSuffix(s, "板块")
	s = strings.TrimSuffix(s, "概念")
	s = strings.TrimSuffix(s, "行业")
	s = strings.TrimSpace(s)
	ban := []string{"帮我", "请", "给我", "今日", "今天", "盘中", "盘前", "尾盘", "全市场", "全局", "整体", "大盘", "市场"}
	for _, b := range ban {
		if s == b {
			return ""
		}
	}
	if len([]rune(s)) < 2 {
		return ""
	}
	return s
}

func (s *Service) resolveBoard(hint string) (*provider.HotBoard, error) {
	if hint == "" {
		return nil, fmt.Errorf("empty board")
	}
	return s.bundle.FindBoard(hint)
}

func (s *Service) loadCandidateBoards() []provider.HotBoard {
	out := make([]provider.HotBoard, 0, 200)
	seen := map[string]bool{}
	add := func(list []provider.HotBoard) {
		for _, b := range list {
			key := b.Code + "|" + b.Name
			if b.Name == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, b)
		}
	}
	if list, err := s.bundle.ListConceptAndIndustryBoards(120); err == nil {
		add(list)
	}
	if hot, err := s.bundle.HotBoards(50); err == nil {
		add(hot)
	}
	return out
}

func picksFromBoardStocks(boardName string, stocks []provider.BoardStock, limit int) []RecommendPick {
	if limit <= 0 {
		limit = 30
	}
	allowST := isRiskBoardName(boardName)
	out := make([]RecommendPick, 0, limit)
	for _, st := range stocks {
		if !allowST && strings.Contains(strings.ToUpper(st.Name), "ST") {
			continue
		}
		out = append(out, RecommendPick{
			Symbol:        st.Symbol,
			Name:          st.Name,
			ChangePercent: st.ChangePercent,
			Reason:        fmt.Sprintf("%s板块成分 · 涨跌 %+.2f%%", boardName, st.ChangePercent),
			Board:         boardName,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func isRiskBoardName(name string) bool {
	n := strings.ToUpper(strings.TrimSpace(name))
	return strings.Contains(n, "ST") || strings.Contains(name, "风险警示")
}

func (s *Service) askScreeningNL(msg string, boards []provider.HotBoard, modelCode string) (*screeningNLResult, error) {
	rows := make([]map[string]interface{}, 0, len(boards))
	for _, b := range boards {
		rows = append(rows, map[string]interface{}{
			"name": b.Name, "code": b.Code, "changePercent": b.ChangePercent,
		})
	}
	raw, err := s.py.ScreeningNL(map[string]interface{}{
		"query":     msg,
		"boards":    rows,
		"modelCode": modelCode,
	})
	if err != nil {
		return nil, err
	}
	var dest screeningNLResult
	if err := json.Unmarshal(raw, &dest); err != nil {
		return nil, err
	}
	return &dest, nil
}

func (s *Service) matchBoardsByHints(boards []provider.HotBoard, hints []string, fallbackHint string) []provider.HotBoard {
	out := make([]provider.HotBoard, 0)
	seen := map[string]bool{}
	addHit := func(b provider.HotBoard) {
		key := b.Code
		if key == "" {
			key = b.Name
		}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, b)
	}
	tryHints := append([]string{}, hints...)
	if fallbackHint != "" {
		tryHints = append(tryHints, fallbackHint)
	}
	for _, h := range tryHints {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		// exact / contains against candidate list
		for _, b := range boards {
			if b.Name == h || strings.Contains(b.Name, h) || strings.Contains(h, b.Name) {
				addHit(b)
			}
		}
		// provider FindBoard (aliases)
		if hit, err := s.bundle.FindBoard(h); err == nil && hit != nil {
			addHit(*hit)
		}
	}
	return out
}

func (s *Service) collectBoardStocks(boards []provider.HotBoard, perBoard int) ([]provider.BoardStock, string) {
	if perBoard <= 0 {
		perBoard = 60
	}
	seen := map[string]bool{}
	stocks := make([]provider.BoardStock, 0)
	names := make([]string, 0, len(boards))
	for _, b := range boards {
		names = append(names, b.Name)
		list, err := s.bundle.BoardStocks(b.Code, perBoard)
		if err != nil {
			continue
		}
		for _, st := range list {
			if seen[st.Symbol] {
				continue
			}
			seen[st.Symbol] = true
			stocks = append(stocks, st)
		}
	}
	return stocks, strings.Join(names, "、")
}

func (s *Service) recommendInBoard(action, msg, hint string) (*ChatResponse, error) {
	boards := s.loadCandidateBoards()
	nl, _ := s.askScreeningNL(msg, boards, "")
	matched := s.matchBoardsByHints(boards, nil, hint)
	if nl != nil && len(nl.BoardHints) > 0 {
		matched = s.matchBoardsByHints(boards, nl.BoardHints, hint)
	}
	if len(matched) == 0 {
		if b, err := s.resolveBoard(hint); err == nil && b != nil {
			matched = []provider.HotBoard{*b}
		}
	}
	if len(matched) == 0 {
		return &ChatResponse{
			Reply:  fmt.Sprintf("还是没定位到「%s」相关板块。可以换个说法，例如「在化学制药选股」「在中药推荐」。", hint),
			Intent: "recommend",
			Blocks: []Block{{
				Type:  "suggestions",
				Title: "试试",
				Items: []string{"在化学制药推荐", "在中药选股", "在创新药推荐", "今日热点"},
			}},
		}, nil
	}
	stocks, boardLabel := s.collectBoardStocks(matched, 40)
	if len(stocks) == 0 {
		return &ChatResponse{
			Reply:  fmt.Sprintf("已理解要看「%s」，但暂时拉不到成分股。请稍后再试。", boardLabel),
			Intent: "recommend",
			Blocks: []Block{{Type: "suggestions", Items: []string{"今日热点", "帮我选股"}}},
		}, nil
	}
	title := fmt.Sprintf("%s · 板块推荐", boardLabel)
	kind := "recommend"
	switch action {
	case "preopen":
		kind, title = "preopen", fmt.Sprintf("%s · 盘前推荐", boardLabel)
	case "intraday":
		kind, title = "intraday", fmt.Sprintf("%s · 盘中推荐", boardLabel)
	case "close_auction":
		kind, title = "close_auction", fmt.Sprintf("%s · 尾盘推荐", boardLabel)
	case "review":
		kind, title = "review", fmt.Sprintf("%s · 复盘观察", boardLabel)
	case "screening":
		kind, title = "screening", fmt.Sprintf("%s · 板块选股", boardLabel)
	}
	picks := picksFromBoardStocks(boardLabel, stocks, 30)
	understand := ""
	if nl != nil && nl.Summary != "" {
		understand = nl.Summary + " "
	}
	summary := fmt.Sprintf("%s%s：在「%s」相关成分中取出 %d 只（按涨跌幅）。", understand, title, boardLabel, len(picks))
	if s.picks != nil {
		items := make([]dailypicks.Pick, 0, len(picks))
		for _, p := range picks {
			items = append(items, dailypicks.Pick{
				Symbol: p.Symbol, Name: p.Name, ChangePercent: p.ChangePercent,
				Reason: p.Reason, Board: p.Board,
			})
		}
		_ = s.picks.Save(dailypicks.Record{Kind: kind, Title: title, Summary: summary, Picks: items})
	}
	blocks := []Block{
		{Type: "text", Text: summary},
		{Type: "picks", Title: title, Items: picks, Meta: map[string]interface{}{"pageSize": 10, "board": boardLabel}},
		{Type: "boards", Title: "目标板块", Items: matched},
		{Type: "suggestions", Title: "可以继续", Items: []string{
			fmt.Sprintf("在%s选股", hintOr(boardLabel, hint)),
			fmt.Sprintf("分析%s", picks[0].Name),
			"今日热点",
			"帮我选股",
		}},
	}
	return &ChatResponse{
		Reply:  summary,
		Intent: "recommend",
		Blocks: blocks,
		Workspace: &WorkspaceHint{
			Type: "sector", Symbol: matched[0].Code, Name: matched[0].Name, Tab: "overview",
			BoardCode: matched[0].Code, BoardName: matched[0].Name, Topic: hintOr(boardLabel, hint),
		},
	}, nil
}

func hintOr(a, b string) string {
	if a != "" {
		// take first board name if multi
		if i := strings.Index(a, "、"); i > 0 {
			return a[:i]
		}
		return a
	}
	return b
}

func (s *Service) screenInBoard(msg, hint string) (*ChatResponse, error) {
	boards := s.loadCandidateBoards()
	nl, nlErr := s.askScreeningNL(msg, boards, "")
	matched := s.matchBoardsByHints(boards, nil, hint)
	algos := []string{}
	limit := 30
	if nl != nil {
		if len(nl.BoardHints) > 0 {
			matched = s.matchBoardsByHints(boards, nl.BoardHints, hint)
		}
		algos = nl.Algorithms
		if nl.Limit > 0 {
			limit = nl.Limit
		}
	}
	if len(matched) == 0 {
		if b, err := s.resolveBoard(hint); err == nil && b != nil {
			matched = []provider.HotBoard{*b}
		}
	}
	if len(matched) == 0 {
		suggest := []string{"在化学制药选股", "在中药选股", "在创新药选股", "帮我选股"}
		reply := fmt.Sprintf("模型理解了要在「%s」方向选股，但候选列表里没对上具体板块。", hint)
		if nl != nil && nl.Summary != "" {
			reply = nl.Summary + " 不过暂时匹配不到东财板块名，换个更具体的板块名再试。"
		}
		if nlErr != nil {
			reply = fmt.Sprintf("「%s」板块未直接命中，且选股理解服务暂不可用：%s", hint, nlErr.Error())
		}
		return &ChatResponse{
			Reply:  reply,
			Intent: "screening",
			Blocks: []Block{{Type: "suggestions", Items: suggest}},
		}, nil
	}

	stocks, boardLabel := s.collectBoardStocks(matched, 80)
	if len(stocks) == 0 {
		return s.recommendInBoard("screening", msg, hint)
	}
	symbols := make([]string, 0, len(stocks))
	for _, st := range stocks {
		symbols = append(symbols, st.Symbol)
	}
	payload := map[string]interface{}{
		"detail":  minInt(len(symbols), 120),
		"limit":   limit,
		"symbols": symbols,
		"board":   boardLabel,
	}
	if len(algos) > 0 {
		payload["algorithms"] = algos
	}
	raw, err := s.py.Screening(json.RawMessage(mustJSON(payload)))
	if err != nil {
		fb, ferr := s.recommendInBoard("screening", msg, hint)
		if ferr == nil && fb != nil {
			fb.Reply = fmt.Sprintf("「%s」五算法选股暂不可用（%s），先按成分涨跌幅给出观察名单。", boardLabel, err.Error())
			if len(fb.Blocks) > 0 && fb.Blocks[0].Type == "text" {
				fb.Blocks[0].Text = fb.Reply
			}
			return fb, nil
		}
		return &ChatResponse{
			Reply:  "板块选股暂不可用：" + err.Error(),
			Intent: "screening",
			Blocks: []Block{{Type: "suggestions", Items: []string{"今日热点", "帮我选股"}}},
		}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	result["board"] = boardLabel
	scanned, _ := result["scanned"].(float64)
	qualified, _ := result["qualified"].(float64)
	picks, _ := result["picks"].([]interface{})
	if len(picks) > limit {
		picks = picks[:limit]
		result["picks"] = picks
	}
	if len(picks) == 0 {
		fb, _ := s.recommendInBoard("screening", msg, hint)
		if fb != nil {
			prefix := ""
			if nl != nil && nl.Summary != "" {
				prefix = nl.Summary + " "
			}
			fb.Reply = prefix + fmt.Sprintf("在「%s」跑完算法后没有达标票，先按板块涨跌幅给出观察名单。", boardLabel)
			if len(fb.Blocks) > 0 && fb.Blocks[0].Type == "text" {
				fb.Blocks[0].Text = fb.Reply
			}
			return fb, nil
		}
	}
	understand := ""
	if nl != nil && nl.Summary != "" {
		understand = nl.Summary + " "
		if nl.ModelCode != "" && nl.ModelCode != "quant-rules" {
			understand += fmt.Sprintf("（理解模型 %s）", nl.ModelCode)
		}
	}
	algoText := ""
	if len(algos) > 0 {
		algoText = "算法：" + strings.Join(algos, "/") + "。"
	}
	reply := fmt.Sprintf("%s「%s」板块选股完成：%s扫描 %.0f 只成分，列出前 %d 只达标（共达标 %.0f）。",
		understand, boardLabel, algoText, scanned, len(picks), qualified)
	s.saveScreeningRecord(picks, reply)
	blocks := []Block{
		{Type: "text", Text: reply},
		{Type: "screen_picks", Title: boardLabel + " · 选股结果", Data: result, Items: picks, Meta: map[string]interface{}{"pageSize": 10, "board": boardLabel}},
		{Type: "boards", Title: "目标板块", Items: matched},
		{Type: "suggestions", Items: []string{fmt.Sprintf("在%s推荐", hintOr(boardLabel, hint)), "今日热点", "帮我选股"}},
	}
	ws := &WorkspaceHint{
		Type: "sector", Symbol: matched[0].Code, Name: matched[0].Name, Tab: "overview",
		BoardCode: matched[0].Code, BoardName: matched[0].Name, Topic: hintOr(boardLabel, hint),
	}
	return &ChatResponse{Reply: reply, Intent: "screening", Blocks: blocks, Workspace: ws}, nil
}

func mustJSON(v interface{}) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
