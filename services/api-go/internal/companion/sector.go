package companion

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

var reSectorStudy = regexp.MustCompile(`(?:看看|打开|研究|查看|分析|关注)?\s*([^\s，,。！？、/]{2,12}?)(?:板块|概念|行业|题材)(?:怎么样|如何|怎样|强度|走势)?`)

func extractSectorHint(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	if h := extractBoardHint(msg); h != "" {
		return h
	}
	if m := reSectorStudy.FindStringSubmatch(msg); len(m) > 1 {
		return cleanBoardHint(m[1])
	}
	// 「半导体板块」「AI算力题材」整句
	for _, suf := range []string{"板块", "概念", "行业", "题材"} {
		if strings.HasSuffix(msg, suf) {
			return cleanBoardHint(strings.TrimSuffix(msg, suf))
		}
	}
	return ""
}

func looksLikeSectorStudy(msg string) bool {
	if looksLikeScreening(msg) {
		return false
	}
	if strings.Contains(msg, "推荐") || strings.Contains(msg, "选股") {
		return false
	}
	hint := extractSectorHint(msg)
	if hint == "" {
		return false
	}
	return strings.Contains(msg, "板块") ||
		strings.Contains(msg, "概念") ||
		strings.Contains(msg, "行业") ||
		strings.Contains(msg, "题材")
}

func looksLikeTopicStudy(msg string) bool {
	return strings.Contains(msg, "题材") || strings.Contains(msg, "热点主题")
}

// studySector 打开行业/概念板块研究工作台。
func (s *Service) studySector(msg string, preferred string) (*ChatResponse, error) {
	hint := strings.TrimSpace(preferred)
	if hint == "" {
		hint = extractSectorHint(msg)
	}
	if hint == "" {
		return &ChatResponse{
			Reply:  "说一下要看的板块，例如「看看半导体板块」「打开新能源题材」。",
			Intent: "sector",
			Blocks: []Block{{Type: "suggestions", Items: []string{"看看半导体板块", "打开新能源题材", "今日热点", "帮我选股"}}},
		}, nil
	}

	asTopic := looksLikeTopicStudy(msg) && !strings.Contains(msg, "板块") && !strings.Contains(msg, "行业") && !strings.Contains(msg, "概念")

	board, err := s.resolveBoard(hint)
	var matched []provider.HotBoard
	if err == nil && board != nil {
		matched = []provider.HotBoard{*board}
	} else {
		matched = s.matchBoardsByHints(s.loadCandidateBoards(), []string{hint}, hint)
	}
	if len(matched) == 0 {
		return &ChatResponse{
			Reply:  fmt.Sprintf("暂时没定位到「%s」相关板块。可以换个说法，例如「半导体板块」「化学制药」。", hint),
			Intent: "sector",
			Blocks: []Block{{Type: "suggestions", Items: []string{"看看半导体板块", "打开创新药板块", "今日热点"}}},
		}, nil
	}

	primary := matched[0]
	stocks, _ := s.bundle.BoardStocks(primary.Code, 20)
	wsType := "sector"
	if asTopic {
		wsType = "topic"
	}

	lines := []string{
		fmt.Sprintf("%s（%s）今日 %+.2f%%，近5日 %+.2f%%。", primary.Name, primary.Code, primary.ChangePercent, primary.Change5),
	}
	if primary.Leader != "" {
		lines = append(lines, fmt.Sprintf("领涨：%s %+.2f%%。", primary.Leader, primary.LeaderChangePercent))
	}
	if len(matched) > 1 {
		names := make([]string, 0, len(matched)-1)
		for _, b := range matched[1:] {
			names = append(names, fmt.Sprintf("%s %+.2f%%", b.Name, b.ChangePercent))
		}
		lines = append(lines, "相关："+strings.Join(names, "、")+"。")
	}
	if len(stocks) > 0 {
		top := make([]string, 0, 3)
		for i, st := range stocks {
			if i >= 3 {
				break
			}
			top = append(top, fmt.Sprintf("%s %+.2f%%", st.Name, st.ChangePercent))
		}
		lines = append(lines, "成分靠前："+strings.Join(top, "、")+"。")
	}
	reply := strings.Join(lines, "")

	blocks := []Block{
		{Type: "text", Text: reply},
		{Type: "boards", Title: "研究板块", Items: matched},
	}
	if len(stocks) > 0 {
		picks := picksFromBoardStocks(primary.Name, stocks, 15)
		blocks = append(blocks, Block{
			Type:  "picks",
			Title: primary.Name + " · 成分观察",
			Items: picks,
			Meta:  map[string]interface{}{"pageSize": 8, "board": primary.Name},
		})
	}

	// 题材：附带相关快讯
	var news []provider.NewsItem
	if hot, herr := s.bundle.HotFeed(20); herr == nil && hot != nil {
		news = filterNewsByHint(hot.News, hint)
		if len(news) > 0 {
			blocks = append(blocks, Block{Type: "news", Title: "相关快讯", Items: news})
		}
	}

	blocks = append(blocks, Block{
		Type:  "suggestions",
		Title: "可以继续",
		Items: []string{
			fmt.Sprintf("在%s选股", primary.Name),
			fmt.Sprintf("在%s推荐", primary.Name),
			"今日热点",
			"今日行情",
		},
	})

	if s.mem != nil {
		kind := "sector"
		if wsType == "topic" {
			kind = "topic"
		}
		s.mem.RecordResearch(primary.Code, kind, primary.Name)
	}

	return &ChatResponse{
		Reply:  reply,
		Intent: wsType,
		Blocks: blocks,
		Workspace: &WorkspaceHint{
			Type:      wsType,
			Symbol:    primary.Code,
			Name:      primary.Name,
			Tab:       "overview",
			BoardCode: primary.Code,
			BoardName: primary.Name,
			Topic:     hint,
		},
	}, nil
}

func filterNewsByHint(items []provider.NewsItem, hint string) []provider.NewsItem {
	hint = strings.TrimSpace(hint)
	if hint == "" || len(items) == 0 {
		return nil
	}
	out := make([]provider.NewsItem, 0, 5)
	for _, n := range items {
		if strings.Contains(n.Title, hint) {
			out = append(out, n)
			if len(out) >= 5 {
				break
			}
		}
	}
	return out
}
