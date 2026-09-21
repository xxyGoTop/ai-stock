package companion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type AnomalyRules struct {
	ChangeMild        float64 `json:"changeMild"`
	ChangeStrong      float64 `json:"changeStrong"`
	VolumeRatioMild   float64 `json:"volumeRatioMild"`
	VolumeRatioStrong float64 `json:"volumeRatioStrong"`
	TurnoverMild      float64 `json:"turnoverMild"`
	FundMildYi        float64 `json:"fundMildYi"`
	FundStrongYi      float64 `json:"fundStrongYi"`
	FundPctMild       float64 `json:"fundPctMild"`
	FundPctStrong     float64 `json:"fundPctStrong"`
	SuperMildYi       float64 `json:"superMildYi"`
	LiftMild          float64 `json:"liftMild"`
	LiftNotable       float64 `json:"liftNotable"`
	LiftStrong        float64 `json:"liftStrong"`
}

type RulesStore struct {
	mu   sync.Mutex
	path string
}

func DefaultAnomalyRules() AnomalyRules {
	return AnomalyRules{
		ChangeMild:        3,
		ChangeStrong:      5,
		VolumeRatioMild:   1.8,
		VolumeRatioStrong: 2.5,
		TurnoverMild:      8,
		FundMildYi:        0.1,
		FundStrongYi:      0.5,
		FundPctMild:       2,
		FundPctStrong:     5,
		SuperMildYi:       0.3,
		LiftMild:          1.5,
		LiftNotable:       2.2,
		LiftStrong:        4,
	}
}

func NewRulesStore() *RulesStore {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &RulesStore{path: filepath.Join(dir, "anomaly_rules.json")}
}

func (s *RulesStore) Get() AnomalyRules {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return DefaultAnomalyRules()
	}
	var rules AnomalyRules
	if json.Unmarshal(raw, &rules) != nil {
		return DefaultAnomalyRules()
	}
	return rules.Normalize()
}

func (s *RulesStore) Save(rules AnomalyRules) (AnomalyRules, error) {
	next := rules.Normalize()
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return next, err
	}
	if err := os.WriteFile(s.path, raw, 0o644); err != nil {
		return next, err
	}
	return next, nil
}

func (r AnomalyRules) Normalize() AnomalyRules {
	d := DefaultAnomalyRules()
	r.ChangeMild = clamp(r.ChangeMild, 0.5, 20, d.ChangeMild)
	r.ChangeStrong = clamp(r.ChangeStrong, r.ChangeMild, 30, d.ChangeStrong)
	r.VolumeRatioMild = clamp(r.VolumeRatioMild, 0.8, 8, d.VolumeRatioMild)
	r.VolumeRatioStrong = clamp(r.VolumeRatioStrong, r.VolumeRatioMild, 12, d.VolumeRatioStrong)
	r.TurnoverMild = clamp(r.TurnoverMild, 1, 40, d.TurnoverMild)
	r.FundMildYi = clamp(r.FundMildYi, 0.01, 10, d.FundMildYi)
	r.FundStrongYi = clamp(r.FundStrongYi, r.FundMildYi, 20, d.FundStrongYi)
	r.FundPctMild = clamp(r.FundPctMild, 0.5, 20, d.FundPctMild)
	r.FundPctStrong = clamp(r.FundPctStrong, r.FundPctMild, 40, d.FundPctStrong)
	r.SuperMildYi = clamp(r.SuperMildYi, 0.01, 10, d.SuperMildYi)
	r.LiftMild = clamp(r.LiftMild, 0.5, 10, d.LiftMild)
	r.LiftNotable = clamp(r.LiftNotable, r.LiftMild, 15, d.LiftNotable)
	r.LiftStrong = clamp(r.LiftStrong, r.LiftNotable, 20, d.LiftStrong)
	return r
}

func clamp(v, lo, hi, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
