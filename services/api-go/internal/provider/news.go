package provider

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type NewsItem struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Source  string `json:"source"`
	Time    string `json:"time"`
	Tag     string `json:"tag"`
	URL     string `json:"url"`
	Code    string `json:"code"`
}

type HotTopic struct {
	Topic     string `json:"topic"`
	Heat      int    `json:"heat"`
	NewsCount int    `json:"newsCount"`
}

type HotBoard struct {
	Code                 string  `json:"code"`
	Name                 string  `json:"name"`
	ChangePercent        float64 `json:"changePercent"`
	Change5              float64 `json:"change5"`
	Leader               string  `json:"leader"`
	LeaderCode           string  `json:"leaderCode"`
	LeaderChangePercent  float64 `json:"leaderChangePercent"`
}

type HotFeed struct {
	News   []NewsItem `json:"news"`
	Topics []HotTopic `json:"topics"`
	Boards []HotBoard `json:"boards"`
}

var themeWords = []string{
	"算力", "光模块", "光通信", "CPO", "机器人", "人形机器人", "低空经济", "卫星",
	"创新药", "存储", "芯片", "半导体", "光刻机", "先进封装", "黄金", "有色",
	"白酒", "新能源", "光伏", "锂电", "固态电池", "军工", "重组", "并购",
	"脑机", "折叠屏", "高股息", "银行", "券商", "地产", "煤炭", "石油",
	"航运", "传媒", "游戏", "中药", "猪肉", "农业", "数据要素", "国企改革",
	"可控核聚变", "智能驾驶", "虚拟现实", "鸿蒙", "国产软件", "AI",
}

var noiseBoard = regexp.MustCompile(`连板|涨停|跌停|昨日|次新|ST|风险警示|退市|融资融券|标准普尔|富时|MSCI|沪股通|深股通|中字头|破净|预盈|预亏|高送转|参股|机构重仓|基金重仓|QFII|社保|举牌|大盘|中盘|小盘|微盘`)
var digestPrefix = regexp.MustCompile(`^【[^】]*】`)

func (b *Bundle) HotNews(limit int) ([]NewsItem, error) {
	if limit <= 0 || limit > 30 {
		limit = 15
	}
	var payload map[string]interface{}
	if err := getJSON(b.Client, "https://eminfo.eastmoney.com/pc_news/FastNews/GetImportantNewsList", "https://finance.eastmoney.com/", &payload); err != nil {
		return nil, err
	}
	raw, _ := payload["items"].([]interface{})
	type scored struct {
		item NewsItem
		rank int
		ts   float64
	}
	rows := make([]scored, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]interface{})
		title := asString(m["title"])
		if title == "" {
			continue
		}
		tag := asString(m["articleTagMarket"])
		code := asString(m["code"])
		digest := digestPrefix.ReplaceAllString(asString(m["digest"]), "")
		if len([]rune(digest)) > 120 {
			digest = string([]rune(digest)[:120])
		}
		source := asString(m["source"])
		if source == "" {
			source = "东财"
		}
		ts := asFloat(m["updateTime"])
		url := ""
		if code != "" {
			url = "https://finance.eastmoney.com/a/" + code + ".html"
		}
		rows = append(rows, scored{
			item: NewsItem{
				Title:   title,
				Summary: strings.TrimSpace(digest),
				Source:  source,
				Time:    formatNewsTime(ts),
				Tag:     tag,
				URL:     url,
				Code:    code,
			},
			rank: newsRank(tag),
			ts:   ts,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].rank != rows[j].rank {
			return rows[i].rank < rows[j].rank
		}
		return rows[i].ts > rows[j].ts
	})
	seen := map[string]bool{}
	out := make([]NewsItem, 0, limit)
	for _, r := range rows {
		if seen[r.item.Title] {
			continue
		}
		seen[r.item.Title] = true
		out = append(out, r.item)
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("hot news empty")
	}
	return out, nil
}

func (b *Bundle) HotBoards(limit int) ([]HotBoard, error) {
	if limit <= 0 || limit > 30 {
		limit = 12
	}
	query := "pn=1&pz=40&po=1&np=1&fltt=2&invt=2&fid=f3&fs=m%3A90%2Bt%3A2&fields=f12,f14,f3,f109,f128,f136,f140"
	var payload map[string]interface{}
	var last error
	for _, host := range emHosts {
		if err := getJSON(b.Client, host+"/api/qt/clist/get?"+query, "https://quote.eastmoney.com/", &payload); err != nil {
			last = err
			continue
		}
		last = nil
		break
	}
	if last != nil {
		return nil, last
	}
	data, _ := payload["data"].(map[string]interface{})
	diff, _ := data["diff"].([]interface{})
	out := make([]HotBoard, 0, limit)
	for _, it := range diff {
		m, _ := it.(map[string]interface{})
		name := asString(m["f14"])
		if name == "" || noiseBoard.MatchString(name) {
			continue
		}
		out = append(out, HotBoard{
			Code:                asString(m["f12"]),
			Name:                name,
			ChangePercent:       asFloat(m["f3"]),
			Change5:             asFloat(m["f109"]),
			Leader:              asString(m["f128"]),
			LeaderCode:          cleanLeader(asString(m["f140"])),
			LeaderChangePercent: asFloat(m["f136"]),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (b *Bundle) HotFeed(limit int) (*HotFeed, error) {
	var news []NewsItem
	var boards []HotBoard
	var newsErr, boardErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		news, newsErr = b.HotNews(limit)
	}()
	go func() {
		defer wg.Done()
		boards, boardErr = b.HotBoards(12)
	}()
	wg.Wait()
	if newsErr != nil && boardErr != nil {
		return nil, newsErr
	}
	if news == nil {
		news = []NewsItem{}
	}
	if boards == nil {
		boards = []HotBoard{}
	}
	return &HotFeed{News: news, Topics: clusterTopics(news), Boards: boards}, nil
}

func clusterTopics(news []NewsItem) []HotTopic {
	counts := map[string]int{}
	for _, n := range news {
		text := n.Title + n.Summary
		for _, w := range themeWords {
			if strings.Contains(text, w) {
				key := w
				if w == "人工智能" {
					key = "AI"
				}
				counts[key]++
			}
		}
	}
	out := make([]HotTopic, 0, len(counts))
	for topic, n := range counts {
		heat := n*22 + 40
		if heat > 99 {
			heat = 99
		}
		out = append(out, HotTopic{Topic: topic, Heat: heat, NewsCount: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Heat == out[j].Heat {
			return out[i].Topic < out[j].Topic
		}
		return out[i].Heat > out[j].Heat
	})
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

func cleanLeader(code string) string {
	if code == "" {
		return ""
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return ""
		}
	}
	return PadSymbol(code)
}

func newsRank(tag string) int {
	switch tag {
	case "SHSZ_STOCK":
		return 0
	case "HK_STOCK":
		return 1
	case "US_STOCK":
		return 2
	default:
		return 3
	}
}

func formatNewsTime(ms float64) string {
	if ms <= 0 {
		return ""
	}
	if ms < 1e12 {
		ms *= 1000
	}
	t := time.UnixMilli(int64(ms)).In(time.FixedZone("CST", 8*3600))
	return t.Format("01-02 15:04")
}
