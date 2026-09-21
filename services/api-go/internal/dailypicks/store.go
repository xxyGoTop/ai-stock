package dailypicks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Pick struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	ChangePercent float64 `json:"changePercent"`
	Reason        string  `json:"reason,omitempty"`
	Board         string  `json:"board,omitempty"`
	Score         float64 `json:"score,omitempty"`
	PrimaryName   string  `json:"primaryName,omitempty"`
	Industry      string  `json:"industry,omitempty"`
}

type Record struct {
	Date    string `json:"date"`
	Kind    string `json:"kind"` // recommend | screening | preopen | intraday | close_auction | review
	Title   string `json:"title"`
	AsOf    string `json:"asOf"`
	Count   int    `json:"count"`
	Picks   []Pick `json:"picks"`
	Summary string `json:"summary,omitempty"`
}

type fileData struct {
	Records map[string]Record `json:"records"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &Store{path: filepath.Join(dir, "daily_picks.json")}
}

func Today() string {
	return time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
}

func key(date, kind string) string {
	return date + ":" + kind
}

func (s *Store) Save(rec Record) error {
	if rec.Date == "" {
		rec.Date = Today()
	}
	if rec.AsOf == "" {
		rec.AsOf = time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04")
	}
	rec.Count = len(rec.Picks)
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return err
	}
	data.Records[key(rec.Date, rec.Kind)] = rec
	return s.save(data)
}

func (s *Store) Get(date, kind string) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	if date == "" {
		date = Today()
	}
	rec, ok := data.Records[key(date, kind)]
	if !ok {
		return nil, nil
	}
	return &rec, nil
}

// Latest 返回某 kind 最近一条有标的的记录（跨日期）。
func (s *Store) Latest(kind string) (*Record, error) {
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Kind == kind && len(list[i].Picks) > 0 {
			rec := list[i]
			return &rec, nil
		}
	}
	return nil, nil
}

// LatestAny 按 kinds 优先级取最近一条有标的记录。
func (s *Store) LatestAny(kinds ...string) (*Record, error) {
	for _, k := range kinds {
		rec, err := s.Latest(k)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			return rec, nil
		}
	}
	return nil, nil
}

func (s *Store) List() ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(data.Records))
	for _, r := range data.Records {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Date == out[j].Date {
			return out[i].AsOf > out[j].AsOf
		}
		return out[i].Date > out[j].Date
	})
	return out, nil
}

func (s *Store) load() (*fileData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &fileData{Records: map[string]Record{}}, nil
		}
		return nil, err
	}
	var data fileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if data.Records == nil {
		data.Records = map[string]Record{}
	}
	return &data, nil
}

func (s *Store) save(data *fileData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}
