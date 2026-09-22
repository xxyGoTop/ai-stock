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
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	ChangePercent       float64 `json:"changePercent"`
	Change5             float64 `json:"change5"`
	Leader              string  `json:"leader"`
	LeaderCode          string  `json:"leaderCode"`
	LeaderChangePercent float64 `json:"leaderChangePercent"`
}

// BoardStock 板块成分股快照。
type BoardStock struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	ChangePercent float64 `json:"changePercent"`
	Amount        float64 `json:"amount"`
	Turnover      float64 `json:"turnover"`
	VolumeRatio   float64 `json:"volumeRatio"`
	Industry      string  `json:"industry"`
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

var noiseBoard = regexp.MustCompile(`连板|涨停|跌停|昨日|次新|退市整理|融资融券|标准普尔|富时|MSCI|沪股通|深股通|中字头|破净|预盈|预亏|高送转|参股|机构重仓|基金重仓|QFII|社保|举牌|大盘|中盘|小盘|微盘`)

// knownBoardCodes 用户常搜、但未必出现在涨跌幅前列的东财板块。
var knownBoardCodes = map[string]string{
	"ST":     "BK0511",
	"*ST":    "BK0511",
	"ST股":   "BK0511",
	"ST板":   "BK0511",
	"风险警示": "BK0511",
	"风险警示板": "BK0511",
	"风险警示板块": "BK0511",
}
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
	if limit <= 0 || limit > 50 {
		limit = 30
	}
	query := "pn=1&pz=60&po=1&np=1&fltt=2&invt=2&fid=f3&fs=m%3A90%2Bt%3A2&fields=f12,f14,f3,f109,f128,f136,f140"
	hosts := append([]string{}, emHosts...)
	hosts = append(hosts,
		"https://push2his.eastmoney.com",
		"https://89.push2.eastmoney.com",
		"https://7.push2.eastmoney.com",
	)
	var payload map[string]interface{}
	var last error
	for _, host := range hosts {
		if err := getJSON(b.Client, host+"/api/qt/clist/get?"+query, "https://quote.eastmoney.com/", &payload); err != nil {
			last = err
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		diff, _ := data["diff"].([]interface{})
		if len(diff) == 0 {
			last = fmt.Errorf("hot boards empty")
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
		// 热门列表仍不推 ST/风险警示；显式搜索走 FindBoard / knownBoardCodes
		if isSpecialRiskBoard(name) {
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
	if len(out) == 0 {
		return nil, fmt.Errorf("hot boards empty")
	}
	return out, nil
}

// FindBoard 按名称在概念板(t:2)+行业板(t:3)中匹配板块。
func (b *Bundle) FindBoard(keyword string) (*HotBoard, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("empty board keyword")
	}
	aliases := boardAliases(keyword)
	if code := knownBoardCode(aliases); code != "" {
		if hit, err := b.BoardByCode(code); err == nil && hit != nil {
			return hit, nil
		}
	}
	boards, err := b.listBoards(2, 200)
	if err == nil {
		if hit := matchBoard(boards, aliases); hit != nil {
			return hit, nil
		}
	}
	ind, err2 := b.listBoards(3, 200)
	if err2 == nil {
		if hit := matchBoard(ind, aliases); hit != nil {
			return hit, nil
		}
	}
	if err != nil {
		return nil, err
	}
	if err2 != nil {
		return nil, err2
	}
	return nil, fmt.Errorf("board not found: %s", keyword)
}

func knownBoardCode(aliases []string) string {
	for _, a := range aliases {
		key := strings.ToUpper(strings.TrimSpace(a))
		if code, ok := knownBoardCodes[a]; ok {
			return code
		}
		if code, ok := knownBoardCodes[key]; ok {
			return code
		}
	}
	return ""
}

// BoardByCode 按东财板块代码取快照（如 BK0511=ST股）。
func (b *Bundle) BoardByCode(code string) (*HotBoard, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, fmt.Errorf("empty board code")
	}
	if !strings.HasPrefix(code, "BK") {
		code = "BK" + code
	}
	name := ""
	change := 0.0
	change5 := 0.0
	var payload map[string]interface{}
	secURL := "https://push2.eastmoney.com/api/qt/stock/get?secid=90." + code + "&fltt=2&fields=f57,f58,f169,f170,f109"
	if err := getJSON(b.Client, secURL, "https://quote.eastmoney.com/", &payload); err == nil {
		data, _ := payload["data"].(map[string]interface{})
		if n := asString(data["f58"]); n != "" {
			name = n
		}
		change = asFloat(data["f170"])
		change5 = asFloat(data["f109"])
		// 未带 fltt 时涨跌幅常放大 100 倍
		if change > 30 || change < -30 {
			change = change / 100
		}
	}
	if name == "" {
		if code == "BK0511" {
			name = "ST股"
		} else {
			name = code
		}
	}
	stocks, _ := b.BoardStocks(code, 5)
	out := &HotBoard{Code: code, Name: name, ChangePercent: change, Change5: change5}
	if len(stocks) > 0 {
		out.Leader = stocks[0].Name
		out.LeaderCode = stocks[0].Symbol
		out.LeaderChangePercent = stocks[0].ChangePercent
	}
	return out, nil
}

func boardAliases(keyword string) []string {
	k := strings.TrimSpace(keyword)
	k = strings.TrimSuffix(k, "板块")
	k = strings.TrimSuffix(k, "概念")
	k = strings.TrimSuffix(k, "行业")
	out := []string{k}
	extra := map[string][]string{
		"半导体": {"芯片", "集成电路", "半导体"},
		"芯片":   {"半导体", "集成电路", "芯片"},
		"新能源": {"光伏", "锂电", "储能", "新能源车", "新能源"},
		"光伏":   {"新能源", "光伏"},
		"锂电":   {"新能源", "锂电池", "锂电"},
		"创新药":  {"医药", "生物制药", "化学制药", "生物制品", "创新药", "CXO"},
		"医药":    {"创新药", "生物制药", "化学制药", "中药", "生物制品", "医疗器械", "医疗服务", "医药商业", "化学原料药", "疫苗", "医药"},
		"医疗":    {"医疗器械", "医疗服务", "医药", "化学制药"},
		"中药":    {"中药", "医药"},
		"制药":    {"化学制药", "生物制药", "医药"},
		"AI":    {"人工智能", "算力", "AI应用", "AI"},
		"人工智能": {"AI", "算力", "人工智能"},
		"算力":   {"人工智能", "AI", "算力"},
		"军工":   {"航天航空", "国防军工", "军工"},
		"消费":   {"食品饮料", "白酒", "消费电子", "消费"},
		"白酒":   {"食品饮料", "白酒"},
		"地产":   {"房地产", "房产", "地产"},
		"银行":   {"银行"},
		"券商":   {"证券", "券商"},
		"证券":   {"券商", "证券"},
		"有色":   {"有色金属", "铜", "铝", "有色"},
		"汽车":   {"汽车整车", "新能源汽车", "汽车"},
		"ST":     {"ST股", "风险警示", "ST"},
		"ST股":   {"ST", "风险警示", "ST股"},
		"*ST":    {"ST股", "风险警示", "ST"},
		"风险警示": {"ST股", "ST", "风险警示板", "风险警示"},
		"风险警示板": {"ST股", "ST", "风险警示"},
	}
	for key, vals := range extra {
		if k == key || strings.Contains(k, key) {
			out = append(out, vals...)
		}
		for _, v := range vals {
			if k == v {
				out = append(out, key)
				out = append(out, vals...)
			}
		}
	}
	seen := map[string]bool{}
	uniq := make([]string, 0, len(out))
	for _, x := range out {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		uniq = append(uniq, x)
	}
	return uniq
}

func matchBoard(boards []HotBoard, aliases []string) *HotBoard {
	// exact then contains
	for _, a := range aliases {
		for i := range boards {
			if boards[i].Name == a {
				b := boards[i]
				return &b
			}
		}
	}
	for _, a := range aliases {
		for i := range boards {
			if strings.Contains(boards[i].Name, a) || strings.Contains(a, boards[i].Name) {
				b := boards[i]
				return &b
			}
		}
	}
	// 弱匹配：别名与板块名有 2 字以上公共子串（如 医药↔化学制药 靠「药」不够；靠「制药」）
	for _, a := range aliases {
		runes := []rune(a)
		if len(runes) < 2 {
			continue
		}
		for n := len(runes); n >= 2; n-- {
			for i := 0; i+n <= len(runes); i++ {
				sub := string(runes[i : i+n])
				if len([]rune(sub)) < 2 {
					continue
				}
				for j := range boards {
					if strings.Contains(boards[j].Name, sub) {
						b := boards[j]
						return &b
					}
				}
			}
		}
	}
	return nil
}

func (b *Bundle) listBoards(boardType, limit int) ([]HotBoard, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	fs := fmt.Sprintf("m:90+t:%d", boardType)
	query := fmt.Sprintf("pn=1&pz=%d&po=1&np=1&fltt=2&invt=2&fid=f3&fs=%s&fields=f12,f14,f3,f109,f128,f136,f140",
		limit, strings.ReplaceAll(fs, ":", "%3A"))
	query = strings.ReplaceAll(query, "+", "%2B")
	hosts := append([]string{}, emHosts...)
	hosts = append(hosts, "https://push2his.eastmoney.com", "https://89.push2.eastmoney.com")
	var payload map[string]interface{}
	var last error
	for _, host := range hosts {
		if err := getJSON(b.Client, host+"/api/qt/clist/get?"+query, "https://quote.eastmoney.com/", &payload); err != nil {
			last = err
			continue
		}
		data, _ := payload["data"].(map[string]interface{})
		diff, _ := data["diff"].([]interface{})
		if len(diff) == 0 {
			last = fmt.Errorf("boards empty")
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
	out := make([]HotBoard, 0, len(diff))
	for _, it := range diff {
		m, _ := it.(map[string]interface{})
		name := asString(m["f14"])
		if name == "" || noiseBoard.MatchString(name) {
			continue
		}
		// 热门列表仍不推 ST/风险警示；显式搜索走 FindBoard / knownBoardCodes
		if isSpecialRiskBoard(name) {
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
	}
	return out, nil
}

func isSpecialRiskBoard(name string) bool {
	n := strings.ToUpper(name)
	return strings.Contains(n, "ST") || strings.Contains(name, "风险警示")
}

// ListConceptAndIndustryBoards 拉取概念板+行业板候选，供自然语言选股匹配。
func (b *Bundle) ListConceptAndIndustryBoards(limit int) ([]HotBoard, error) {
	if limit <= 0 {
		limit = 120
	}
	out := make([]HotBoard, 0, limit*2)
	seen := map[string]bool{}
	add := func(list []HotBoard, err error) {
		if err != nil {
			return
		}
		for _, x := range list {
			key := x.Code + "|" + x.Name
			if x.Name == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, x)
		}
	}
	c, err1 := b.listBoards(2, limit)
	add(c, err1)
	i, err2 := b.listBoards(3, limit)
	add(i, err2)
	if len(out) == 0 {
		if err1 != nil {
			return nil, err1
		}
		if err2 != nil {
			return nil, err2
		}
		return nil, fmt.Errorf("boards empty")
	}
	return out, nil
}

// BoardStocks 拉取板块成分股（fs=b:BKxxxx），默认按涨跌幅排序。
func (b *Bundle) BoardStocks(boardCode string, limit int) ([]BoardStock, error) {
	code := strings.ToUpper(strings.TrimSpace(boardCode))
	code = strings.TrimPrefix(code, "BK")
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("board code required")
	}
	boardCode = "BK" + code
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	fs := "b:" + boardCode
	query := fmt.Sprintf(
		"pn=1&pz=%d&po=1&np=1&fltt=2&invt=2&fid=f3&fs=%s&fields=f12,f14,f2,f3,f6,f8,f10,f100&ut=fa5fd1943c7b386f172d6893dbfba10b",
		limit, strings.ReplaceAll(strings.ReplaceAll(fs, ":", "%3A"), "+", "%2B"),
	)
	hosts := append([]string{}, emHosts...)
	hosts = append(hosts, "https://push2his.eastmoney.com", "https://89.push2.eastmoney.com", "https://7.push2.eastmoney.com")
	var payload map[string]interface{}
	var last error
	for attempt := 0; attempt < 2; attempt++ {
		for _, host := range hosts {
			if err := getJSON(b.Client, host+"/api/qt/clist/get?"+query, "https://quote.eastmoney.com/center/boardlist.html", &payload); err != nil {
				last = err
				continue
			}
			data, _ := payload["data"].(map[string]interface{})
			diff, _ := data["diff"].([]interface{})
			if len(diff) == 0 {
				last = fmt.Errorf("board stocks empty")
				continue
			}
			last = nil
			break
		}
		if last == nil {
			break
		}
	}
	if last != nil {
		return nil, last
	}
	data, _ := payload["data"].(map[string]interface{})
	diff, _ := data["diff"].([]interface{})
	out := make([]BoardStock, 0, len(diff))
	for _, it := range diff {
		m, _ := it.(map[string]interface{})
		sym := asString(m["f12"])
		name := asString(m["f14"])
		if sym == "" || name == "" {
			continue
		}
		out = append(out, BoardStock{
			Symbol:        PadSymbol(sym),
			Name:          name,
			Price:         asFloat(m["f2"]),
			ChangePercent: asFloat(m["f3"]),
			Amount:        asFloat(m["f6"]),
			Turnover:      asFloat(m["f8"]),
			VolumeRatio:   asFloat(m["f10"]),
			Industry:      asString(m["f100"]),
		})
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("board stocks empty")
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
		boards, boardErr = b.HotBoards(40)
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
