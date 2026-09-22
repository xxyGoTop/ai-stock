package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const maxConversations = 30
const maxEvents = 500

type Message struct {
	ID        string          `json:"id"`
	Role      string          `json:"role"`
	Text      string          `json:"text"`
	Blocks    json.RawMessage `json:"blocks,omitempty"`
	Progress  json.RawMessage `json:"progress,omitempty"`
	Tools     json.RawMessage `json:"tools,omitempty"`
	Proactive bool            `json:"proactive,omitempty"`
}

type Conversation struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Messages     []Message       `json:"messages"`
	Workspace    json.RawMessage `json:"workspace,omitempty"`
	Briefing     json.RawMessage `json:"briefing,omitempty"`
	PhaseLabel   string          `json:"phaseLabel,omitempty"`
	Bootstrapped bool            `json:"bootstrapped"`
	CreatedAt    int64           `json:"createdAt"`
	UpdatedAt    int64           `json:"updatedAt"`
}

type ResearchEvent struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	RunID          string          `json:"runId,omitempty"`
	EventType      string          `json:"eventType"`
	ToolName       string          `json:"toolName,omitempty"`
	Status         string          `json:"status,omitempty"`
	Summary        string          `json:"summary,omitempty"`
	Input          json.RawMessage `json:"input,omitempty"`
	Output         json.RawMessage `json:"output,omitempty"`
	CreatedAt      int64           `json:"createdAt"`
}

type Snapshot struct {
	ActiveID      string         `json:"activeId"`
	Conversations []Conversation `json:"conversations"`
}

type fileData struct {
	ActiveID      string          `json:"activeId"`
	Conversations []Conversation  `json:"conversations"`
	Events        []ResearchEvent `json:"events"`
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &Store{path: filepath.Join(dir, "conversations.json")}
}

func (s *Store) load() fileData {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return fileData{Conversations: []Conversation{}, Events: []ResearchEvent{}}
	}
	var data fileData
	if json.Unmarshal(raw, &data) != nil {
		return fileData{Conversations: []Conversation{}, Events: []ResearchEvent{}}
	}
	if data.Conversations == nil {
		data.Conversations = []Conversation{}
	}
	if data.Events == nil {
		data.Events = []ResearchEvent{}
	}
	return data
}

func (s *Store) save(data fileData) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o644)
}

func (s *Store) GetSnapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	return Snapshot{ActiveID: data.ActiveID, Conversations: data.Conversations}
}

func (s *Store) PutSnapshot(snap Snapshot) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	if snap.Conversations == nil {
		snap.Conversations = []Conversation{}
	}
	if len(snap.Conversations) > maxConversations {
		sort.SliceStable(snap.Conversations, func(i, j int) bool {
			return snap.Conversations[i].UpdatedAt > snap.Conversations[j].UpdatedAt
		})
		snap.Conversations = snap.Conversations[:maxConversations]
	}
	for i := range snap.Conversations {
		if snap.Conversations[i].Messages == nil {
			snap.Conversations[i].Messages = []Message{}
		}
		if snap.Conversations[i].UpdatedAt == 0 {
			snap.Conversations[i].UpdatedAt = time.Now().UnixMilli()
		}
		if snap.Conversations[i].CreatedAt == 0 {
			snap.Conversations[i].CreatedAt = snap.Conversations[i].UpdatedAt
		}
	}
	active := snap.ActiveID
	if active == "" && len(snap.Conversations) > 0 {
		active = snap.Conversations[0].ID
	}
	if active != "" {
		found := false
		for _, c := range snap.Conversations {
			if c.ID == active {
				found = true
				break
			}
		}
		if !found && len(snap.Conversations) > 0 {
			active = snap.Conversations[0].ID
		}
	}
	data.ActiveID = active
	data.Conversations = snap.Conversations
	if err := s.save(data); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{ActiveID: data.ActiveID, Conversations: data.Conversations}, nil
}

func (s *Store) Get(id string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	for i := range data.Conversations {
		if data.Conversations[i].ID == id {
			c := data.Conversations[i]
			return &c, nil
		}
	}
	return nil, nil
}

func (s *Store) Upsert(c Conversation) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	now := time.Now().UnixMilli()
	if c.ID == "" {
		c.ID = fmt.Sprintf("%d-%d", now, len(data.Conversations)+1)
	}
	if c.Messages == nil {
		c.Messages = []Message{}
	}
	if c.CreatedAt == 0 {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	found := false
	for i := range data.Conversations {
		if data.Conversations[i].ID == c.ID {
			if data.Conversations[i].CreatedAt > 0 {
				c.CreatedAt = data.Conversations[i].CreatedAt
			}
			data.Conversations[i] = c
			found = true
			break
		}
	}
	if !found {
		data.Conversations = append([]Conversation{c}, data.Conversations...)
	}
	if len(data.Conversations) > maxConversations {
		data.Conversations = data.Conversations[:maxConversations]
	}
	if data.ActiveID == "" {
		data.ActiveID = c.ID
	}
	if err := s.save(data); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) Delete(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	next := make([]Conversation, 0, len(data.Conversations))
	for _, c := range data.Conversations {
		if c.ID != id {
			next = append(next, c)
		}
	}
	data.Conversations = next
	if data.ActiveID == id {
		data.ActiveID = ""
		if len(next) > 0 {
			data.ActiveID = next[0].ID
		}
	}
	// drop events for deleted conversation
	kept := make([]ResearchEvent, 0, len(data.Events))
	for _, e := range data.Events {
		if e.ConversationID != id {
			kept = append(kept, e)
		}
	}
	data.Events = kept
	if err := s.save(data); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{ActiveID: data.ActiveID, Conversations: data.Conversations}, nil
}

func (s *Store) SetActive(id string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	found := false
	for _, c := range data.Conversations {
		if c.ID == id {
			found = true
			break
		}
	}
	if !found && id != "" {
		return Snapshot{}, fmt.Errorf("conversation not found")
	}
	data.ActiveID = id
	if err := s.save(data); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{ActiveID: data.ActiveID, Conversations: data.Conversations}, nil
}

func (s *Store) AppendEvents(events []ResearchEvent) ([]ResearchEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	now := time.Now().UnixMilli()
	out := make([]ResearchEvent, 0, len(events))
	for _, e := range events {
		if e.ConversationID == "" || e.EventType == "" {
			continue
		}
		if e.ID == "" {
			e.ID = fmt.Sprintf("e-%d-%d", now, len(data.Events)+len(out)+1)
		}
		if e.CreatedAt == 0 {
			e.CreatedAt = now
		}
		data.Events = append(data.Events, e)
		out = append(out, e)
	}
	if len(data.Events) > maxEvents {
		data.Events = data.Events[len(data.Events)-maxEvents:]
	}
	if err := s.save(data); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListEvents(conversationID string, limit int) []ResearchEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.load()
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	out := make([]ResearchEvent, 0)
	for i := len(data.Events) - 1; i >= 0 && len(out) < limit; i-- {
		e := data.Events[i]
		if conversationID != "" && e.ConversationID != conversationID {
			continue
		}
		out = append(out, e)
	}
	return out
}
