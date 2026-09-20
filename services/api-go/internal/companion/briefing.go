package companion

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/dailypicks"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/python"
	"github.com/xxyGoTop/ai-stock/services/api-go/internal/watchlist"
)

type SessionPhase string

const (
	PhasePreOpen  SessionPhase = "preopen"
	PhaseIntraday SessionPhase = "intraday"
	PhaseCloseAuc SessionPhase = "close_auction"
	PhaseReview   SessionPhase = "review"
)

type Block struct {
	Type    string                 `json:"type"`
	Title   string                 `json:"title,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Items   interface{}            `json:"items,omitempty"`
	Quote   *provider.Quote        `json:"quote,omitempty"`
	Data    interface{}            `json:"data,omitempty"`
	Symbol  string                 `json:"symbol,omitempty"`
	Actions []string               `json:"actions,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

type SessionCard struct {
	Phase   SessionPhase     `json:"phase"`
	Title   string           `json:"title"`
	Active  bool             `json:"active"`
	Summary string           `json:"summary"`
	Picks   []RecommendPick  `json:"picks"`
}

type RecommendPick struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	ChangePercent float64 `json:"changePercent"`
	Reason        string  `json:"reason"`
	Board         string  `json:"board,omitempty"`
}

type Briefing struct {
	Phase         SessionPhase             `json:"phase"`
	PhaseLabel    string                   `json:"phaseLabel"`
	Greeting      string                   `json:"greeting"`
	MarketSummary string                   `json:"marketSummary"`
	Indices       []provider.Quote         `json:"indices"`
	Northbound    *provider.NorthboundFlow `json:"northbound,omitempty"`
	Boards        []provider.HotBoard      `json:"boards"`
	Topics        []provider.HotTopic      `json:"topics"`
	News          []provider.NewsItem      `json:"news"`
	SessionCards  []SessionCard            `json:"sessionCards"`
	WatchPreview  []watchlist.Item         `json:"watchPreview,omitempty"`
	Anomalies     []Anomaly                `json:"anomalies,omitempty"`
	Blocks        []Block                  `json:"blocks"`
	AsOf          string                   `json:"asOf"`
}

type WorkspaceHint struct {
	Type   string `json:"type"` // market | stock | empty
	Symbol string `json:"symbol,omitempty"`
	Name   string `json:"name,omitempty"`
	Tab    string `json:"tab,omitempty"` // overview | kline | analysis | paper
}

type ChatRequest struct {
	Message string `json:"message"`
	Symbol  string `json:"symbol,omitempty"`
	Action  string `json:"action,omitempty"` // analyze | watch | paper | kline | briefing | recommend
}

type ChatResponse struct {
	Reply     string         `json:"reply"`
	Intent    string         `json:"intent"`
	Blocks    []Block        `json:"blocks"`
	Workspace *WorkspaceHint `json:"workspace,omitempty"`
}

type Service struct {
	bundle *provider.Bundle
	py     *python.Client
	watch  *watchlist.Store
	picks  *dailypicks.Store
}

func New(bundle *provider.Bundle, py *python.Client, watch *watchlist.Store, picks *dailypicks.Store) *Service {
	return &Service{bundle: bundle, py: py, watch: watch, picks: picks}
}

func DetectPhase(now time.Time) SessionPhase {
	loc := time.FixedZone("CST", 8*3600)
	t := now.In(loc)
	wd := t.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return PhaseReview
	}
	mins := t.Hour()*60 + t.Minute()
	switch {
	case mins < 9*60+25:
		return PhasePreOpen
	case mins < 14*60+45:
		return PhaseIntraday
	case mins < 15*60:
		return PhaseCloseAuc
	default:
		return PhaseReview
	}
}

func phaseLabel(p SessionPhase) string {
	switch p {
	case PhasePreOpen:
		return "盘前"
	case PhaseIntraday:
		return "盘中"
	case PhaseCloseAuc:
		return "尾盘"
	default:
		return "收盘复盘"
	}
}

