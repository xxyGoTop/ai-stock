package watchlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

const maxItems = 40

const (
	CategoryDefault      = "default"
	CategoryTomorrowPlan = "tomorrow_plan"
)

// TradePlan 明日/今日操作计划（站稳买入、止损、分批等）。
type TradePlan struct {
	BuyLow     float64 `json:"buyLow,omitempty"`
	BuyHigh    float64 `json:"buyHigh,omitempty"`
	Stop       float64 `json:"stop,omitempty"`
	Target1    float64 `json:"target1,omitempty"`
	Target2    float64 `json:"target2,omitempty"`
	BuyBatches int     `json:"buyBatches,omitempty"` // 分几次买入
	Position   float64 `json:"position,omitempty"`   // 建议仓位比例，如 0.1
	Stance     string  `json:"stance,omitempty"`
	EntryType  string  `json:"entryType,omitempty"`
	Note       string  `json:"note,omitempty"`
	Action     string  `json:"action,omitempty"` // 买入 / 观望 / 减仓 等
}

type Item struct {
	ID            string     `json:"id"`
	Symbol        string     `json:"symbol"`
	Name          string     `json:"name"`
	Market        string     `json:"market"`
	Category      string     `json:"category"` // default | tomorrow_plan
	SortOrder     int        `json:"sortOrder"`
	CreatedAt     string     `json:"createdAt"`
	PlanForDate   string     `json:"planForDate,omitempty"` // 计划生效日 YYYY-MM-DD
	Plan          *TradePlan `json:"plan,omitempty"`
	Price         float64    `json:"price,omitempty"`
	Change        float64    `json:"change,omitempty"`
	ChangePercent float64    `json:"changePercent,omitempty"`
	Industry      string     `json:"industry,omitempty"`
}

type List struct {
	Items []Item `json:"items"`
}

type UpsertInput struct {
	Symbol      string
	Name        string
	Market      string
	Category    string
	PlanForDate string
	Plan        *TradePlan
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &Store{path: filepath.Join(dir, "watchlist.json")}
}

func NormalizeCategory(c string) string {
	c = strings.TrimSpace(c)
	switch c {
	case CategoryTomorrowPlan, "明日计划", "tomorrow", "plan":
		return CategoryTomorrowPlan
	default:
		return CategoryDefault
	}
}

func CategoryLabel(c string) string {
	if NormalizeCategory(c) == CategoryTomorrowPlan {
		return "明日计划"
	}
	return "普通自选"
}

func NextPlanDate(now time.Time) string {
	loc := time.FixedZone("CST", 8*3600)
	t := now.In(loc)
	// 简单取下一自然日；周末顺延到周一
	d := t.AddDate(0, 0, 1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d.Format("2006-01-02")
}

func TodayCST(now time.Time) string {
	return now.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
}

func (s *Store) All() ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		normalizeItem(&list.Items[i])
	}
	return list.Items, nil
}

func (s *Store) ByCategory(category string) ([]Item, error) {
	cat := NormalizeCategory(category)
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0)
	for _, it := range all {
		if NormalizeCategory(it.Category) == cat {
			out = append(out, it)
		}
	}
	return out, nil
}

// DueTodayOps 明日计划里「计划日已到/过」的标的，作为今日操作推送。
func (s *Store) DueTodayOps(now time.Time) ([]Item, error) {
	today := TodayCST(now)
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0)
	for _, it := range all {
		if NormalizeCategory(it.Category) != CategoryTomorrowPlan {
			continue
		}
		if it.PlanForDate == "" || it.PlanForDate <= today {
			out = append(out, it)
		}
	}
	return out, nil
}

func (s *Store) Add(symbol, name, market string) (*Item, error) {
	return s.Upsert(UpsertInput{Symbol: symbol, Name: name, Market: market, Category: CategoryDefault})
}

