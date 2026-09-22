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

func picksFromBoardStocks(boardName string, stocks []provider.BoardStock, limit int) []RecommendPick {
	if limit <= 0 {
		limit = 30
	}
	out := make([]RecommendPick, 0, limit)
	for _, st := range stocks {
		if strings.Contains(strings.ToUpper(st.Name), "ST") {
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

func (s *Service) recommendInBoard(action, msg, hint string) (*ChatResponse, error) {
	board, err := s.resolveBoard(hint)
	if err != nil || board == nil {
		return &ChatResponse{
			Reply:  fmt.Sprintf("没有找到「%s」对应板块。可以说「在半导体选股」「在新能源推荐」，或先看「今日热点」。", hint),
			Intent: "recommend",
			Blocks: []Block{{
				Type:  "suggestions",
				Title: "试试",
				Items: []string{"在半导体推荐", "在新能源选股", "今日热点", "帮我选股"},
			}},
		}, nil
	}
	stocks, err := s.bundle.BoardStocks(board.Code, 40)
	if err != nil || len(stocks) == 0 {
		return &ChatResponse{
			Reply:  fmt.Sprintf("已定位到「%s」，但暂时拉不到成分股（行情源繁忙）。可稍后再试或先看全市场推荐。", board.Name),
			Intent: "recommend",
			Blocks: []Block{{Type: "suggestions", Items: []string{"盘中推荐", "今日热点", "帮我选股"}}},
		}, nil
	}
	title := fmt.Sprintf("%s · 板块推荐", board.Name)
	kind := "recommend"
	switch action {
	case "preopen":
		kind, title = "preopen", fmt.Sprintf("%s · 盘前推荐", board.Name)
	case "intraday":
		kind, title = "intraday", fmt.Sprintf("%s · 盘中推荐", board.Name)
	case "close_auction":
		kind, title = "close_auction", fmt.Sprintf("%s · 尾盘推荐", board.Name)
	case "review":
		kind, title = "review", fmt.Sprintf("%s · 复盘观察", board.Name)
	case "screening":
		kind, title = "screening", fmt.Sprintf("%s · 板块选股", board.Name)
	}
	picks := picksFromBoardStocks(board.Name, stocks, 30)
	if len(picks) == 0 {
		return &ChatResponse{
			Reply:  fmt.Sprintf("「%s」成分股暂无可用样本。", board.Name),
			Intent: "recommend",
		}, nil
	}
	summary := fmt.Sprintf("%s：在「%s」成分中按涨跌幅取出 %d 只。板块今日 %+.2f%%，领涨 %s。",
		title, board.Name, len(picks), board.ChangePercent, board.Leader)
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
		{Type: "picks", Title: title, Items: picks, Meta: map[string]interface{}{"pageSize": 10, "board": board.Name, "boardCode": board.Code}},
		{Type: "boards", Title: "目标板块", Items: []provider.HotBoard{*board}},
		{Type: "suggestions", Title: "可以继续", Items: []string{
			fmt.Sprintf("在%s选股", board.Name),
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
			Type: "stock", Symbol: picks[0].Symbol, Name: picks[0].Name, Tab: "overview",
		},
	}, nil
}

func (s *Service) screenInBoard(msg, hint string) (*ChatResponse, error) {
	board, err := s.resolveBoard(hint)
	if err != nil || board == nil {
		return &ChatResponse{
			Reply:  fmt.Sprintf("没有找到「%s」对应板块，无法在板块内选股。可以说「在半导体选股」。", hint),
			Intent: "screening",
			Blocks: []Block{{Type: "suggestions", Items: []string{"在半导体选股", "在新能源选股", "帮我选股", "今日热点"}}},
		}, nil
	}
	stocks, err := s.bundle.BoardStocks(board.Code, 80)
	if err != nil || len(stocks) == 0 {
		// fallback: recommend-style list from board
		return s.recommendInBoard("screening", msg, hint)
	}
	symbols := make([]string, 0, len(stocks))
	for _, st := range stocks {
		symbols = append(symbols, st.Symbol)
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"detail":   minInt(len(symbols), 120),
		"limit":    30,
		"symbols":  symbols,
		"board":    board.Name,
		"industry": board.Name,
	})
	raw, err := s.py.Screening(payload)
	if err != nil {
		// fallback to momentum list inside board
		fb, ferr := s.recommendInBoard("screening", msg, hint)
		if ferr == nil && fb != nil {
			fb.Reply = fmt.Sprintf("板块「%s」五算法选股暂不可用（%s），先按成分涨跌幅给出观察名单。", board.Name, err.Error())
			if len(fb.Blocks) > 0 && fb.Blocks[0].Type == "text" {
				fb.Blocks[0].Text = fb.Reply
			}
			return fb, nil
		}
		return &ChatResponse{
			Reply:  "板块选股暂不可用：" + err.Error(),
			Intent: "screening",
			Blocks: []Block{{Type: "suggestions", Items: []string{"今日热点", fmt.Sprintf("在%s推荐", board.Name)}}},
		}, nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	result["board"] = board.Name
	result["boardCode"] = board.Code
	scanned, _ := result["scanned"].(float64)
	qualified, _ := result["qualified"].(float64)
	picks, _ := result["picks"].([]interface{})
	if len(picks) > 30 {
		picks = picks[:30]
		result["picks"] = picks
	}
	if len(picks) == 0 {
		fb, _ := s.recommendInBoard("screening", msg, hint)
		if fb != nil {
			fb.Reply = fmt.Sprintf("在「%s」跑完五算法后没有达标票，先按板块涨跌幅给出观察名单。", board.Name)
			if len(fb.Blocks) > 0 && fb.Blocks[0].Type == "text" {
				fb.Blocks[0].Text = fb.Reply
			}
			return fb, nil
		}
	}
	reply := fmt.Sprintf("「%s」板块选股完成：扫描 %.0f 只成分，列出前 %d 只达标（共达标 %.0f）。", board.Name, scanned, len(picks), qualified)
	s.saveScreeningRecord(picks, reply)
	blocks := []Block{
		{Type: "text", Text: reply},
		{Type: "screen_picks", Title: board.Name + " · 选股结果", Data: result, Items: picks, Meta: map[string]interface{}{"pageSize": 10, "board": board.Name}},
		{Type: "boards", Title: "目标板块", Items: []provider.HotBoard{*board}},
		{Type: "suggestions", Items: []string{fmt.Sprintf("在%s推荐", board.Name), "今日热点", "帮我选股"}},
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