func (s *Service) BuildBriefing() (*Briefing, error) {
	phase := DetectPhase(time.Now())
	var (
		indices []provider.Quote
		feed    *provider.HotFeed
		nb      *provider.NorthboundFlow
		watch   []watchlist.Item
	)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		indices, _ = s.bundle.IndexQuotes()
	}()
	go func() {
		defer wg.Done()
		feed, _ = s.bundle.HotFeed(10)
	}()
	go func() {
		defer wg.Done()
		nb, _ = s.bundle.NorthboundFlow()
	}()
	go func() {
		defer wg.Done()
		if s.watch != nil {
			items, _ := s.watch.All()
			watch = items
			if len(watch) > 6 {
				watch = watch[:6]
			}
		}
	}()
	wg.Wait()

	if indices == nil {
		indices = []provider.Quote{}
	}
	boards := []provider.HotBoard{}
	topics := []provider.HotTopic{}
	news := []provider.NewsItem{}
	if feed != nil {
		boards = feed.Boards
		topics = feed.Topics
		news = feed.News
		if len(news) > 6 {
			news = news[:6]
		}
	}

	picks := picksFromBoards(boards, 30)
	cards := buildSessionCards(phase, picks, indices, nb)
	summary := buildMarketSummary(indices, boards, nb, phase)

	var anomalies []Anomaly
	if scan, err := s.ScanWatchAnomalies(); err == nil && scan != nil {
		anomalies = scan.Items
		if scan.Count > 0 {
			summary = summary + " " + scan.Summary
		}
	}

	greeting := fmt.Sprintf("我是你的投研伙伴。当前处于%s时段，先帮你扫一眼今天的市场。", phaseLabel(phase))
	if len(anomalies) > 0 {
		greeting = fmt.Sprintf("我是你的投研伙伴。当前处于%s时段。你的自选股中有 %d 只出现较明显异动，要不要先从这里开始？", phaseLabel(phase), len(anomalies))
	}

	b := &Briefing{
		Phase:         phase,
		PhaseLabel:    phaseLabel(phase),
		Greeting:      greeting,
		MarketSummary: summary,
		Indices:       indices,
		Northbound:    nb,
		Boards:        boards,
		Topics:        topics,
		News:          news,
		SessionCards:  cards,
		WatchPreview:  watch,
		Anomalies:     anomalies,
		AsOf:          time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04"),
	}
	b.Blocks = briefingBlocks(b)
	return b, nil
}

