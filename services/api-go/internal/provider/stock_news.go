package provider

import (
	"fmt"
	"net/url"
	"strings"
)

type StockNewsFeed struct {
	Symbol string     `json:"symbol"`
	News   []NewsItem `json:"news"`
	Notices []NewsItem `json:"notices"`
}

func (b *Bundle) StockNews(symbol string, limit int) (*StockNewsFeed, error) {
	symbol = PadSymbol(symbol)
	if symbol == "" || symbol == "000000" {
		return nil, fmt.Errorf("symbol required")
	}
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	out := &StockNewsFeed{Symbol: symbol, News: []NewsItem{}, Notices: []NewsItem{}}
	if news, err := b.stockCmsNews(symbol, limit); err == nil {
		out.News = news
	}
	if notes, err := b.stockNotices(symbol, limit); err == nil {
		out.Notices = notes
	}
	if len(out.News) == 0 && len(out.Notices) == 0 {
		return nil, fmt.Errorf("stock news empty")
	}
	return out, nil
}

func (b *Bundle) stockCmsNews(symbol string, limit int) ([]NewsItem, error) {
	param := fmt.Sprintf(
		`{"uid":"","keyword":"%s","type":["cmsArticleWebOld"],"client":"web","clientType":"web","clientVersion":"curr","param":{"cmsArticleWebOld":{"searchScope":"default","sort":"default","pageIndex":1,"pageSize":%d,"preTag":"","postTag":""}}}`,
		symbol, limit,
	)
	u := "https://search-api-web.eastmoney.com/search/jsonp?cb=jQuery&param=" + url.QueryEscape(param)
	var payload map[string]interface{}
	if err := getJSON(b.Client, u, "https://so.eastmoney.com/", &payload); err != nil {
		return nil, err
	}
	result, _ := payload["result"].(map[string]interface{})
	raw, _ := result["cmsArticleWebOld"].([]interface{})
	out := make([]NewsItem, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]interface{})
		title := stripEm(asString(m["title"]))
		if title == "" {
			continue
		}
		link := asString(m["url"])
		if link == "" {
			link = "https://finance.eastmoney.com/a/" + asString(m["code"]) + ".html"
		}
		if strings.HasPrefix(link, "http://") {
			link = "https://" + strings.TrimPrefix(link, "http://")
		}
		out = append(out, NewsItem{
			Title:   title,
			Summary: stripEm(asString(m["content"])),
			Source:  asString(m["mediaName"]),
			Time:    asString(m["date"]),
			Tag:     "新闻",
			URL:     link,
			Code:    asString(m["code"]),
		})
	}
	return out, nil
}

func (b *Bundle) stockNotices(symbol string, limit int) ([]NewsItem, error) {
	u := fmt.Sprintf(
		"https://np-anotice-stock.eastmoney.com/api/security/ann?sr=-1&page_size=%d&page_index=1&ann_type=A&client_source=web&stock_list=%s&f_node=0&s_node=0",
		limit, symbol,
	)
	var payload map[string]interface{}
	if err := getJSON(b.Client, u, "https://data.eastmoney.com/notices/", &payload); err != nil {
		return nil, err
	}
	data, _ := payload["data"].(map[string]interface{})
	raw, _ := data["list"].([]interface{})
	out := make([]NewsItem, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]interface{})
		title := asString(m["title_ch"])
		if title == "" {
			title = asString(m["title"])
		}
		if title == "" {
			continue
		}
		art := asString(m["art_code"])
		col := ""
		if cols, ok := m["columns"].([]interface{}); ok && len(cols) > 0 {
			if c, ok := cols[0].(map[string]interface{}); ok {
				col = asString(c["column_name"])
			}
		}
		when := asString(m["notice_date"])
		if len(when) >= 10 {
			when = when[:10]
		}
		out = append(out, NewsItem{
			Title:   title,
			Summary: col,
			Source:  "巨潮/东财公告",
			Time:    when,
			Tag:     "公告",
			URL:     fmt.Sprintf("https://data.eastmoney.com/notices/detail/%s/%s.html", symbol, art),
			Code:    art,
		})
	}
	return out, nil
}

func stripEm(s string) string {
	s = strings.ReplaceAll(s, "<em>", "")
	s = strings.ReplaceAll(s, "</em>", "")
	return strings.TrimSpace(s)
}
