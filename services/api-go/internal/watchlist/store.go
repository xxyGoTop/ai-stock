package watchlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/provider"
)

const maxItems = 40

type Item struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Market        string  `json:"market"`
	SortOrder     int     `json:"sortOrder"`
	CreatedAt     string  `json:"createdAt"`
	Price         float64 `json:"price,omitempty"`
	Change        float64 `json:"change,omitempty"`
	ChangePercent float64 `json:"changePercent,omitempty"`
	Industry      string  `json:"industry,omitempty"`
}

type List struct {
	Items []Item `json:"items"`
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

func (s *Store) All() ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (s *Store) Add(symbol, name, market string) (*Item, error) {
	symbol = provider.PadSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol required")
	}
	if name == "" {
		name = symbol
	}
	if market == "" {
		market = string(provider.GuessMarket(symbol))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].Symbol == symbol {
			return &list.Items[i], nil
		}
	}
	if len(list.Items) >= maxItems {
		return nil, fmt.Errorf("自选最多 %d 只", maxItems)
	}
	item := Item{
		ID:        symbol,
		Symbol:    symbol,
		Name:      name,
		Market:    market,
		SortOrder: len(list.Items),
		CreatedAt: time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04"),
	}
	list.Items = append([]Item{item}, list.Items...)
	if err := s.save(list); err != nil {
		return nil, err
	}
	return &item, nil
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
