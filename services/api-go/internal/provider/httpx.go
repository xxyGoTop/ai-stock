package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

var jsonpWrap = regexp.MustCompile(`^[a-zA-Z0-9_]+\(`)

func newClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func getBytes(client *http.Client, url, referer string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d %s", res.StatusCode, url)
	}
	return body, nil
}

func getJSON(client *http.Client, url, referer string, dest interface{}) error {
	raw, err := getBytes(client, url, referer)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(raw))
	if jsonpWrap.MatchString(text) {
		if i := strings.Index(text, "("); i >= 0 {
			text = text[i+1:]
		}
		text = strings.TrimSuffix(strings.TrimSpace(text), ";")
		text = strings.TrimSuffix(text, ")")
	}
	if err := json.Unmarshal([]byte(text), dest); err != nil {
		return fmt.Errorf("json %s: %w", url, err)
	}
	return nil
}

func asFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s)
	default:
		return strings.TrimSpace(fmt.Sprint(s))
	}
}
