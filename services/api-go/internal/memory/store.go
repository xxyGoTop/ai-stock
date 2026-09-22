package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	maxResearch  = 40
	maxSummaries = 40
	maxThemes    = 12
	maxFacts     = 6
)

var (
	symbolRe = regexp.MustCompile(`\b([0-9]{6})\b`)
	themeHints = []string{
		"AI", "人工智能", "算力", "机器人", "脑机", "新能源", "光伏", "储能", "锂电",
		"半导体", "芯片", "医药", "创新药", "白酒", "消费", "军工", "航天", "有色",
		"黄金", "券商", "银行", "地产", "汽车", "智能驾驶", "华为", "苹果链",
	}
)

type Preferences struct {
	FocusThemes []string `json:"focusThemes"`
	Note        string   `json:"note"`
}

type ResearchItem struct {
	Key    string `json:"key"`
	Kind   string `json:"kind"` // symbol | topic | board
	Label  string `json:"label"`
	Count  int    `json:"count"`
	LastAt int64  `json:"lastAt"`
}

type ConversationSummary struct {
	ConversationID string   `json:"conversationId"`
	Topic          string   `json:"topic"`
	ImportantFacts []string `json:"importantFacts"`
	OpenQuestions  []string `json:"openQuestions"`
	UpdatedAt      int64    `json:"updatedAt"`
}