func (s *Store) Upsert(in UpsertInput) (*Item, error) {
	symbol := provider.PadSymbol(in.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol required")
	}
	name := in.Name
	if name == "" {
		name = symbol
	}
	market := in.Market
	if market == "" {
		market = string(provider.GuessMarket(symbol))
	}
	cat := NormalizeCategory(in.Category)
	planDate := strings.TrimSpace(in.PlanForDate)
	if cat == CategoryTomorrowPlan && planDate == "" {
		planDate = NextPlanDate(time.Now())
	}
	plan := normalizePlan(in.Plan)

	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Symbol == symbol {
			it := &list.Items[i]
			if name != "" {
				it.Name = name
			}
			if market != "" {
				it.Market = market
			}
			it.Category = cat
			if cat == CategoryTomorrowPlan {
				it.PlanForDate = planDate
				if plan != nil {
					it.Plan = plan
				} else if it.Plan == nil {
					it.Plan = &TradePlan{BuyBatches: 2, Action: "观察"}
				}
			}
			normalizeItem(it)
			if err := s.save(list); err != nil {
				return nil, err
			}
			cp := *it
			return &cp, nil
		}
	}
	if len(list.Items) >= maxItems {
		return nil, fmt.Errorf("自选最多 %d 只", maxItems)
	}
	item := Item{
		ID:          symbol,
		Symbol:      symbol,
		Name:        name,
		Market:      market,
		Category:    cat,
		SortOrder:   len(list.Items),
		CreatedAt:   time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04"),
		PlanForDate: planDate,
		Plan:        plan,
	}
	if cat == CategoryTomorrowPlan && item.Plan == nil {
		item.Plan = &TradePlan{BuyBatches: 2, Action: "观察"}
	}
	normalizeItem(&item)
	list.Items = append([]Item{item}, list.Items...)
	if err := s.save(list); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) Update(symbol string, patch UpsertInput) (*Item, error) {
	symbol = provider.PadSymbol(symbol)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Symbol != symbol && list.Items[i].ID != symbol {
			continue
		}
		it := &list.Items[i]
		if patch.Name != "" {
			it.Name = patch.Name
		}
		if patch.Market != "" {
			it.Market = patch.Market
		}
		if patch.Category != "" {
			it.Category = NormalizeCategory(patch.Category)
		}
		if patch.PlanForDate != "" {
			it.PlanForDate = patch.PlanForDate
		}
		if patch.Plan != nil {
			it.Plan = normalizePlan(patch.Plan)
		}
		if NormalizeCategory(it.Category) == CategoryTomorrowPlan && it.PlanForDate == "" {
			it.PlanForDate = NextPlanDate(time.Now())
		}
		if NormalizeCategory(it.Category) == CategoryDefault {
			// 移回普通自选时保留 plan 字段以便再启用，但清空日期也可
		}
		normalizeItem(it)
		if err := s.save(list); err != nil {
			return nil, err
		}
		cp := *it
		return &cp, nil
	}
	return nil, fmt.Errorf("not found")
}

func (s *Store) Remove(id string) error {
	id = provider.PadSymbol(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return err
	}
	out := list.Items[:0]
	for _, it := range list.Items {
		if it.ID != id && it.Symbol != id {
			out = append(out, it)
		}
	}
	list.Items = out
	return s.save(list)
}

func (s *Store) Has(symbol string) bool {
	symbol = provider.PadSymbol(symbol)
	items, err := s.All()
	if err != nil {
		return false
	}
	for _, it := range items {
		if it.Symbol == symbol {
			return true
		}
	}
	return false
}

func normalizeItem(it *Item) {
	if it.Category == "" {
		it.Category = CategoryDefault
	} else {
		it.Category = NormalizeCategory(it.Category)
	}
	if it.Plan != nil {
		it.Plan = normalizePlan(it.Plan)
	}
}

func normalizePlan(p *TradePlan) *TradePlan {
	if p == nil {
		return nil
	}
	cp := *p
	if cp.BuyBatches <= 0 {
		cp.BuyBatches = 2
	}
	if cp.BuyBatches > 5 {
		cp.BuyBatches = 5
	}
	return &cp
}

func (s *Store) load() (*List, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &List{Items: []Item{}}, nil
		}
		return nil, err
	}
	var list List
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	if list.Items == nil {
		list.Items = []Item{}
	}
	return &list, nil
}

func (s *Store) save(list *List) error {
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}
