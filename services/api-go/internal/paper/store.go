package paper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const startCash = 1_000_000.0

type Account struct {
	Cash      float64    `json:"cash"`
	MarketValue float64  `json:"marketValue"`
	Equity    float64    `json:"equity"`
	PnL       float64    `json:"pnl"`
	PnLPct    float64    `json:"pnlPct"`
	Positions []Position `json:"positions"`
	Orders    []Order    `json:"orders"`
}

type Position struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Qty           int     `json:"qty"`
	Available     int     `json:"available"`
	Cost          float64 `json:"cost"`
	Price         float64 `json:"price"`
	MarketValue   float64 `json:"marketValue"`
	PnL           float64 `json:"pnl"`
}

type Order struct {
	ID        string  `json:"id"`
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Qty       int     `json:"qty"`
	Amount    float64 `json:"amount"`
	Fee       float64 `json:"fee"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"createdAt"`
}

type PlaceReq struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Side   string  `json:"side"`
	Price  float64 `json:"price"`
	Qty    int     `json:"qty"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &Store{path: filepath.Join(dir, "paper.json")}
}

func (s *Store) Snapshot(quotes map[string]float64) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, err := s.load()
	if err != nil {
		return nil, err
	}
	today := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
	s.releaseT1(acc, today)
	s.mark(acc, quotes)
	return acc, nil
}

func (s *Store) Place(req PlaceReq, lastPrice float64) (*Account, error) {
	if req.Qty < 100 || req.Qty%100 != 0 {
		return nil, fmt.Errorf("数量必须是 100 的整数倍")
	}
	if req.Price <= 0 {
		req.Price = lastPrice
	}
	if req.Price <= 0 {
		return nil, fmt.Errorf("价格无效")
	}
	side := req.Side
	if side != "buy" && side != "sell" {
		return nil, fmt.Errorf("只支持 buy / sell")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, err := s.load()
	if err != nil {
		return nil, err
	}
	now := time.Now().In(time.FixedZone("CST", 8*3600))
	s.releaseT1(acc, now.Format("2006-01-02"))
	amount := req.Price * float64(req.Qty)
	fee := amount * 0.00025
	if side == "sell" {
		fee += amount * 0.001
	}
	if side == "buy" {
		if acc.Cash < amount+fee {
			return nil, fmt.Errorf("可用资金不足")
		}
		acc.Cash -= amount + fee
		pos := findPos(acc, req.Symbol)
		if pos == nil {
			acc.Positions = append(acc.Positions, Position{Symbol: req.Symbol, Name: req.Name, Qty: req.Qty, Available: 0, Cost: req.Price, Price: req.Price})
		} else {
			total := pos.Cost*float64(pos.Qty) + amount
			pos.Qty += req.Qty
			pos.Cost = total / float64(pos.Qty)
			pos.Name = req.Name
		}
	} else {
		pos := findPos(acc, req.Symbol)
		if pos == nil || pos.Available < req.Qty {
			return nil, fmt.Errorf("可卖数量不足（T+1）")
		}
		acc.Cash += amount - fee
		pos.Qty -= req.Qty
		pos.Available -= req.Qty
		if pos.Qty == 0 {
			acc.Positions = removePos(acc, req.Symbol)
		}
	}
	acc.Orders = append([]Order{{
		ID:        now.Format("20060102150405"),
		Symbol:    req.Symbol,
		Name:      req.Name,
		Side:      side,
		Price:     req.Price,
		Qty:       req.Qty,
		Amount:    amount,
		Fee:       fee,
		Status:    "filled",
		CreatedAt: now.Format("2006-01-02 15:04"),
	}}, acc.Orders...)
	if len(acc.Orders) > 80 {
		acc.Orders = acc.Orders[:80]
	}
	s.releaseT1(acc, now.Format("2006-01-02"))
	s.mark(acc, map[string]float64{req.Symbol: req.Price})
	if err := s.save(acc); err != nil {
		return nil, err
	}
	return acc, nil
}

func (s *Store) load() (*Account, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Account{Cash: startCash, Equity: startCash, Positions: []Position{}, Orders: []Order{}}, nil
		}
		return nil, err
	}
	var acc Account
	if err := json.Unmarshal(raw, &acc); err != nil {
		return nil, err
	}
	if acc.Positions == nil {
		acc.Positions = []Position{}
	}
	if acc.Orders == nil {
		acc.Orders = []Order{}
	}
	return &acc, nil
}

func (s *Store) save(acc *Account) error {
	raw, err := json.MarshalIndent(acc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func (s *Store) mark(acc *Account, quotes map[string]float64) {
	mv := 0.0
	for i := range acc.Positions {
		p := &acc.Positions[i]
		if q, ok := quotes[p.Symbol]; ok && q > 0 {
			p.Price = q
		}
		if p.Price <= 0 {
			p.Price = p.Cost
		}
		p.MarketValue = p.Price * float64(p.Qty)
		p.PnL = (p.Price - p.Cost) * float64(p.Qty)
		mv += p.MarketValue
	}
	acc.MarketValue = mv
	acc.Equity = acc.Cash + mv
	acc.PnL = acc.Equity - startCash
	acc.PnLPct = acc.PnL / startCash * 100
}

func (s *Store) releaseT1(acc *Account, today string) {
	bought := map[string]int{}
	for _, o := range acc.Orders {
		if o.Side == "buy" && len(o.CreatedAt) >= 10 && o.CreatedAt[:10] == today {
			bought[o.Symbol] += o.Qty
		}
	}
	for i := range acc.Positions {
		p := &acc.Positions[i]
		p.Available = p.Qty - bought[p.Symbol]
		if p.Available < 0 {
			p.Available = 0
		}
	}
}

func findPos(acc *Account, symbol string) *Position {
	for i := range acc.Positions {
		if acc.Positions[i].Symbol == symbol {
			return &acc.Positions[i]
		}
	}
	return nil
}

func removePos(acc *Account, symbol string) []Position {
	out := acc.Positions[:0]
	for _, p := range acc.Positions {
		if p.Symbol != symbol {
			out = append(out, p)
		}
	}
	return out
}