type Snapshot struct {
	Preferences Preferences           `json:"preferences"`
	Research    []ResearchItem        `json:"research"`
	Summaries   []ConversationSummary `json:"summaries"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &Store{path: filepath.Join(dir, "memory.json")}
}

func DefaultPreferences() Preferences {
	return Preferences{FocusThemes: []string{}, Note: ""}
}

func (s *Store) load() Snapshot {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return Snapshot{Preferences: DefaultPreferences(), Research: []ResearchItem{}, Summaries: []ConversationSummary{}}
	}
	var data Snapshot
	if json.Unmarshal(raw, &data) != nil {
		return Snapshot{Preferences: DefaultPreferences(), Research: []ResearchItem{}, Summaries: []ConversationSummary{}}
	}
	if data.Research == nil {
		data.Research = []ResearchItem{}
	}
	if data.Summaries == nil {
		data.Summaries = []ConversationSummary{}
	}
	data.Preferences = normalizePrefs(data.Preferences)
	return data
}

func (s *Store) save(data Snapshot) error {
	data.Preferences = normalizePrefs(data.Preferences)
	if len(data.Research) > maxResearch {
		sort.Slice(data.Research, func(i, j int) bool { return data.Research[i].LastAt > data.Research[j].LastAt })
		data.Research = data.Research[:maxResearch]
	}
	if len(data.Summaries) > maxSummaries {
		sort.Slice(data.Summaries, func(i, j int) bool { return data.Summaries[i].UpdatedAt > data.Summaries[j].UpdatedAt })
		data.Summaries = data.Summaries[:maxSummaries]
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func normalizePrefs(p Preferences) Preferences {
	seen := map[string]bool{}
	out := make([]string, 0, len(p.FocusThemes))
	for _, t := range p.FocusThemes {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
		if len(out) >= maxThemes {
			break
		}
	}
	p.FocusThemes = out
	p.Note = strings.TrimSpace(p.Note)
	if utf8.RuneCountInString(p.Note) > 200 {
		p.Note = string([]rune(p.Note)[:200])
	}
	return p
}

func (s *Store) Get() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) SavePreferences(p Preferences) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	data.Preferences = normalizePrefs(p)
	if err := s.save(data); err != nil {
		return data, err
	}
	return data, nil
}

func (s *Store) RecordResearch(key, kind, label string) Snapshot {
	key = strings.TrimSpace(key)
	if key == "" {
		return s.Get()
	}
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "topic"
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = key
	}
	now := time.Now().UnixMilli()

	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	idx := -1
	for i, r := range data.Research {
		if r.Key == key && r.Kind == kind {
			idx = i
			break
		}
	}
	if idx >= 0 {
		data.Research[idx].Count++
		data.Research[idx].LastAt = now
		if label != "" {
			data.Research[idx].Label = label
		}
	} else {
		data.Research = append(data.Research, ResearchItem{
			Key: key, Kind: kind, Label: label, Count: 1, LastAt: now,
		})
	}
	sort.Slice(data.Research, func(i, j int) bool { return data.Research[i].LastAt > data.Research[j].LastAt })
	_ = s.save(data)
	return data
}

func (s *Store) UpsertSummary(sum ConversationSummary) (ConversationSummary, error) {
	sum.ConversationID = strings.TrimSpace(sum.ConversationID)
	if sum.ConversationID == "" {
		return sum, fmt.Errorf("conversationId required")
	}
	sum.Topic = strings.TrimSpace(sum.Topic)
	sum.ImportantFacts = trimList(sum.ImportantFacts, maxFacts)
	sum.OpenQuestions = trimList(sum.OpenQuestions, maxFacts)
	sum.UpdatedAt = time.Now().UnixMilli()

	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	found := false
	for i, cur := range data.Summaries {
		if cur.ConversationID == sum.ConversationID {
			data.Summaries[i] = sum
			found = true
			break
		}
	}
	if !found {
		data.Summaries = append(data.Summaries, sum)
	}
	if err := s.save(data); err != nil {
		return sum, err
	}
	return sum, nil
}

func (s *Store) GetSummary(conversationID string) *ConversationSummary {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	for i := range data.Summaries {
		if data.Summaries[i].ConversationID == conversationID {
			cp := data.Summaries[i]
			return &cp
		}
	}
	return nil
}

func trimList(items []string, max int) []string {
	out := make([]string, 0, max)
	seen := map[string]bool{}
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" || seen[it] {
			continue
		}
		seen[it] = true
		out = append(out, it)
		if len(out) >= max {
			break
		}
	}
	return out
}

// RelevantContext builds a short prompt block for LLM injection.
func (s *Store) RelevantContext(conversationID string) string {
	snap := s.Get()
	var parts []string
	if len(snap.Preferences.FocusThemes) > 0 {
		parts = append(parts, "用户关注方向："+strings.Join(snap.Preferences.FocusThemes, "、"))
	}
	if note := strings.TrimSpace(snap.Preferences.Note); note != "" {
		parts = append(parts, "用户备注："+note)
	}
	if len(snap.Research) > 0 {
		n := len(snap.Research)
		if n > 8 {
			n = 8
		}
		labels := make([]string, 0, n)
		for i := 0; i < n; i++ {
			r := snap.Research[i]
			if r.Label != "" && r.Label != r.Key {
				labels = append(labels, fmt.Sprintf("%s(%s)", r.Label, r.Key))
			} else {
				labels = append(labels, r.Label)
			}
		}
		parts = append(parts, "近期研究过："+strings.Join(labels, "、"))
	}
	if sum := s.GetSummary(conversationID); sum != nil {
		if sum.Topic != "" {
			parts = append(parts, "本会话主题："+sum.Topic)
		}
		if len(sum.ImportantFacts) > 0 {
			parts = append(parts, "已知要点："+strings.Join(sum.ImportantFacts, "；"))
		}
		if len(sum.OpenQuestions) > 0 {
			parts = append(parts, "待跟进："+strings.Join(sum.OpenQuestions, "；"))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

type ChatTurn struct {
	Role    string
	Content string
}

// RefreshSummaryFromTurns builds a heuristic summary (no LLM) from recent turns.
func (s *Store) RefreshSummaryFromTurns(conversationID string, turns []ChatTurn, latestUser string) *ConversationSummary {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return nil
	}
	texts := make([]string, 0, len(turns)+1)
	for _, t := range turns {
		if strings.TrimSpace(t.Content) != "" {
			texts = append(texts, t.Content)
		}
	}
	if latestUser != "" {
		texts = append(texts, latestUser)
	}
	if len(texts) < 2 {
		return nil
	}

	joined := strings.Join(texts, "\n")
	syms := unique(symbolRe.FindAllString(joined, -1))
	themes := detectThemes(joined)

	facts := make([]string, 0, maxFacts)
	for _, sym := range syms {
		facts = append(facts, "提到股票 "+sym)
		if len(facts) >= maxFacts {
			break
		}
	}
	for _, th := range themes {
		facts = append(facts, "关注题材 "+th)
		if len(facts) >= maxFacts {
			break
		}
	}

	topic := "投研对话"
	if len(themes) > 0 {
		topic = themes[0] + "研究"
	} else if len(syms) > 0 {
		topic = "个股 " + syms[0]
	} else {
		// 用最近一条用户话压缩标题
		u := strings.TrimSpace(latestUser)
		if u == "" && len(turns) > 0 {
			for i := len(turns) - 1; i >= 0; i-- {
				if turns[i].Role == "user" && strings.TrimSpace(turns[i].Content) != "" {
					u = turns[i].Content
					break
				}
			}
		}
		if u != "" {
			r := []rune(u)
			if len(r) > 16 {
				topic = string(r[:16]) + "…"
			} else {
				topic = u
			}
		}
	}

	questions := []string{}
	for _, t := range texts {
		if strings.Contains(t, "？") || strings.Contains(t, "?") {
			r := []rune(strings.TrimSpace(t))
			if len(r) > 40 {
				questions = append(questions, string(r[:40])+"…")
			} else if len(r) > 0 {
				questions = append(questions, string(r))
			}
		}
		if len(questions) >= 3 {
			break
		}
	}

	sum := ConversationSummary{
		ConversationID: conversationID,
		Topic:          topic,
		ImportantFacts: facts,
		OpenQuestions:  questions,
	}
	saved, err := s.UpsertSummary(sum)
	if err != nil {
		return nil
	}
	return &saved
}

func detectThemes(text string) []string {
	out := make([]string, 0, 6)
	seen := map[string]bool{}
	upper := strings.ToUpper(text)
	for _, th := range themeHints {
		hit := false
		if th == "AI" {
			hit = strings.Contains(upper, "AI") || strings.Contains(text, "人工智能")
		} else {
			hit = strings.Contains(text, th)
		}
		if hit && !seen[th] {
			seen[th] = true
			out = append(out, th)
		}
		if len(out) >= 6 {
			break
		}
	}
	return out
}

func unique(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it == "" || seen[it] {
			continue
		}
		seen[it] = true
		out = append(out, it)
	}
	return out
}