func picksFromBoards(boards []provider.HotBoard, limit int) []RecommendPick {
	out := make([]RecommendPick, 0, limit)
	seen := map[string]bool{}
	for _, b := range boards {
		if b.LeaderCode == "" || b.Leader == "" {
			continue
		}
		sym := provider.PadSymbol(b.LeaderCode)
		if seen[sym] {
			continue
		}
		seen[sym] = true
		out = append(out, RecommendPick{
			Symbol:        sym,
			Name:          b.Leader,
			ChangePercent: b.LeaderChangePercent,
			Reason:        fmt.Sprintf("%s板块领涨", b.Name),
			Board:         b.Name,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func buildSessionCards(phase SessionPhase, picks []RecommendPick, indices []provider.Quote, nb *provider.NorthboundFlow) []SessionCard {
	idxText := "等待指数数据"
	if len(indices) > 0 {
		parts := make([]string, 0, len(indices))
		for _, q := range indices {
			parts = append(parts, fmt.Sprintf("%s %+.2f%%", q.Name, q.ChangePercent))
		}
		idxText = strings.Join(parts, " · ")
	}
	nbText := "北向资金暂无数据"
	if nb != nil {
		nbText = nb.Text
	}
	return []SessionCard{
		{
			Phase: PhasePreOpen, Title: "盘前推荐", Active: phase == PhasePreOpen,
			Summary: fmt.Sprintf("结合隔夜情绪与热门板块，列出今日观察名单（%d 只）。", len(picks)), Picks: picks,
		},
		{
			Phase: PhaseIntraday, Title: "盘中推荐", Active: phase == PhaseIntraday,
			Summary: "跟踪主流板块与领涨股，适合盘中跟踪。指数：" + idxText, Picks: picks,
		},
		{
			Phase: PhaseCloseAuc, Title: "尾盘推荐", Active: phase == PhaseCloseAuc,
			Summary: "关注尾盘资金是否回流。" + nbText, Picks: picks,
		},
		{
			Phase: PhaseReview, Title: "收盘复盘", Active: phase == PhaseReview,
			Summary: "复盘全天强弱、北向与热点，准备明日观察。", Picks: picks,
		},
	}
}

func buildMarketSummary(indices []provider.Quote, boards []provider.HotBoard, nb *provider.NorthboundFlow, phase SessionPhase) string {
	var parts []string
	parts = append(parts, "【"+phaseLabel(phase)+"】")
	if len(indices) > 0 {
		up, down := 0, 0
		for _, q := range indices {
			if q.ChangePercent > 0 {
				up++
			} else if q.ChangePercent < 0 {
				down++
			}
			parts = append(parts, fmt.Sprintf("%s %+.2f%%", q.Name, q.ChangePercent))
		}
		tone := "分化"
		if up >= 2 {
			tone = "偏强"
		} else if down >= 2 {
			tone = "偏弱"
		}
		parts = append(parts, "大盘整体"+tone)
	}
	if len(boards) > 0 {
		top := boards[0]
		parts = append(parts, fmt.Sprintf("主流板块看 %s（%+.2f%%）", top.Name, top.ChangePercent))
	}
	if nb != nil {
		parts = append(parts, nb.Text)
	}
	return strings.Join(parts, "。") + "。"
}

func briefingBlocks(b *Briefing) []Block {
	blocks := []Block{
		{Type: "text", Text: b.Greeting},
		{Type: "text", Title: "今日行情", Text: b.MarketSummary},
		{Type: "indices", Title: "指数情况", Items: b.Indices},
	}
	if b.Northbound != nil {
		blocks = append(blocks, Block{Type: "northbound", Title: "北向资金", Data: b.Northbound, Text: b.Northbound.Text})
	}
	if len(b.Boards) > 0 {
		blocks = append(blocks, Block{Type: "boards", Title: "热门 / 主流板块", Items: b.Boards})
	}
	if len(b.Topics) > 0 {
		blocks = append(blocks, Block{Type: "topics", Title: "热点主题", Items: b.Topics})
	}
	blocks = append(blocks, Block{Type: "session_cards", Title: "今日陪伴节奏", Items: b.SessionCards})
	if len(b.News) > 0 {
		blocks = append(blocks, Block{Type: "news", Title: "重要快讯", Items: b.News})
	}
	if len(b.Anomalies) > 0 {
		blocks = append(blocks, Block{
			Type:  "anomaly",
			Title: "自选异动",
			Text:  fmt.Sprintf("你的自选股中有 %d 只出现较明显异动。", len(b.Anomalies)),
			Items: b.Anomalies,
			Meta:  map[string]interface{}{"asOf": b.AsOf, "count": len(b.Anomalies)},
		})
	} else if len(b.WatchPreview) > 0 {
		blocks = append(blocks, Block{Type: "watchlist", Title: "自选速览", Items: b.WatchPreview})
	}
	suggest := []string{"今天行情", "今日热点", "帮我选股", "盘中推荐", "看看自选异动", "分析贵州茅台"}
	if len(b.Anomalies) > 0 {
		suggest = []string{"看看自选异动", "分析" + b.Anomalies[0].Name, "今天行情", "帮我选股", "我的自选"}
	}
	blocks = append(blocks, Block{
		Type:  "suggestions",
		Title: "你可以继续问我",
		Items: suggest,
	})
	return blocks
}
