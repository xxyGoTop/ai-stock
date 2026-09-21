package companion

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xxyGoTop/ai-stock/services/api-go/internal/watchlist"
)

type Notice struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"`
	Title     string                 `json:"title"`
	Summary   string                 `json:"summary"`
	Action    string                 `json:"action,omitempty"`
	Symbol    string                 `json:"symbol,omitempty"`
	Name      string                 `json:"name,omitempty"`
	CreatedAt string                 `json:"createdAt"`
	Read      bool                   `json:"read"`
	Items     json.RawMessage        `json:"items,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

type inboxFile struct {
	Seen    []string `json:"seen"`
	Notices []Notice `json:"notices"`
}

type notifyStore struct {
	mu   sync.Mutex
	path string
}

func newNotifyStore() *notifyStore {
	dir := "data"
	_ = os.MkdirAll(dir, 0o755)
	return &notifyStore{path: filepath.Join(dir, "notifications.json")}
}

func (s *Service) StartNotifier() {
	s.once.Do(func() {
		if s.notify == nil {
			s.notify = newNotifyStore()
		}
		interval := 120 * time.Second
		if v := strings.TrimSpace(os.Getenv("NOTIFY_INTERVAL_SEC")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 30 {
				interval = time.Duration(n) * time.Second
			}
		}
		log.Printf("notify agent every %s (weekdays 08:50-15:15 CST)", interval)
		go s.notifyLoop(interval)
	})
}

func (s *Service) notifyLoop(interval time.Duration) {
	time.Sleep(8 * time.Second)
	s.runNotifyPass()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for range tick.C {
		s.runNotifyPass()
	}
}

func (s *Service) runNotifyPass() {
	if !inNotifyWindow(time.Now()) {
		return
	}
	if n, err := s.queueAnomalyNotices(); err != nil {
		log.Printf("notify anomalies: %v", err)
	} else if n > 0 {
		log.Printf("notify: queued %d anomaly notice(s)", n)
	}
	if n, err := s.queueTodayOpsNotices(); err != nil {
		log.Printf("notify today-ops: %v", err)
	} else if n > 0 {
		log.Printf("notify: queued %d today-ops notice(s)", n)
	}
}

func inNotifyWindow(now time.Time) bool {
	t := now.In(time.FixedZone("CST", 8*3600))
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 8*60+50 && mins <= 15*60+15
}

func (s *Service) queueAnomalyNotices() (int, error) {
	if s.notify == nil {
		s.notify = newNotifyStore()
	}
	scan, err := s.ScanWatchAnomalies()
	if err != nil || scan == nil || scan.Count == 0 {
		return 0, err
	}
	fps := make([]string, 0, len(scan.Items))
	for _, it := range scan.Items {
		if it.Fingerprint != "" {
			fps = append(fps, "a|"+it.Fingerprint)
		}
	}
	fresh := s.notify.unpublished(fps)
	if len(fresh) == 0 {
		return 0, nil
	}
	keep := map[string]bool{}
	for _, fp := range fresh {
		keep[fp] = true
	}
	items := make([]Anomaly, 0, len(fresh))
	for _, it := range scan.Items {
		if keep["a|"+it.Fingerprint] {
			items = append(items, it)
		}
	}
	if len(items) == 0 {
		return 0, nil
	}
	top := items[0]
	raw, _ := json.Marshal(items)
	ok, err := s.notify.pushIfNew(fresh, Notice{
		Kind:      "anomaly",
		Title:     "你的自选股出现异动",
		Summary:   researchAnomaly(top, scan.Summary),
		Action:    "research_stock",
		Symbol:    top.Symbol,
		Name:      top.Name,
		CreatedAt: scan.AsOf,
		Items:     raw,
		Meta:      map[string]interface{}{"asOf": scan.AsOf, "count": len(items)},
	})
	if err != nil || !ok {
		return 0, err
	}
	return 1, nil
}

func researchAnomaly(top Anomaly, scanSummary string) string {
	bits := make([]string, 0, 4)
	if top.FundText != "" {
		bits = append(bits, top.FundText)
	}
	if top.LiftText != "" {
		bits = append(bits, top.LiftText)
	}
	if len(bits) == 0 && len(top.Reasons) > 0 {
		bits = append(bits, strings.Join(top.Reasons, "；"))
	}
	detail := strings.Join(bits, "，")
	if detail == "" {
		return scanSummary
	}
	return fmt.Sprintf("%s 例如 %s（%s）：%s。可继续研究成交、资金和新闻。", scanSummary, top.Name, top.Symbol, detail)
}

func (s *Service) queueTodayOpsNotices() (int, error) {
	if s.notify == nil {
		s.notify = newNotifyStore()
	}
	scan, err := s.ScanTodayOps()
	if err != nil || scan == nil || scan.Count == 0 {
		return 0, err
	}
	fps := make([]string, 0, len(scan.Items))
	for _, it := range scan.Items {
		fps = append(fps, "t|"+it.Symbol+"|"+it.PlanForDate)
	}
	fresh := s.notify.unpublished(fps)
	if len(fresh) == 0 {
		return 0, nil
	}
	keep := map[string]bool{}
	for _, fp := range fresh {
		keep[fp] = true
	}
	items := make([]watchlist.Item, 0, len(fresh))
	for _, it := range scan.Items {
		if keep["t|"+it.Symbol+"|"+it.PlanForDate] {
			items = append(items, it)
		}
	}
	if len(items) == 0 {
		return 0, nil
	}
	top := items[0]
	raw, _ := json.Marshal(items)
	ok, err := s.notify.pushIfNew(fresh, Notice{
		Kind:      "today_ops",
		Title:     "今日操作到期",
		Summary:   scan.Summary,
		Action:    "today_ops",
		Symbol:    top.Symbol,
		Name:      top.Name,
		CreatedAt: scan.AsOf,
		Items:     raw,
		Meta:      map[string]interface{}{"asOf": scan.AsOf, "date": scan.Date, "count": len(items)},
	})
	if err != nil || !ok {
		return 0, err
	}
	return 1, nil
}

func (s *Service) UnreadNotices() ([]Notice, error) {
	if s.notify == nil {
		s.notify = newNotifyStore()
	}
	return s.notify.unread()
}

func (s *Service) AckNotices(ids []string) error {
	if s.notify == nil {
		s.notify = newNotifyStore()
	}
	return s.notify.ack(ids)
}

func (st *notifyStore) load() inboxFile {
	raw, err := os.ReadFile(st.path)
	if err != nil {
		return inboxFile{}
	}
	var file inboxFile
	if json.Unmarshal(raw, &file) != nil {
		return inboxFile{}
	}
	return file
}

func (st *notifyStore) save(file inboxFile) error {
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(st.path, raw, 0o644)
}

func (st *notifyStore) unpublished(fps []string) []string {
	st.mu.Lock()
	defer st.mu.Unlock()
	file := st.load()
	seen := map[string]bool{}
	for _, x := range file.Seen {
		seen[x] = true
	}
	fresh := make([]string, 0)
	for _, fp := range fps {
		if fp == "" || seen[fp] {
			continue
		}
		fresh = append(fresh, fp)
	}
	return fresh
}

func (st *notifyStore) pushIfNew(fps []string, n Notice) (bool, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	file := st.load()
	seen := map[string]bool{}
	for _, x := range file.Seen {
		seen[x] = true
	}
	fresh := make([]string, 0, len(fps))
	for _, fp := range fps {
		if fp == "" || seen[fp] {
			continue
		}
		fresh = append(fresh, fp)
		file.Seen = append(file.Seen, fp)
	}
	if len(fresh) == 0 {
		return false, nil
	}
	if len(file.Seen) > 240 {
		file.Seen = file.Seen[len(file.Seen)-240:]
	}
	if n.ID == "" {
		n.ID = fmt.Sprintf("n-%d", time.Now().UnixNano())
	}
	file.Notices = append(file.Notices, n)
	if len(file.Notices) > 60 {
		file.Notices = file.Notices[len(file.Notices)-60:]
	}
	if err := st.save(file); err != nil {
		return false, err
	}
	return true, nil
}

func (st *notifyStore) unread() ([]Notice, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	file := st.load()
	out := make([]Notice, 0)
	for _, n := range file.Notices {
		if !n.Read {
			out = append(out, n)
		}
	}
	return out, nil
}

func (st *notifyStore) ack(ids []string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	file := st.load()
	mark := map[string]bool{}
	for _, id := range ids {
		if id != "" {
			mark[id] = true
		}
	}
	if len(mark) == 0 {
		return nil
	}
	for i := range file.Notices {
		if mark[file.Notices[i].ID] {
			file.Notices[i].Read = true
		}
	}
	return st.save(file)
}
